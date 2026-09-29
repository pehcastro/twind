package raster

import (
	"fmt"
	"image"
	"math"

	konst "github.com/twind-dev/twind/internal/konst/raster"
	"github.com/twind-dev/twind/twi/color"
)

type Rect struct{ X, Y, W, H float64 }

type Box struct {
	Rect
	Radii [4]float64
}

type Kind uint8

const (
	Fill Kind = iota
	Border
	Shadow
	Opacity
	Clip
	Pop
)

type Stop struct {
	Color color.RGBA
	At    float64
}

type BoxShadow struct {
	X, Y, Blur, Spread float64
	Inset              bool
}

type Op struct {
	Kind    Kind
	Box     Box
	Color   color.RGBA
	Stops   []Stop
	Angle   float64
	Width   float64
	Shadow  BoxShadow
	Opacity float64
}

type layer struct {
	img     *image.RGBA
	clip    image.Rectangle
	opacity float32
	group   bool
}

type band struct{ left, right, weight float64 }

type Raster struct {
	layers []layer
	groups []image.RGBA
	depth  int
	phi    [konst.PhiSteps + 1]float64
	ramp   [konst.GradientSteps + 1][4]float32
	bands  [konst.ShadowSamples]band
}

func (r *Raster) Draw(dst *image.RGBA, ops []Op, tile image.Rectangle) {
	tile = tile.Intersect(dst.Bounds())
	for y := tile.Min.Y; y < tile.Max.Y; y++ {
		clear(dst.Pix[dst.PixOffset(tile.Min.X, y):dst.PixOffset(tile.Max.X, y)])
	}
	if r.phi[konst.PhiSteps] == 0 {
		low := math.Erf(-konst.ShadowReach / math.Sqrt2)
		for i := range r.phi {
			t := (2*float64(i)/konst.PhiSteps - 1) * konst.ShadowReach
			r.phi[i] = (math.Erf(t/math.Sqrt2) - low) / (-2 * low)
		}
	}
	r.depth = 0
	r.layers = append(r.layers[:0], layer{img: dst, clip: tile, opacity: 1})
	for _, op := range ops {
		switch op.Kind {
		case Fill:
			r.fill(op)
		case Border:
			r.border(op)
		case Shadow:
			r.shadow(op)
		case Opacity:
			r.layers = append(r.layers, layer{img: r.group(tile), clip: r.top().clip, opacity: float32(op.Opacity), group: true})
		case Clip:
			b := op.Box.Rect
			clip := image.Rect(round(b.X), round(b.Y), round(b.X+b.W), round(b.Y+b.H))
			r.layers = append(r.layers, layer{img: r.top().img, clip: r.top().clip.Intersect(clip), opacity: 1})
		case Pop:
			r.pop()
		default:
			panic(fmt.Sprintf("raster: unknown op kind %d", op.Kind))
		}
	}
	for len(r.layers) > 1 {
		r.pop()
	}
}

func Mean(img *image.RGBA, area image.Rectangle) color.RGBA {
	area = area.Intersect(img.Bounds())
	var sum [4]int
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			o := img.PixOffset(x, y)
			for c := range sum {
				sum[c] += int(img.Pix[o+c])
			}
		}
	}
	if sum[3] == 0 {
		return color.RGBA{}
	}
	unmul := func(v int) uint8 { return uint8((v*math.MaxUint8 + sum[3]/2) / sum[3]) }
	n := area.Dx() * area.Dy()
	return color.RGBA{R: unmul(sum[0]), G: unmul(sum[1]), B: unmul(sum[2]), A: uint8((sum[3] + n/2) / n)}
}

func (r *Raster) group(tile image.Rectangle) *image.RGBA {
	if r.depth == len(r.groups) {
		r.groups = append(r.groups, image.RGBA{})
	}
	g := &r.groups[r.depth]
	n := tile.Dx() * tile.Dy() * 4
	if cap(g.Pix) < n {
		g.Pix = make([]uint8, n)
	}
	g.Pix, g.Stride, g.Rect = g.Pix[:n], tile.Dx()*4, tile
	clear(g.Pix)
	r.depth++
	return g
}

func (r *Raster) top() *layer { return &r.layers[len(r.layers)-1] }

func (r *Raster) pop() {
	l := *r.top()
	r.layers = r.layers[:len(r.layers)-1]
	if !l.group {
		return
	}
	r.depth--
	dst := r.top().img
	for y := l.clip.Min.Y; y < l.clip.Max.Y; y++ {
		for x := l.clip.Min.X; x < l.clip.Max.X; x++ {
			i, p := dst.PixOffset(x, y), l.img.Pix[l.img.PixOffset(x, y):]
			blend(dst.Pix[i:i+4], [4]float32{float32(p[0]), float32(p[1]), float32(p[2]), float32(p[3])}, l.opacity)
		}
	}
}

