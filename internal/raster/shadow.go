package raster

import (
	"image"
	"math"

	konst "github.com/pehcastro/twind/internal/konst/raster"
)

type corner struct {
	top, step    float64
	first, count int
}

type strip struct{ hi, at int }

type hit struct {
	w     float64
	strip int
}

type side struct {
	lo, hi      int
	corners     [2]int
	edge        []float64
	sign, total float64
	hits        []hit
}

type cast struct {
	box, shape     Box
	sigma, reach   float64
	area           image.Rectangle
	bandLo, bandHi float64
}

func geometry(op Op) cast {
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
	area := shape.pixels(reach + 0.5)
	if s.Inset {
		area = box.pixels(0.5)
	}
	bandLo, bandHi := box.straight(0)
	shapeLo, shapeHi := shape.straight(reach)
	return cast{box, shape, sigma, reach, area, max(bandLo, shapeLo), min(bandHi, shapeHi)}
}

func (r *Raster) shadow(op Op) {
	g := geometry(op)
	shape, sigma, reach, s := g.shape, g.sigma, g.reach, op.Shadow
	paint, top := premul(op.Color), r.top()
	area := g.area.Intersect(top.clip)
	if area.Empty() {
		return
	}
	per := konst.PhiSteps / (2 * reach)
	if r.phi == nil {
		r.phi = new([konst.PhiSteps + 1]float64)
		for i := range r.phi {
			r.phi[i] = phi(i)
		}
	}
	if sigma > 0 {
		r.profile(shape, per, reach, area)
	}
	quiet := math.Inf(-1)
	if !s.Inset {
		quiet = konst.Quiet / float64(paint[3])
	}
	ax, o := area.Min.X, r.outline(g.box)
	r.row = resize(r.row, area.Dx())
	r.rows.y = math.MinInt
	for y := area.Min.Y; y < area.Max.Y; y++ {
		if r.skip(y) {
			continue
		}
		fy := float64(y) + 0.5
		band := fy >= g.bandLo && fy <= g.bandHi
		if band && r.rows.again(top.img, y) {
			continue
		}
		anyLo, anyHi := o.touched(y, area)
		fullLo, fullHi := o.full(y, area)
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
			if weight < quiet {
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
			r.notches(fy, per, reach)
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
			lo, hi := p[0], p[1]
			for sigma > 0 && lo < hi && weight*(r.right[lo-ax]-r.left[lo-ax]) < quiet {
				lo++
			}
			for sigma > 0 && lo < hi && weight*(r.right[hi-1-ax]-r.left[hi-1-ax]) < quiet {
				hi--
			}
			if lo >= hi {
				continue
			}
			row := r.row[lo-ax : hi-ax]
			if sigma == 0 {
				for i := range row {
					row[i] = float64(shape.cover(float64(lo+i)+0.5, fy))
				}
			} else {
				left, right := r.left[lo-ax:hi-ax], r.right[lo-ax:hi-ax]
				for i := range row {
					row[i] = weight * (right[i] - left[i])
				}
				for n := range r.sides {
					sd := &r.sides[n]
					a, b := max(lo, sd.lo), min(hi, sd.hi)
					if len(sd.hits) == 0 || a >= b {
						continue
					}
					part := row[a-lo : b-lo]
					for _, h := range sd.hits {
						st := r.strips[h.strip]
						end := min(max(st.hi, a), b)
						table, rest := r.tables[st.at+a-sd.lo:st.at+end-sd.lo], part[:end-a]
						for i := range rest {
							rest[i] -= h.w * table[i]
						}
					}
					edge := sd.edge[a-sd.lo : b-sd.lo]
					for i := range part {
						part[i] += sd.total * edge[i]
					}
				}
			}
			if s.Inset {
				for i := range row {
					row[i] = 1 - row[i]
				}
			}
			for _, rim := range rims {
				for x := max(rim[0], lo); x < min(rim[1], hi); x++ {
					mask := float64(o.cover(x, y))
					if !s.Inset {
						mask = 1 - mask
					}
					row[x-lo] *= mask
				}
			}
			pix := top.img.Pix[top.img.PixOffset(lo, y):top.img.PixOffset(hi, y)]
			for i, v := range row {
				blend(pix[4*i:4*i+4:4*i+4], paint, float32(v))
			}
		}
		for n := range spans {
			flood(top.img.Pix[top.img.PixOffset(parts[2*n][1], y):top.img.PixOffset(parts[2*n+1][0], y)], paint, float32(flat))
		}
	}
}

