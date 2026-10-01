package scene

import (
	"fmt"
	"image"
	"math"
	"slices"

	rasterkonst "github.com/twind-dev/twind/internal/konst/raster"
	konst "github.com/twind-dev/twind/internal/konst/scene"
	stylekonst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/style"
)

func (f *Frame) record(n *Node, round int32, origin image.Point, layerClip image.Rectangle) (image.Rectangle, uint64, bool) {
	if !paints(n) {
		return image.Rectangle{}, 0, false
	}
	bounds := f.pixels(n.Bounds).Sub(origin)
	if bounds.Empty() {
		return bounds, 0, false
	}
	start, plain := len(f.ops), n != f.root && n.Gradient.Kind != style.GradientLinear && n.Turn == 0
	var st *stamp
	hit, visual := false, bounds
	if len(n.Shadows) > 0 {
		visual = f.cast(bounds, n.Shadows)
	}
	if plain {
		st = f.stamps.find(n, bounds.Size())
		hit = st.holds(&f.stamps, n, bounds.Size(), visual.Sub(bounds.Min))
	}
	if hit {
		x, y := float64(bounds.Min.X), float64(bounds.Min.Y)
		for i := st.from; i < st.to; i++ {
			f.ops = append(f.ops, f.stamps.ops[i])
			op := &f.ops[len(f.ops)-1]
			op.Box.X += x
			op.Box.Y += y
		}
	} else if visual = f.shapes(n, bounds, visual, origin); len(f.ops) == start {
		return visual, 0, false
	}
	drawn, ok := len(f.ops), true
	if clip := f.pixels(n.Clip); clip != layerClip || round != 0 {
		visual, ok = f.clip(start, visual, clip, layerClip, origin, round)
	}
	switch {
	case !ok:
		return visual, 0, false
	case hit && len(f.ops) == drawn:
		return visual, st.look, true
	}
	look := f.looks.look(f.ops[start:], visual)
	if plain && len(f.ops) == drawn {
		f.stamps.keep(st, n, f.ops[start:], bounds, visual, look)
	}
	return visual, look, true
}

func (f *Frame) cast(bounds image.Rectangle, shadows []style.Shadow) image.Rectangle {
	visual, shape := bounds, rect(bounds)
	for i := len(shadows) - 1; i >= 0; i-- {
		if s := &shadows[i]; shows(s.Color) {
			cast := f.shadow(*s)
			reach := cast.Blur*rasterkonst.SigmaPerBlur*rasterkonst.ShadowReach + cast.Spread
			visual = visual.Union(image.Rect(
				int(math.Floor(shape.X+cast.X-reach)), int(math.Floor(shape.Y+cast.Y-reach)),
				int(math.Ceil(shape.X+shape.W+cast.X+reach)), int(math.Ceil(shape.Y+shape.H+cast.Y+reach)),
			))
		}
	}
	return visual
}

func paints(n *Node) bool {
	return bordered(&n.Border) || shows(n.Background) || len(n.Shadows)+len(n.InsetShadows) > 0 || n.Gradient.Kind == style.GradientLinear
}

func bordered(b *Border) bool { return b.Style != style.BorderNone && shows(b.Color) }

