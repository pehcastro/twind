package raster

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"math"
	"slices"

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
	Dash    Dash
	Shadow  BoxShadow
	Opacity float64
}

type layer struct {
	img     *image.RGBA
	clip    image.Rectangle
	opacity float32
	mask    Box
	group   bool
}

type memo struct {
	along, t float64
	cov      float32
	in, out  [4]uint8
}

type Raster struct {
	layers   []layer
	groups   []image.RGBA
	depth    int
	phi      [konst.PhiSteps + 1]float64
	ramp     [konst.GradientSteps + 1][4]float32
	stops    []Stop
	memo     []memo
	solid    []uint8
	rows     repeat
	first    int
	same     []bool
	frames   []frame
	hide     image.Rectangle
	hider    int
	index    int
	outlines []outline
	shapes   int
	left     []float64
	right    []float64
	row      []float64
	flatLo   int
	flatHi   int
	tables   []float64
	strips   []strip
	corners  [4]corner
	sides    [2]side
}

func (r *Raster) Draw(dst *image.RGBA, ops []Op, tile image.Rectangle) {
	tile = tile.Intersect(dst.Bounds())
	r.plan(ops, tile)
	r.depth = 0
	if !covers(ops, tile) {
		for y := tile.Min.Y; y < tile.Max.Y; y++ {
			if r.skip(y) {
				continue
			}
			for _, run := range r.open(y, tile.Min.X, tile.Max.X) {
				clear(dst.Pix[dst.PixOffset(run[0], y):dst.PixOffset(run[1], y)])
			}
		}
	}
	if r.phi[konst.PhiSteps] == 0 {
		low := math.Erf(-konst.ShadowReach / math.Sqrt2)
		for i := range r.phi {
			t := (2*float64(i)/konst.PhiSteps - 1) * konst.ShadowReach
			r.phi[i] = (math.Erf(t/math.Sqrt2) - low) / (-2 * low)
		}
	}
	r.layers = append(r.layers[:0], layer{img: dst, clip: tile, opacity: 1})
	for i, op := range ops {
		r.index = i
		switch op.Kind {
		case Fill:
			r.fill(op)
		case Border:
			r.border(op)
		case Shadow:
			r.shadow(op)
		case Opacity:
			c := r.top().clip
			whole := Box{Rect: Rect{float64(c.Min.X), float64(c.Min.Y), float64(c.Dx()), float64(c.Dy())}}
			r.layers = append(r.layers, layer{img: r.group(tile, c), clip: c, opacity: float32(op.Opacity), mask: whole, group: true})
		case Clip:
			b := op.Box
			if b.Radii == [4]float64{} {
				clip := image.Rect(round(b.X), round(b.Y), round(b.X+b.W), round(b.Y+b.H))
				r.layers = append(r.layers, layer{img: r.top().img, clip: r.top().clip.Intersect(clip), opacity: 1})
				continue
			}
			c := r.top().clip.Intersect(b.pixels(0))
			r.layers = append(r.layers, layer{img: r.group(tile, c), clip: c, opacity: 1, mask: b.fit(), group: true})
		case Pop:
			r.pop()
		default:
			panic(fmt.Sprintf("raster: unknown op kind %d", op.Kind))
		}
	}
	for len(r.layers) > 1 {
		r.pop()
	}
	var src []uint8
	for y := tile.Min.Y; y < tile.Max.Y; y++ {
		row := dst.Pix[dst.PixOffset(tile.Min.X, y):dst.PixOffset(tile.Max.X, y)]
		if !r.skip(y) {
			src = row
			continue
		}
		copy(row, src)
	}
}

