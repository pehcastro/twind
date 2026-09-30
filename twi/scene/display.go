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

func (f *Frame) record(n *Node, round *clipper, origin image.Point, layerClip image.Rectangle) (image.Rectangle, bool) {
	start := len(f.ops)
	bounds := f.pixels(n.Bounds).Sub(origin)
	if bounds.Empty() {
		return bounds, false
	}
	r := f.radius(n.Border.Radius)
	shape := raster.Box{Rect: rect(bounds), Radii: [4]float64{r, r, r, r}}
	visual := bounds
	for i := len(n.Shadows) - 1; i >= 0; i-- {
		s := n.Shadows[i]
		if !shows(s.Color) {
			continue
		}
		cast := f.shadow(s)
		f.ops = append(f.ops, raster.Op{Kind: raster.Shadow, Box: shape, Color: s.Color.RGBA, Shadow: cast})
		reach := cast.Blur*rasterkonst.SigmaPerBlur*rasterkonst.ShadowReach + cast.Spread
		visual = visual.Union(image.Rect(
			int(math.Floor(shape.X+cast.X-reach)), int(math.Floor(shape.Y+cast.Y-reach)),
			int(math.Ceil(shape.X+shape.W+cast.X+reach)), int(math.Ceil(shape.Y+shape.H+cast.Y+reach)),
		))
	}
	if shows(n.Background) {
		fill := shape
		if n == f.root {
			canvas := f.pixels(f.screen).Sub(origin)
			fill, visual = raster.Box{Rect: rect(canvas)}, visual.Union(canvas)
		}
		f.ops = append(f.ops, raster.Op{Kind: raster.Fill, Box: fill, Color: n.Background.RGBA})
	}
	if n.Gradient.Kind == style.GradientLinear {
		f.ops = append(f.ops, GradientFill(n.Gradient, shape))
	}
	edges := n.Border
	bordered := edges.Style != style.BorderNone && shows(edges.Color)
	ring := bordered && edges.Top && edges.Right && edges.Bottom && edges.Left
	inner := shape
	if ring {
		inner.Rect = raster.Rect{X: shape.X + konst.BorderPixels, Y: shape.Y + konst.BorderPixels, W: shape.W - 2*konst.BorderPixels, H: shape.H - 2*konst.BorderPixels}
		for i := range inner.Radii {
			inner.Radii[i] = max(r-konst.BorderPixels, 0)
		}
	}
	for i := len(n.InsetShadows) - 1; i >= 0; i-- {
		if s := n.InsetShadows[i]; shows(s.Color) {
			cast := f.shadow(s)
			cast.Inset = true
			f.ops = append(f.ops, raster.Op{Kind: raster.Shadow, Box: inner, Color: s.Color.RGBA, Shadow: cast})
		}
	}
	switch {
	case ring:
		f.ops = append(f.ops, raster.Op{Kind: raster.Border, Box: shape, Color: edges.Color.RGBA, Width: konst.BorderPixels, Dash: dash(edges.Style)})
	case bordered:
		s := shape.Rect
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
				f.ops = append(f.ops, raster.Op{Kind: raster.Fill, Box: raster.Box{Rect: side.line}, Color: edges.Color.RGBA, Dash: dash(edges.Style)})
			}
		}
	}
	if len(f.ops) == start {
		return visual, false
	}
	return f.clip(start, visual, f.pixels(n.Clip), layerClip, origin, round)
}

func (f *Frame) clip(start int, visual, clip, layerClip image.Rectangle, origin image.Point, round *clipper) (image.Rectangle, bool) {
	pushed := len(f.ops)
	if clip != layerClip {
		if visual = visual.Intersect(clip.Sub(origin)); visual.Empty() {
			return visual, false
		}
		f.ops = append(f.ops, raster.Op{Kind: raster.Clip, Box: raster.Box{Rect: rect(clip.Sub(origin))}})
	}
	seen := visual.Intersect(clip.Sub(origin))
	for ; round != nil; round = round.up {
		n := round.node
		outer, shape := f.pixels(n.Bounds), f.pixels(n.Padding)
		inset := max(shape.Min.X-outer.Min.X, shape.Min.Y-outer.Min.Y, outer.Max.X-shape.Max.X, outer.Max.Y-shape.Max.Y)
		r := f.radius(n.Border.Radius) - float64(inset)
		if r <= 0 || !clip.In(shape) {
			continue
		}
		shape = shape.Sub(origin)
		reach := int(math.Ceil(min(r, float64(shape.Dx())/2, float64(shape.Dy())/2)))
		if seen.Min.X >= shape.Min.X+reach && seen.Max.X <= shape.Max.X-reach || seen.Min.Y >= shape.Min.Y+reach && seen.Max.Y <= shape.Max.Y-reach {
			continue
		}
		f.ops = append(f.ops, raster.Op{Kind: raster.Clip, Box: raster.Box{Rect: rect(shape), Radii: [4]float64{r, r, r, r}}})
	}
	pops := len(f.ops) - pushed
	slices.Reverse(f.ops[start:pushed])
	slices.Reverse(f.ops[pushed:])
	slices.Reverse(f.ops[start:])
	for range pops {
		f.ops = append(f.ops, raster.Op{Kind: raster.Pop})
	}
	return visual, true
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

func (f *Frame) thumb(n *Node, origin image.Point, layerClip image.Rectangle) (image.Rectangle, bool) {
	from, to, ok := n.Thumb(f.cell.Y)
	if !ok {
		return image.Rectangle{}, false
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
	radius := float64(width) / 2
	start := len(f.ops)
	f.ops = append(f.ops, raster.Op{Kind: raster.Fill, Box: raster.Box{Rect: rect(visual), Radii: [4]float64{radius, radius, radius, radius}}, Color: c})
	return f.clip(start, visual, f.pixels(n.Clip).Intersect(view), layerClip, origin, nil)
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