func (f *Frame) shapes(n *Node, bounds, visual image.Rectangle, origin image.Point) image.Rectangle {
	edges, start, bordered := &n.Border, len(f.ops), bordered(&n.Border)
	shape := rect(bounds)
	outer := raster.Box{Rect: shape, Radii: f.radii(n.Border.Radius)}
	for i := len(n.Shadows) - 1; i >= 0; i-- {
		if s := &n.Shadows[i]; shows(s.Color) {
			f.put(raster.Shadow, outer, s.Color.RGBA).Shadow = f.shadow(*s)
		}
	}
	if shows(n.Background) {
		if n == f.root {
			canvas := f.pixels(f.screen).Sub(origin)
			visual = visual.Union(canvas)
			f.put(raster.Fill, raster.Box{Rect: rect(canvas)}, n.Background.RGBA)
		} else {
			f.put(raster.Fill, outer, n.Background.RGBA)
		}
	}
	if n.Gradient.Kind == style.GradientLinear {
		f.ops = append(f.ops, GradientFill(n.Gradient, outer))
	}
	ring := bordered && edges.Top && edges.Right && edges.Bottom && edges.Left
	for i := len(n.InsetShadows) - 1; i >= 0; i-- {
		if s := &n.InsetShadows[i]; shows(s.Color) {
			cast := f.shadow(*s)
			cast.Inset = true
			inner := outer
			if ring {
				inner = outer.Inset(konst.BorderPixels)
			}
			f.put(raster.Shadow, inner, s.Color.RGBA).Shadow = cast
		}
	}
	switch {
	case ring:
		op := f.put(raster.Border, outer, edges.Color.RGBA)
		op.Width, op.Dash = konst.BorderPixels, dash(edges.Style)
	case bordered:
		s := shape
		for _, side := range [...]struct {
			on   bool
			line raster.Rect
		}{
			{edges.Top, raster.Rect{X: s.X, Y: s.Y, W: s.W, H: konst.BorderPixels}},
			{edges.Right, raster.Rect{X: s.X + s.W - konst.BorderPixels, Y: s.Y, W: konst.BorderPixels, H: s.H}},
			{edges.Bottom, raster.Rect{X: s.X, Y: s.Y + s.H - konst.BorderPixels, W: s.W, H: konst.BorderPixels}},
			{edges.Left, raster.Rect{X: s.X, Y: s.Y, W: konst.BorderPixels, H: s.H}},
		} {
			if side.on {
				f.put(raster.Fill, raster.Box{Rect: side.line}, edges.Color.RGBA).Dash = dash(edges.Style)
			}
		}
	}
	if len(f.ops) == start || n.Turn == 0 {
		return visual
	}
	sin, cos := math.Sincos(n.Turn * 2 * math.Pi)
	cx, cy := shape.X+shape.W/2, shape.Y+shape.H/2
	x, y := float64(visual.Min.X+visual.Max.X)/2-cx, float64(visual.Min.Y+visual.Max.Y)/2-cy
	x, y = cx+x*cos-y*sin, cy+x*sin+y*cos
	w, h := float64(visual.Dx())/2, float64(visual.Dy())/2
	reachX, reachY := w*math.Abs(cos)+h*math.Abs(sin), w*math.Abs(sin)+h*math.Abs(cos)
	return image.Rect(int(math.Floor(x-reachX)), int(math.Floor(y-reachY)), int(math.Ceil(x+reachX)), int(math.Ceil(y+reachY)))
}

func (f *Frame) clip(start int, visual, clip, layerClip image.Rectangle, origin image.Point, round int32) (image.Rectangle, bool) {
	if clip == layerClip && round == 0 {
		return visual, true
	}
	pushed := len(f.ops)
	if clip != layerClip {
		if visual = visual.Intersect(clip.Sub(origin)); visual.Empty() {
			return visual, false
		}
		f.put(raster.Clip, raster.Box{Rect: rect(clip.Sub(origin))}, color.RGBA{})
	}
	seen := visual.Intersect(clip.Sub(origin))
	for ; round != 0; round = f.entries[round].round {
		n := f.entries[round].node
		outer, shape := f.pixels(n.Bounds), f.pixels(n.Padding)
		inset := max(shape.Min.X-outer.Min.X, shape.Min.Y-outer.Min.Y, outer.Max.X-shape.Max.X, outer.Max.Y-shape.Max.Y)
		r := f.radii(n.Border.Radius)
		for i := range r {
			r[i] = max(r[i]-float64(inset), 0)
		}
		if r == [4]float64{} || !clip.In(shape) {
			continue
		}
		shape = shape.Sub(origin)
		reach := int(math.Ceil(min(max(r[0], r[1], r[2], r[3]), float64(shape.Dx())/2, float64(shape.Dy())/2)))
		if seen.Min.X >= shape.Min.X+reach && seen.Max.X <= shape.Max.X-reach || seen.Min.Y >= shape.Min.Y+reach && seen.Max.Y <= shape.Max.Y-reach {
			continue
		}
		f.put(raster.Clip, raster.Box{Rect: rect(shape), Radii: r}, color.RGBA{})
	}
	pops := len(f.ops) - pushed
	slices.Reverse(f.ops[start:pushed])
	slices.Reverse(f.ops[pushed:])
	slices.Reverse(f.ops[start:])
	for range pops {
		f.put(raster.Pop, raster.Box{}, color.RGBA{})
	}
	return visual, true
}

func (f *Frame) put(kind raster.Kind, at raster.Box, c color.RGBA) *raster.Op {
	f.ops = append(f.ops, raster.Op{})
	op := &f.ops[len(f.ops)-1]
	op.Kind, op.Box, op.Color = kind, at, c
	return op
}

func dash(s style.BorderStyle) raster.Dash {
	switch s {
	case style.BorderNone, style.BorderSingle, style.BorderDouble:
		return raster.Solid
	case style.BorderDashed:
		return raster.Dashed
	case style.BorderDotted:
		return raster.Dotted
	}
	panic(fmt.Sprintf("scene: unknown border style %d", s))
}

