package raster

import (
	"bytes"
	"encoding/binary"
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

type strip struct{ lo, at, n int }

type corner struct {
	top, step    float64
	first, count int
}

type side struct {
	lo, hi  int
	corners [2]int
	edge    []float64
	sign    float64
}

type Raster struct {
	layers  []layer
	groups  []image.RGBA
	depth   int
	phi     [konst.PhiSteps + 1]float64
	ramp    [konst.GradientSteps + 1][4]float32
	left    []float64
	right   []float64
	row     []float64
	solid   []uint8
	rows    repeat
	flatLo  int
	flatHi  int
	tables  []float64
	strips  []strip
	corners [4]corner
	sides   [2]side
}

func (r *Raster) Draw(dst *image.RGBA, ops []Op, tile image.Rectangle) {
	tile = tile.Intersect(dst.Bounds())
	if !covers(ops, tile) {
		for y := tile.Min.Y; y < tile.Max.Y; y++ {
			clear(dst.Pix[dst.PixOffset(tile.Min.X, y):dst.PixOffset(tile.Max.X, y)])
		}
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
	dst := r.top().img
	for y := l.clip.Min.Y; y < l.clip.Max.Y; y++ {
		fy := float64(y) + 0.5
		lo, hi := l.mask.touched(fy, l.clip)
		fullLo, fullHi := l.mask.full(fy, l.clip)
		for x := lo; x < hi; x++ {
			cov := l.opacity
			if x < fullLo || x >= fullHi {
				cov *= l.mask.cover(float64(x)+0.5, fy)
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
	opaque := len(op.Stops) == 0 && op.Color.A == math.MaxUint8 && op.Dash == Solid
	top := r.top()
	area := b.pixels(0.5).Intersect(top.clip)
	if opaque {
		r.solid = resize(r.solid, 4*area.Dx())
		for i := 0; i < len(r.solid); i += 4 {
			r.solid[i], r.solid[i+1], r.solid[i+2], r.solid[i+3] = op.Color.R, op.Color.G, op.Color.B, op.Color.A
		}
	}
	bandLo, bandHi := b.straight(0)
	var lo, hi, fullLo, fullHi int
	banded := false
	for y := area.Min.Y; y < area.Max.Y; y++ {
		fy := float64(y) + 0.5
		band := fy >= bandLo && fy <= bandHi
		if !banded || !band {
			lo, hi = b.touched(fy, area)
			fullLo, fullHi = b.full(fy, area)
		}
		banded = band
		for x, i := lo, top.img.PixOffset(lo, y); x < hi; x, i = x+1, i+4 {
			fx := float64(x) + 0.5
			cov := float32(1)
			if x < fullLo || x >= fullHi {
				cov = b.cover(fx, fy)
			} else if opaque {
				n := copy(top.img.Pix[i:i+(fullHi-x)*4], r.solid)
				x, i = fullHi-1, i+n-4
				continue
			}
			if len(op.Stops) > 0 {
				paint = r.sample((fx-cx)*gx + (fy-cy)*gy + 0.5)
			}
			switch {
			case op.Dash == Solid:
			case across:
				cov *= dash.cover(fy - b.Y)
			default:
				cov *= dash.cover(fx - b.X)
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
	if op.Dash != Solid {
		r.dashedBorder(op, outer, inner, area)
		return
	}
	bandLo, bandHi := outer.straight(0)
	innerLo, innerHi := inner.straight(0)
	bandLo, bandHi = max(bandLo, innerLo), min(bandHi, innerHi)
	r.rows.y = math.MinInt
	for y := area.Min.Y; y < area.Max.Y; y++ {
		fy := float64(y) + 0.5
		band := fy >= bandLo && fy <= bandHi
		if band && r.rows.again(top.img, y) {
			continue
		}
		lo, hi := outer.touched(fy, area)
		fullLo, fullHi := outer.full(fy, area)
		holeLo, holeHi := inner.full(fy, area)
		edgeLo, edgeHi := inner.touched(fy, area)
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
					fx := float64(x) + 0.5
					blend(top.img.Pix[i:i+4], paint, max(outer.cover(fx, fy)-inner.cover(fx, fy), 0))
				}
			}
		}
	}
}

func flood(pix []uint8, src [4]float32, cov float32) {
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

func (r *Raster) shadow(op Op) {
	box := op.Box.fit()
	s := op.Shadow
	spread := s.Spread
	if s.Inset {
		spread = -spread
	}
	shape := box.grow(spread).fit()
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
	if area.Empty() {
		return
	}
	per := konst.PhiSteps / (2 * reach)
	if sigma > 0 {
		r.profile(shape, per, reach, area)
	}
	r.row = resize(r.row, area.Dx())
	row, ax := r.row, area.Min.X
	bandLo, bandHi := box.straight(0)
	shapeLo, shapeHi := shape.straight(reach)
	bandLo, bandHi = max(bandLo, shapeLo), min(bandHi, shapeHi)
	r.rows.y = math.MinInt
	for y := area.Min.Y; y < area.Max.Y; y++ {
		fy := float64(y) + 0.5
		band := fy >= bandLo && fy <= bandHi
		if band && r.rows.again(top.img, y) {
			continue
		}
		anyLo, anyHi := box.touched(fy, area)
		fullLo, fullHi := box.full(fy, area)
		spans := [2][2]int{{anyLo, anyLo}, {anyLo, anyHi}}
		if !s.Inset {
			spans = [2][2]int{{area.Min.X, fullLo}, {fullHi, area.Max.X}}
		}
		if band {
			r.rows.keep(top.img, y, spans)
		}
		rims := [2][2]int{{anyLo, min(fullLo, anyHi)}, {max(fullHi, anyLo), anyHi}}
		flat, flatLo, flatHi := 0.0, ax, ax
		weight := 0.0
		if sigma > 0 {
			weight = r.cdf((shape.Y+shape.H-fy)*per) - r.cdf((shape.Y-fy)*per)
			if weight <= 0 && !s.Inset {
				continue
			}
			flat, flatLo, flatHi = weight, r.flatLo, r.flatHi
			if r.near(0, fy, reach) || r.near(3, fy, reach) {
				flatLo = max(flatLo, r.sides[0].hi)
			}
			if r.near(1, fy, reach) || r.near(2, fy, reach) {
				flatHi = min(flatHi, r.sides[1].lo)
			}
			for _, rim := range rims {
				if max(rim[0], flatLo) < min(rim[1], flatHi) {
					flatHi = flatLo
				}
			}
		}
		if s.Inset {
			flat = 1 - flat
		}
		var parts [4][2]int
		for n, sp := range spans {
			lo := min(max(flatLo, sp[0]), sp[1])
			hi := min(max(flatHi, lo), sp[1])
			parts[2*n], parts[2*n+1] = [2]int{sp[0], lo}, [2]int{hi, sp[1]}
		}
		for _, p := range parts {
			part := row[p[0]-ax : p[1]-ax]
			if sigma == 0 {
				for i := range part {
					part[i] = float64(shape.cover(float64(p[0]+i)+0.5, fy))
				}
				continue
			}
			left, right := r.left[p[0]-ax:p[1]-ax], r.right[p[0]-ax:p[1]-ax]
			for i := range part {
				part[i] = weight * (right[i] - left[i])
			}
		}
		if sigma > 0 {
			r.notches(fy, per, reach, spans, ax)
		}
		for _, p := range parts {
			for x := p[0]; s.Inset && x < p[1]; x++ {
				row[x-ax] = 1 - row[x-ax]
			}
		}
		for _, rim := range rims {
			for x := rim[0]; x < rim[1]; x++ {
				mask := float64(box.cover(float64(x)+0.5, fy))
				if !s.Inset {
					mask = 1 - mask
				}
				row[x-ax] *= mask
			}
		}
		for _, p := range parts {
			for x, i := p[0], top.img.PixOffset(p[0], y); x < p[1]; x, i = x+1, i+4 {
				blend(top.img.Pix[i:i+4:i+4], paint, float32(row[x-ax]))
			}
		}
		for n := range spans {
			flood(top.img.Pix[top.img.PixOffset(parts[2*n][1], y):top.img.PixOffset(parts[2*n+1][0], y)], paint, float32(flat))
		}
	}
}

func (r *Raster) profile(shape Box, per, reach float64, area image.Rectangle) {
	x0, x1 := shape.X, shape.X+shape.W
	r.left, r.right = resize(r.left, area.Dx()), resize(r.right, area.Dx())
	r.flatLo, r.flatHi = area.Max.X, area.Max.X
	for i := range r.left {
		fx := float64(area.Min.X+i) + 0.5
		r.left[i], r.right[i] = r.cdf((x0-fx)*per), r.cdf((x1-fx)*per)
		if r.right[i]-r.left[i] == 1 {
			r.flatLo = min(r.flatLo, area.Min.X+i)
			r.flatHi = area.Min.X + i + 1
		}
	}
	r.strips, r.tables = r.strips[:0], r.tables[:0]
	for c, rad := range shape.Radii {
		n := int(math.Ceil(rad * konst.ShadowStrips / (2 * reach)))
		k := &r.corners[c]
		*k = corner{top: shape.Y, first: len(r.strips), count: n}
		if n == 0 {
			continue
		}
		k.step = rad / float64(n)
		if c >= 2 {
			k.top = shape.Y + shape.H - rad
		}
		edge := func(f float64) float64 {
			d := rad - f*k.step
			if c >= 2 {
				d = rad - d
			}
			width := rad - math.Sqrt(max(rad*rad-d*d, 0))
			if c == 1 || c == 2 {
				return x1 - width
			}
			return x0 + width
		}
		for j := range n {
			first, last := edge(float64(j)), edge(float64(j+1))
			m := min(konst.StripSamples, 1+int(math.Abs(last-first)*konst.StripSamples/reach))
			var edges [konst.StripSamples]float64
			for i := range m {
				edges[i] = edge(float64(j) + (float64(i)+0.5)/float64(m))
			}
			lo, hi := int(math.Floor(min(first, last)-reach)), int(math.Ceil(max(first, last)+reach))
			r.strips = append(r.strips, strip{lo: lo, at: len(r.tables), n: hi - lo})
			for x := lo; x < hi; x++ {
				sum := 0.0
				for _, e := range edges[:m] {
					sum += r.cdf((e - float64(x) - 0.5) * per)
				}
				r.tables = append(r.tables, sum/float64(m))
			}
		}
	}
	zones := [2][2]float64{{x0 - reach, x0 + max(shape.Radii[0], shape.Radii[3]) + reach}, {x1 - max(shape.Radii[1], shape.Radii[2]) - reach, x1 + reach}}
	for s, z := range zones {
		lo := min(max(int(math.Floor(z[0])), area.Min.X), area.Max.X)
		hi := max(min(int(math.Ceil(z[1])), area.Max.X), lo)
		edge := r.left
		if s == 1 {
			edge = r.right
		}
		r.sides[s] = side{lo: lo, hi: hi, corners: [2][2]int{{0, 3}, {1, 2}}[s], edge: edge[lo-area.Min.X : hi-area.Min.X], sign: float64(1 - 2*s)}
	}
}

func (r *Raster) notches(fy, per, reach float64, spans [2][2]int, ax int) {
	for _, sd := range r.sides {
		total := 0.0
		for _, c := range sd.corners {
			if !r.near(c, fy, reach) {
				continue
			}
			k := r.corners[c]
			j0 := max(int(math.Floor((fy-reach-k.top)/k.step)), 0)
			j1 := min(int(math.Ceil((fy+reach-k.top)/k.step)), k.count)
			below := r.cdf((k.top + float64(j0)*k.step - fy) * per)
			for j := j0; j < j1; j++ {
				above := r.cdf((k.top + float64(j+1)*k.step - fy) * per)
				w := sd.sign * (above - below)
				below = above
				if w == 0 {
					continue
				}
				total += w
				st := r.strips[k.first+j]
				for _, sp := range spans {
					lo, hi := max(sp[0], sd.lo), min(sp[1], sd.hi)
					if lo >= hi {
						continue
					}
					mid, end := min(max(st.lo, lo), hi), min(st.lo+st.n, hi)
					for i := range r.row[lo-ax : mid-ax] {
						r.row[lo-ax+i] -= w
					}
					if mid < end {
						row, table := r.row[mid-ax:end-ax], r.tables[st.at+mid-st.lo:st.at+end-st.lo]
						for i := range row {
							row[i] -= w * table[i]
						}
					}
				}
			}
		}
		if total == 0 {
			continue
		}
		for _, sp := range spans {
			lo, hi := max(sp[0], sd.lo), min(sp[1], sd.hi)
			if lo >= hi {
				continue
			}
			row, edge := r.row[lo-ax:hi-ax], sd.edge[lo-sd.lo:hi-sd.lo]
			for i := range row {
				row[i] += total * edge[i]
			}
		}
	}
}

func (r *Raster) near(c int, fy, reach float64) bool {
	k := r.corners[c]
	return k.count > 0 && fy+reach > k.top && fy-reach < k.top+float64(k.count)*k.step
}

func resize[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	return s[:n]
}

func (r *Raster) cdf(t float64) float64 {
	u := t + konst.PhiSteps/2
	switch {
	case u <= 0:
		return 0
	case u >= konst.PhiSteps:
		return 1
	}
	i := int(u)
	return r.phi[i] + (u-float64(i))*(r.phi[i+1]-r.phi[i])
}

func (r *Raster) gradient(stops []Stop) {
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
	lo = min(max(lo, area.Min.X), area.Max.X)
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