func (r *Raster) profile(shape Box, per, reach float64, area image.Rectangle) {
	x0, x1, ax := shape.X, shape.X+shape.W, area.Min.X
	r.left, r.right = resize(r.left, area.Dx()), resize(r.right, area.Dx())
	zones := [2][2]float64{{x0 - reach, x0 + max(shape.Radii[0], shape.Radii[3]) + reach}, {x1 - max(shape.Radii[1], shape.Radii[2]) - reach, x1 + reach}}
	for s, z := range zones {
		lo := min(max(int(math.Floor(z[0])), ax), area.Max.X)
		hi := max(min(int(math.Ceil(z[1])), area.Max.X), lo)
		edge := r.left
		if s == 1 {
			edge = r.right
		}
		r.sides[s] = side{lo: lo, hi: hi, corners: [2][2]int{{0, 3}, {1, 2}}[s], edge: edge[lo-ax : hi-ax], sign: float64(1 - 2*s), hits: r.sides[s].hits}
	}
	z0, z1 := r.sides[0], r.sides[1]
	for x := ax; x < area.Max.X; x++ {
		fx := float64(x) + 0.5
		switch {
		case x < z0.lo:
			r.left[x-ax], r.right[x-ax] = 1, 1
		case x >= z1.hi:
			r.left[x-ax], r.right[x-ax] = 0, 0
		case x >= z0.hi && x < z1.lo:
			r.left[x-ax], r.right[x-ax] = 0, 1
		default:
			r.left[x-ax], r.right[x-ax] = r.cdf((x0-fx)*per), r.cdf((x1-fx)*per)
		}
	}
	r.flatLo, r.flatHi = area.Max.X, area.Max.X
	for x := ax; x < area.Max.X && r.flatLo == area.Max.X; x++ {
		if r.right[x-ax]-r.left[x-ax] == 1 {
			r.flatLo = x
		}
	}
	for x := area.Max.X - 1; x >= r.flatLo && r.flatHi == area.Max.X; x-- {
		if r.right[x-ax]-r.left[x-ax] == 1 {
			r.flatHi = x + 1
		}
	}
	r.tables, r.strips = r.tables[:0], r.strips[:0]
	for c, rad := range shape.Radii {
		n := int(math.Ceil(rad * konst.ShadowStrips / (2 * reach)))
		k, sd := &r.corners[c], r.sides[(c&1)^(c>>1)]
		*k = corner{top: shape.Y, first: len(r.strips), count: n}
		if n == 0 {
			continue
		}
		k.step = rad / float64(n)
		if c >= 2 {
			k.top = shape.Y + shape.H - rad
		}
		if float64(area.Max.Y-1)+0.5+reach <= k.top || float64(area.Min.Y)+0.5-reach >= k.top+float64(k.count)*k.step {
			k.count = 0
			continue
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
		last := edge(0)
		for j := range n {
			first := last
			last = edge(float64(j + 1))
			m := min(konst.StripSamples, 1+int(math.Abs(last-first)*konst.StripSamples/reach))
			var edges [konst.StripSamples]float64
			for i := range m {
				edges[i] = edge(float64(j) + (float64(i)+0.5)/float64(m))
			}
			lo := min(max(int(math.Floor(min(first, last)-reach)), sd.lo), sd.hi)
			hi := min(max(int(math.Ceil(max(first, last)+reach)), sd.lo), sd.hi)
			r.strips = append(r.strips, strip{hi: hi, at: len(r.tables)})
			for range lo - sd.lo {
				r.tables = append(r.tables, 1)
			}
			for x := lo; x < hi; x++ {
				sum := 0.0
				for _, e := range edges[:m] {
					sum += r.cdf((e - float64(x) - 0.5) * per)
				}
				r.tables = append(r.tables, sum/float64(m))
			}
		}
	}
}

func (r *Raster) notches(fy, per, reach float64) {
	for s := range r.sides {
		sd := &r.sides[s]
		sd.hits, sd.total = sd.hits[:0], 0
		for _, c := range sd.corners {
			if !r.near(c, fy, reach) {
				continue
			}
			k := &r.corners[c]
			j0 := max(int(math.Floor((fy-reach-k.top)/k.step)), 0)
			j1 := min(int(math.Ceil((fy+reach-k.top)/k.step)), k.count)
			below := r.cdf((k.top + float64(j0)*k.step - fy) * per)
			for j := j0; j < j1; j++ {
				above := r.cdf((k.top + float64(j+1)*k.step - fy) * per)
				w := sd.sign * (above - below)
				below = above
				if w != 0 {
					sd.total += w
					sd.hits = append(sd.hits, hit{w: w, strip: k.first + j})
				}
			}
		}
	}
}

func (r *Raster) near(c int, fy, reach float64) bool {
	k := &r.corners[c]
	return k.count > 0 && fy+reach > k.top && fy-reach < k.top+float64(k.count)*k.step
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

func phi(i int) float64 {
	b := phiTable[8*i : 8*i+8]
	return math.Float64frombits(uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 | uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56)
}