func (r *Raster) fill(op Op) {
	b := op.Box.fit()
	paint := premul(op.Color)
	var gx, gy float64
	if len(op.Stops) > 0 {
		r.gradient(op.Stops)
		sin, cos := math.Sincos(op.Angle * math.Pi / 180)
		length := math.Abs(b.W*sin) + math.Abs(b.H*cos)
		gx, gy = sin/length, -cos/length
	}
	cx, cy := b.X+b.W/2, b.Y+b.H/2
	opaque := len(op.Stops) == 0 && op.Color.A == math.MaxUint8
	solid := [4]uint8{op.Color.R, op.Color.G, op.Color.B, op.Color.A}
	top := r.top()
	area := b.pixels(0.5).Intersect(top.clip)
	for y := area.Min.Y; y < area.Max.Y; y++ {
		fy := float64(y) + 0.5
		lo, hi := b.touched(fy, area)
		fullLo, fullHi := b.full(fy, area)
		for x, i := lo, top.img.PixOffset(lo, y); x < hi; x, i = x+1, i+4 {
			fx := float64(x) + 0.5
			cov := float32(1)
			if x < fullLo || x >= fullHi {
				cov = b.cover(fx, fy)
			} else if opaque {
				run := top.img.Pix[i : i+(fullHi-x)*4]
				copy(run, solid[:])
				for n := 4; n < len(run); n *= 2 {
					copy(run[n:], run[:n])
				}
				x, i = fullHi-1, i+len(run)-4
				continue
			}
			if len(op.Stops) > 0 {
				paint = r.sample((fx-cx)*gx + (fy-cy)*gy + 0.5)
			}
			blend(top.img.Pix[i:i+4], paint, cov)
		}
	}
}

func (r *Raster) border(op Op) {
	outer := op.Box.fit()
	inner := outer.inset(op.Width)
	paint := premul(op.Color)
	top := r.top()
	area := outer.pixels(0.5).Intersect(top.clip)
	for y := area.Min.Y; y < area.Max.Y; y++ {
		fy := float64(y) + 0.5
		lo, hi := outer.touched(fy, area)
		fullLo, fullHi := outer.full(fy, area)
		holeLo, holeHi := inner.full(fy, area)
		innerLo, innerHi := inner.touched(fy, area)
		for x, i := lo, top.img.PixOffset(lo, y); x < hi; x, i = x+1, i+4 {
			if x >= holeLo && x < holeHi {
				x, i = holeHi-1, top.img.PixOffset(holeHi-1, y)
				continue
			}
			cov := float32(1)
			if x < fullLo || x >= fullHi || x >= innerLo && x < innerHi {
				fx := float64(x) + 0.5
				cov = max(outer.cover(fx, fy)-inner.cover(fx, fy), 0)
			}
			blend(top.img.Pix[i:i+4], paint, cov)
		}
	}
}

func (r *Raster) shadow(op Op) {
	box := op.Box.fit()
	s := op.Shadow
	spread := s.Spread
	if s.Inset {
		spread = -spread
	}
	shape := box.grow(spread)
	shape.X += s.X
	shape.Y += s.Y
	sigma := s.Blur * konst.SigmaPerBlur
	reach := sigma * konst.ShadowReach
	paint := premul(op.Color)
	top := r.top()
	area := shape.pixels(reach + 0.5)
	if s.Inset {
		area = box.pixels(0.5)
	}
	area = area.Intersect(top.clip)
	for y := area.Min.Y; y < area.Max.Y; y++ {
		fy := float64(y) + 0.5
		bands := r.row(shape, fy, sigma)
		left, right, innerLeft, innerRight, core := math.Inf(1), math.Inf(-1), math.Inf(-1), math.Inf(1), 0.0
		for _, b := range bands {
			left, right = min(left, b.left-reach), max(right, b.right+reach)
			innerLeft, innerRight = max(innerLeft, b.left+reach), min(innerRight, b.right-reach)
			core += b.weight
		}
		anyLo, anyHi := box.touched(fy, area)
		fullLo, fullHi := box.full(fy, area)
		lo, hi := anyLo, anyHi
		if !s.Inset {
			lo, hi = area.Min.X, area.Max.X
			if sigma > 0 {
				if len(bands) == 0 {
					continue
				}
				lo, hi = max(lo, int(math.Floor(left))), min(hi, int(math.Ceil(right)))
			}
		}
		for x, i := lo, top.img.PixOffset(lo, y); x < hi; x, i = x+1, i+4 {
			if !s.Inset && x >= fullLo && x < fullHi {
				x, i = fullHi-1, top.img.PixOffset(fullHi-1, y)
				continue
			}
			fx := float64(x) + 0.5
			mask := float32(1)
			if x >= anyLo && x < anyHi && (x < fullLo || x >= fullHi) {
				mask = box.cover(fx, fy)
				if !s.Inset {
					mask = 1 - mask
				}
			}
			var v float64
			switch {
			case sigma == 0:
				v = float64(shape.cover(fx, fy))
			case fx >= innerLeft && fx <= innerRight:
				v = core
			case fx > left && fx < right:
				for _, b := range bands {
					across := 1.0
					if d := b.right - fx; d < reach {
						across = r.cdf(d / sigma)
					}
					if d := b.left - fx; d > -reach {
						across -= r.cdf(d / sigma)
					}
					v += b.weight * across
				}
			}
			if s.Inset {
				v = 1 - v
			}
			blend(top.img.Pix[i:i+4], paint, mask*float32(v))
		}
	}
}