func covers(ops []Op, tile image.Rectangle) bool {
	if len(ops) == 0 {
		return false
	}
	op := ops[0]
	b := op.Box
	return op.Kind == Fill && len(op.Stops) == 0 && op.Color.A == math.MaxUint8 && b.Radii == [4]float64{} &&
		b.X <= float64(tile.Min.X) && b.Y <= float64(tile.Min.Y) && b.X+b.W >= float64(tile.Max.X) && b.Y+b.H >= float64(tile.Max.Y)
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

func (r *Raster) group(tile, area image.Rectangle) *image.RGBA {
	if r.depth == len(r.groups) {
		r.groups = append(r.groups, image.RGBA{})
	}
	g := &r.groups[r.depth]
	n := tile.Dx() * tile.Dy() * 4
	if cap(g.Pix) < n {
		g.Pix = make([]uint8, n)
	}
	g.Pix, g.Stride, g.Rect = g.Pix[:n], tile.Dx()*4, tile
	for y := area.Min.Y; y < area.Max.Y; y++ {
		clear(g.Pix[g.PixOffset(area.Min.X, y):g.PixOffset(area.Max.X, y)])
	}
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
	dst, mask := r.top().img, r.outline(l.mask)
	for y := l.clip.Min.Y; y < l.clip.Max.Y; y++ {
		if r.skip(y) {
			continue
		}
		lo, hi := mask.touched(y, l.clip)
		fullLo, fullHi := mask.full(y, l.clip)
		for x := lo; x < hi; x++ {
			cov := l.opacity
			if x < fullLo || x >= fullHi {
				cov *= mask.cover(x, y)
			}
			i, p := dst.PixOffset(x, y), l.img.Pix[l.img.PixOffset(x, y):]
			blend(dst.Pix[i:i+4], [4]float32{float32(p[0]), float32(p[1]), float32(p[2]), float32(p[3])}, cov)
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
	var dash pattern
	across := b.W < b.H
	if op.Dash != Solid {
		long, gap := dashes(op.Dash, min(b.W, b.H))
		dash = loop(long, gap, max(b.W, b.H)+gap)
	}
	uniform := len(op.Stops) == 0 && op.Dash == Solid
	opaque := uniform && op.Color.A == math.MaxUint8
	top := r.top()
	area := b.pixels(0.5).Intersect(top.clip)
	if opaque && !area.Empty() {
		r.solid = resize(r.solid, 4*area.Dx())
		r.solid[0], r.solid[1], r.solid[2], r.solid[3] = op.Color.R, op.Color.G, op.Color.B, op.Color.A
		for n := 4; n < len(r.solid); n *= 2 {
			copy(r.solid[n:], r.solid[:n])
		}
	}
	if len(op.Stops) > 0 {
		r.memo = resize(r.memo, area.Dx())
		for i := range r.memo {
			r.memo[i] = memo{along: (float64(area.Min.X+i) + 0.5 - cx) * gx, t: math.NaN()}
		}
	}
	o := r.outline(b)
	bandLo, bandHi := b.straight(0)
	var lo, hi, fullLo, fullHi int
	banded := false
	for y := area.Min.Y; y < area.Max.Y; y++ {
		if r.skip(y) {
			continue
		}
		fy := float64(y) + 0.5
		band := fy >= bandLo && fy <= bandHi
		if !banded || !band {
			lo, hi = o.touched(y, area)
			fullLo, fullHi = o.full(y, area)
		}
		banded = band
		down := (fy - cy) * gy
		for x, i := lo, top.img.PixOffset(lo, y); x < hi; x, i = x+1, i+4 {
			fx := float64(x) + 0.5
			cov := float32(1)
			switch {
			case x < fullLo || x >= fullHi:
				cov = o.cover(x, y)
			case uniform:
				for _, run := range r.open(y, x, fullHi) {
					pix := top.img.Pix[top.img.PixOffset(run[0], y):top.img.PixOffset(run[1], y)]
					if opaque {
						copy(pix, r.solid)
					} else {
						flood(pix, paint, 1)
					}
				}
				x, i = fullHi-1, top.img.PixOffset(fullHi-1, y)
				continue
			case op.Dash == Solid:
				r.shade(top.img.Pix[i:top.img.PixOffset(fullHi, y)], r.memo[x-area.Min.X:fullHi-area.Min.X], down, 1)
				x, i = fullHi-1, top.img.PixOffset(fullHi-1, y)
				continue
			}
			switch {
			case op.Dash == Solid:
			case across:
				cov *= dash.cover(fy - b.Y)
			default:
				cov *= dash.cover(fx - b.X)
			}
			px := top.img.Pix[i : i+4 : i+4]
			if len(op.Stops) == 0 {
				blend(px, paint, cov)
				continue
			}
			r.shade(px, r.memo[x-area.Min.X:][:1], down, cov)
		}
	}
}

func (r *Raster) shade(pix []uint8, memos []memo, down float64, cov float32) {
	for k := range memos {
		m, px := &memos[k], (*[4]uint8)(pix[4*k:])
		if t := m.along + down + 0.5; t != m.t || cov != m.cov || *px != m.in {
			m.t, m.cov, m.in = t, cov, *px
			blend(px[:], r.sample(t), cov)
			m.out = *px
		}
		*px = m.out
	}
}

func (r *Raster) border(op Op) {
	outer := op.Box.fit()
	inner := outer.inset(op.Width)
	paint := premul(op.Color)
	top := r.top()
	area := outer.pixels(0.5).Intersect(top.clip)
	out, in := r.outline(outer), r.outline(inner)
	if op.Dash != Solid {
		r.dashedBorder(op, out, in, area)
		return
	}
	bandLo, bandHi := outer.straight(0)
	innerLo, innerHi := inner.straight(0)
	bandLo, bandHi = max(bandLo, innerLo), min(bandHi, innerHi)
	r.rows.y = math.MinInt
	for y := area.Min.Y; y < area.Max.Y; y++ {
		if r.skip(y) {
			continue
		}
		fy := float64(y) + 0.5
		band := fy >= bandLo && fy <= bandHi
		if band && r.rows.again(top.img, y) {
			continue
		}
		lo, hi := out.touched(y, area)
		fullLo, fullHi := out.full(y, area)
		holeLo, holeHi := in.full(y, area)
		edgeLo, edgeHi := in.touched(y, area)
		spans := [2][2]int{{lo, max(lo, min(holeLo, hi))}, {max(lo, min(holeHi, hi)), hi}}
		if band {
			r.rows.keep(top.img, y, spans)
		}
		solid := [2][2]int{{max(spans[0][0], fullLo), min(spans[0][1], fullHi, edgeLo)}, {max(spans[1][0], fullLo, edgeHi), min(spans[1][1], fullHi)}}
		for n, sp := range spans {
			a := min(solid[n][0], sp[1])
			b := max(a, solid[n][1])
			flood(top.img.Pix[top.img.PixOffset(a, y):top.img.PixOffset(b, y)], paint, 1)
			for _, p := range [2][2]int{{sp[0], a}, {b, sp[1]}} {
				for x, i := p[0], top.img.PixOffset(p[0], y); x < p[1]; x, i = x+1, i+4 {
					outside, inside := float32(1), float32(0)
					if x < fullLo || x >= fullHi {
						outside = out.cover(x, y)
					}
					if x >= edgeLo && x < edgeHi {
						inside = in.cover(x, y)
					}
					blend(top.img.Pix[i:i+4], paint, max(outside-inside, 0))
				}
			}
		}
	}
}

func flood(pix []uint8, src [4]float32, cov float32) {
	if len(pix) > 4 && bytes.Equal(pix[4:], pix[:len(pix)-4]) {
		blend(pix[:4], src, cov)
		for n := 4; n < len(pix); n *= 2 {
			copy(pix[n:], pix[:n])
		}
		return
	}
	var in, out uint32
	for i := 0; i < len(pix); i += 4 {
		px := pix[i : i+4 : i+4]
		if v := binary.LittleEndian.Uint32(px); i == 0 || v != in {
			in = v
			blend(px, src, cov)
			out = binary.LittleEndian.Uint32(px)
			continue
		}
		binary.LittleEndian.PutUint32(px, out)
	}
}

type repeat struct {
	y     int
	spans [2][2]int
	kept  []uint8
}

func (p *repeat) again(img *image.RGBA, y int) bool {
	if p.y != y-1 {
		return false
	}
	n := 0
	for _, sp := range p.spans {
		row := img.Pix[img.PixOffset(sp[0], y):img.PixOffset(sp[1], y)]
		if !bytes.Equal(row, p.kept[n:n+len(row)]) {
			return false
		}
		n += len(row)
	}
	for _, sp := range p.spans {
		copy(img.Pix[img.PixOffset(sp[0], y):img.PixOffset(sp[1], y)], img.Pix[img.PixOffset(sp[0], y-1):img.PixOffset(sp[1], y-1)])
	}
	p.y = y
	return true
}

func (p *repeat) keep(img *image.RGBA, y int, spans [2][2]int) {
	p.y, p.spans, p.kept = y, spans, p.kept[:0]
	for _, sp := range spans {
		p.kept = append(p.kept, img.Pix[img.PixOffset(sp[0], y):img.PixOffset(sp[1], y)]...)
	}
}

func resize[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	return s[:n]
}

func (r *Raster) gradient(stops []Stop) {
	if slices.Equal(stops, r.stops) {
		return
	}
	r.stops = append(r.stops[:0], stops...)
	var l1, l2 [3]float64
	pair := -1
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
		if k != pair {
			l1, l2, pair = oklab(from.Color), oklab(to.Color), k
		}
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
		return 1.055*math.Exp(1/2.4*math.Log(v)) - 0.055
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
	a := src[3] * cov
	if a < 0.5 {
		return
	}
	k := 1 - a/math.MaxUint8
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

func (b Box) straight(pad float64) (float64, float64) {
	return b.Y + max(b.Radii[0], b.Radii[1], 0.5) + pad, b.Y + b.H - max(b.Radii[2], b.Radii[3], 0.5) - pad
}

func (b Box) pixels(grow float64) image.Rectangle {
	return image.Rect(int(math.Floor(b.X-grow)), int(math.Floor(b.Y-grow)), int(math.Ceil(b.X+b.W+grow)), int(math.Ceil(b.Y+b.H+grow)))
}

func (b Box) extent(fy, grow float64) (left, right float64) {
	l, r, d := b.Radii[3], b.Radii[2], fy-b.Y-b.H-grow
	if fy < b.Y+b.H/2 {
		l, r, d = b.Radii[0], b.Radii[1], b.Y-grow-fy
	}
	return b.X - grow + curve(l+grow, d), b.X + b.W + grow - curve(r+grow, d)
}

func curve(r, d float64) float64 {
	r = max(r, 0)
	into := d + r
	if into <= 0 {
		return 0
	}
	return r - math.Sqrt(max(r*r-into*into, 0))
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