func (f *Frame) thumb(n *Node, origin image.Point, layerClip image.Rectangle) (image.Rectangle, uint64, bool) {
	from, to, ok := n.Thumb(f.cell.Y)
	if !ok {
		return image.Rectangle{}, 0, false
	}
	inset := int(math.Round(konst.ThumbInsetCell * float64(f.cell.X)))
	width := int(math.Round(konst.ThumbWidthCell * float64(f.cell.X)))
	view := f.pixels(n.Padding)
	visual := image.Rect(view.Max.X-inset-width, view.Min.Y+from+inset, view.Max.X-inset, view.Min.Y+to-inset).Sub(origin)
	c := color.RGBA{R: konst.ThumbGrey, G: konst.ThumbGrey, B: konst.ThumbGrey, A: math.MaxUint8}
	if n.Foreground.Kind == color.Literal {
		c = n.Foreground.RGBA
	}
	c.A = uint8(float64(c.A) * konst.ThumbAlpha)
	start := len(f.ops)
	half := float64(width) / 2
	f.put(raster.Fill, raster.Box{Rect: rect(visual), Radii: [4]float64{half, half, half, half}}, c)
	if visual, ok = f.clip(start, visual, f.pixels(n.Clip).Intersect(view), layerClip, origin, 0); !ok {
		return visual, 0, false
	}
	return visual, f.looks.look(f.ops[start:], visual), true
}

func GradientFill(g style.Gradient, shape raster.Box) raster.Op {
	stops := []raster.Stop{{Color: g.From.Color.RGBA, At: g.From.Position}}
	if g.HasVia {
		stops = append(stops, raster.Stop{Color: g.Via.Color.RGBA, At: g.Via.Position})
	}
	stops = append(stops, raster.Stop{Color: g.To.Color.RGBA, At: g.To.Position})
	return raster.Op{Kind: raster.Fill, Box: shape, Stops: stops, Angle: angle(g.GradientLine, shape.Rect)}
}

func (f *Frame) shadow(s style.Shadow) raster.BoxShadow {
	px := float64(f.cell.Y) / stylekonst.RemPixels
	return raster.BoxShadow{X: float64(s.X) * px, Y: float64(s.Y) * px, Blur: float64(s.Blur) * px, Spread: float64(s.Spread) * px}
}

func (f *Frame) radii(r style.Radius) [4]float64 {
	if r == style.RadiusNone {
		return [4]float64{}
	}
	return f.corners(r)
}

func (f *Frame) corners(r style.Radius) (out [4]float64) {
	for c := range out {
		out[c] = f.radius(r.At(style.Corner(c)))
	}
	return out
}

func (f *Frame) radius(r style.Radius) float64 {
	rem := float64(f.cell.Y)
	switch r {
	case style.RadiusNone:
		return 0
	case style.RadiusSm:
		return konst.RadiusSmRem * rem
	case style.RadiusMd:
		return konst.RadiusMdRem * rem
	case style.RadiusLg:
		return konst.RadiusLgRem * rem
	case style.RadiusFull:
		return konst.RadiusFullPixels
	}
	panic(fmt.Sprintf("scene: unknown radius %d", r))
}

func angle(g style.GradientLine, box raster.Rect) float64 {
	corner := math.Atan2(box.H, box.W) * konst.DegreesToBottom / math.Pi
	switch g.Direction {
	case style.ToTop:
		return 0
	case style.ToRight:
		return konst.DegreesToRight
	case style.ToBottom:
		return konst.DegreesToBottom
	case style.ToLeft:
		return konst.DegreesToLeft
	case style.ToTopRight:
		return corner
	case style.ToBottomRight:
		return konst.DegreesToBottom - corner
	case style.ToBottomLeft:
		return konst.DegreesToBottom + corner
	case style.ToTopLeft:
		return konst.DegreesTurn - corner
	case style.GradientAngle:
		return g.Angle
	}
	panic(fmt.Sprintf("scene: unknown gradient direction %d", g.Direction))
}

func (f *Frame) pixels(r layout.Rect) image.Rectangle {
	return image.Rect(r.X*f.cell.X, r.Y*f.cell.Y, (r.X+r.W)*f.cell.X, (r.Y+r.H)*f.cell.Y)
}

func rect(r image.Rectangle) raster.Rect {
	return raster.Rect{X: float64(r.Min.X), Y: float64(r.Min.Y), W: float64(r.Dx()), H: float64(r.Dy())}
}

func shows(c color.Color) bool { return c.Kind == color.Literal && c.RGBA.A > 0 }