func (r *Raster) row(shape Box, fy, sigma float64) []band {
	bands := r.bands[:0]
	half := shape.H / 2
	dy := fy - shape.Y - half
	reach := sigma * konst.ShadowReach
	start := min(max(-reach, dy-half), dy+half)
	end := min(max(reach, dy-half), dy+half)
	if end <= start {
		return bands
	}
	step := (end - start) / konst.ShadowSamples
	below := r.cdf(start / sigma)
	for k := range konst.ShadowSamples {
		y0 := start + float64(k)*step
		above := r.cdf((y0 + step) / sigma)
		weight := above - below
		below = above
		left, right := shape.extent(fy-y0-step/2, 0)
		if n := len(bands); n > 0 && bands[n-1].left == left && bands[n-1].right == right {
			bands[n-1].weight += weight
			continue
		}
		bands = append(bands, band{left, right, weight})
	}
	return bands
}

func (r *Raster) cdf(t float64) float64 {
	u := min(max((t/konst.ShadowReach+1)*konst.PhiSteps/2, 0), konst.PhiSteps)
	i := min(int(u), konst.PhiSteps-1)
	return r.phi[i] + (u-float64(i))*(r.phi[i+1]-r.phi[i])
}

func (r *Raster) gradient(stops []Stop) {
	for i := range r.ramp {
		t := float64(i) / konst.GradientSteps
		k := 0
		for k < len(stops)-2 && stops[k+1].At < t {
			k++
		}
		from, to := stops[k], stops[min(k+1, len(stops)-1)]
		f := 0.0
		if to.At > from.At {
			f = min(max((t-from.At)/(to.At-from.At), 0), 1)
		} else if t >= to.At {
			f = 1
		}
		a1, a2 := float64(from.Color.A)/math.MaxUint8, float64(to.Color.A)/math.MaxUint8
		l1, l2 := oklab(from.Color), oklab(to.Color)
		alpha := a1 + f*(a2-a1)
		var mixed [3]float64
		for c := range mixed {
			if alpha > 0 {
				mixed[c] = (l1[c]*a1 + f*(l2[c]*a2-l1[c]*a1)) / alpha
			}
		}
		rgb, a := srgb(mixed), alpha*math.MaxUint8
		r.ramp[i] = [4]float32{float32(rgb[0] * a), float32(rgb[1] * a), float32(rgb[2] * a), float32(a)}
	}
}

func (r *Raster) sample(t float64) [4]float32 {
	u := min(max(t, 0), 1) * konst.GradientSteps
	i := min(int(u), konst.GradientSteps-1)
	f := float32(u - float64(i))
	a, b := r.ramp[i], r.ramp[i+1]
	return [4]float32{a[0] + f*(b[0]-a[0]), a[1] + f*(b[1]-a[1]), a[2] + f*(b[2]-a[2]), a[3] + f*(b[3]-a[3])}
}

func oklab(c color.RGBA) [3]float64 {
	linear := func(v uint8) float64 {
		s := float64(v) / math.MaxUint8
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	r, g, b := linear(c.R), linear(c.G), linear(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return [3]float64{
		0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		0.0259040371*l + 0.7827717662*m - 0.8086757660*s,
	}
}

func srgb(lab [3]float64) [3]float64 {
	l := lab[0] + 0.3963377774*lab[1] + 0.2158037573*lab[2]
	m := lab[0] - 0.1055613458*lab[1] - 0.0638541728*lab[2]
	s := lab[0] - 0.0894841775*lab[1] - 1.2914855480*lab[2]
	l, m, s = l*l*l, m*m*m, s*s*s
	encode := func(v float64) float64 {
		v = min(max(v, 0), 1)
		if v <= 0.0031308 {
			return 12.92 * v
		}
		return 1.055*math.Pow(v, 1/2.4) - 0.055
	}
	return [3]float64{
		encode(4.0767416621*l - 3.3077115913*m + 0.2309699292*s),
		encode(-1.2684380046*l + 2.6097574011*m - 0.3413193965*s),
		encode(-0.0041960863*l - 0.7034186147*m + 1.7076147010*s),
	}
}

func premul(c color.RGBA) [4]float32 {
	a := float32(c.A) / math.MaxUint8
	return [4]float32{float32(c.R) * a, float32(c.G) * a, float32(c.B) * a, float32(c.A)}
}

func blend(dst []uint8, src [4]float32, cov float32) {
	if src[3]*cov < 0.5 {
		return
	}
	k := 1 - src[3]*cov/math.MaxUint8
	for c, v := range src {
		dst[c] = uint8(v*cov + float32(dst[c])*k + 0.5)
	}
}

func round(v float64) int { return int(math.Round(v)) }

func (b Box) fit() Box {
	r := b.Radii
	f := 1.0
	for _, side := range [4][3]float64{{b.W, r[0], r[1]}, {b.W, r[3], r[2]}, {b.H, r[0], r[3]}, {b.H, r[1], r[2]}} {
		if sum := side[1] + side[2]; sum > side[0] {
			f = min(f, side[0]/sum)
		}
	}
	for i := range b.Radii {
		b.Radii[i] *= f
	}
	return b
}

func (b Box) inset(d float64) Box {
	out := Box{Rect: Rect{b.X + d, b.Y + d, max(b.W-2*d, 0), max(b.H-2*d, 0)}}
	for i, r := range b.Radii {
		out.Radii[i] = max(r-d, 0)
	}
	return out
}

func (b Box) grow(d float64) Box {
	out := Box{Rect: Rect{b.X - d, b.Y - d, max(b.W+2*d, 0), max(b.H+2*d, 0)}}
	for i, r := range b.Radii {
		s := d
		if d > 0 && r < d {
			k := r/d - 1
			s *= 1 + k*k*k
		}
		out.Radii[i] = max(r+s, 0)
	}
	return out
}

func (b Box) pixels(grow float64) image.Rectangle {
	return image.Rect(int(math.Floor(b.X-grow)), int(math.Floor(b.Y-grow)), int(math.Ceil(b.X+b.W+grow)), int(math.Ceil(b.Y+b.H+grow)))
}

func (b Box) extent(fy, grow float64) (left, right float64) {
	y0, y1 := b.Y-grow, b.Y+b.H+grow
	l, r, into := b.Radii[3], b.Radii[2], func(r float64) float64 { return fy - (y1 - r) }
	if fy < b.Y+b.H/2 {
		l, r, into = b.Radii[0], b.Radii[1], func(r float64) float64 { return y0 + r - fy }
	}
	corner := func(r float64) float64 {
		r = max(r+grow, 0)
		if d := into(r); d > 0 {
			return r - math.Sqrt(max(r*r-d*d, 0))
		}
		return 0
	}
	return b.X - grow + corner(l), b.X + b.W + grow - corner(r)
}

func (b Box) full(fy float64, area image.Rectangle) (int, int) {
	if fy < b.Y+0.5 || fy > b.Y+b.H-0.5 {
		return area.Min.X, area.Min.X
	}
	left, right := b.extent(fy, -0.5)
	return within(int(math.Ceil(left-0.5)), int(math.Floor(right-0.5))+1, area)
}

func (b Box) touched(fy float64, area image.Rectangle) (int, int) {
	if fy <= b.Y-0.5 || fy >= b.Y+b.H+0.5 {
		return area.Min.X, area.Min.X
	}
	left, right := b.extent(fy, 0.5)
	return within(int(math.Floor(left-0.5))+1, int(math.Ceil(right-0.5)), area)
}

func within(lo, hi int, area image.Rectangle) (int, int) {
	lo = max(lo, area.Min.X)
	return lo, max(lo, min(hi, area.Max.X))
}

func (b Box) cover(px, py float64) float32 {
	if b.W <= 0 || b.H <= 0 {
		return 0
	}
	dx, dy := px-b.X-b.W/2, py-b.Y-b.H/2
	corner := 0
	switch {
	case dx >= 0 && dy < 0:
		corner = 1
	case dx >= 0:
		corner = 2
	case dy >= 0:
		corner = 3
	}
	rad := b.Radii[corner]
	qx, qy := math.Abs(dx)-b.W/2+rad, math.Abs(dy)-b.H/2+rad
	ox, oy := max(qx, 0), max(qy, 0)
	d := math.Sqrt(ox*ox+oy*oy) + min(max(qx, qy), 0) - rad
	return float32(min(max(0.5-d, 0), 1))
}
