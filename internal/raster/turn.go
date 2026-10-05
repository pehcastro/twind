package raster

import (
	"image"
	"math"
)

type spin struct{ sin, cos, cx, cy float64 }

func turning(op Op) spin {
	sin, cos := math.Sincos(op.Turn * 2 * math.Pi)
	return spin{sin, cos, op.Box.X + op.Box.W/2 + op.Pivot.X, op.Box.Y + op.Box.H/2 + op.Pivot.Y}
}

func (s spin) back(x, y int) (float64, float64) {
	dx, dy := float64(x)+0.5-s.cx, float64(y)+0.5-s.cy
	return s.cx + dx*s.cos + dy*s.sin, s.cy - dx*s.sin + dy*s.cos
}

func Turned(a image.Rectangle, turn float64, about Point) image.Rectangle {
	sin, cos := math.Sincos(turn * 2 * math.Pi)
	x, y := float64(a.Min.X+a.Max.X)/2-about.X, float64(a.Min.Y+a.Max.Y)/2-about.Y
	x, y = about.X+x*cos-y*sin, about.Y+x*sin+y*cos
	w, h := float64(a.Dx())/2, float64(a.Dy())/2
	rx, ry := w*math.Abs(cos)+h*math.Abs(sin), w*math.Abs(sin)+h*math.Abs(cos)
	return image.Rect(int(math.Floor(x-rx)), int(math.Floor(y-ry)), int(math.Ceil(x+rx)), int(math.Ceil(y+ry)))
}

func (s spin) span(y int, a Rect, area image.Rectangle) (int, int) {
	if a.W < 0 || a.H < 0 {
		return area.Min.X, area.Min.X
	}
	dy := float64(y) + 0.5 - s.cy
	lo, hi := math.Inf(-1), math.Inf(1)
	for _, k := range [2][4]float64{{s.cx + dy*s.sin, s.cos, a.X, a.X + a.W}, {s.cy + dy*s.cos, -s.sin, a.Y, a.Y + a.H}} {
		base, slope := k[0], k[1]
		if slope == 0 {
			if base < k[2] || base > k[3] {
				return area.Min.X, area.Min.X
			}
			continue
		}
		p, q := (k[2]-base)/slope, (k[3]-base)/slope
		lo, hi = max(lo, min(p, q)), min(hi, max(p, q))
	}
	return within(int(math.Ceil(s.cx+lo-0.5)), int(math.Floor(s.cx+hi-0.5))+1, area)
}

func upright(op Op) image.Rectangle {
	if op.Kind == Shadow {
		return geometry(op).area
	}
	return op.Box.pixels(0.5)
}

func (b Box) core() Rect {
	d := max(b.Radii[0], b.Radii[1], b.Radii[2], b.Radii[3], 0.5)
	return Rect{b.X + d, b.Y + d, b.W - 2*d, b.H - 2*d}
}

func (r *Raster) turn(op Op) {
	s, from, top := turning(op), upright(op), r.top()
	area := Turned(from, op.Turn, Point{s.cx, s.cy}).Intersect(top.clip)
	if area.Empty() {
		return
	}
	if op.Kind == Shadow {
		r.cast(op, s, from, area)
		return
	}
	b := op.Box.fit()
	inner, line, paint := b.Inset(op.Width), b.Inset(op.Width/2), premul(op.Color)
	var gx, gy float64
	if len(op.Stops) > 0 {
		r.gradient(op.Stops)
		sin, cos := math.Sincos(op.Angle * math.Pi / 180)
		length := math.Abs(b.W*sin) + math.Abs(b.H*cos)
		gx, gy = sin/length, -cos/length
	}
	var dash pattern
	core := Rect{W: -1}
	switch {
	case op.Dash == Solid && op.Kind == Border:
		core = inner.core()
	case op.Dash == Solid && len(op.Stops) == 0:
		core = b.core()
	case op.Dash == Solid:
	case op.Kind == Border:
		long, gap := dashes(op.Dash, op.Width)
		dash = loop(long, gap, line.perimeter())
		dash.phase = dash.dash / 2
	default:
		long, gap := dashes(op.Dash, min(b.W, b.H))
		dash = loop(long, gap, max(b.W, b.H)+gap)
	}
	for y := area.Min.Y; y < area.Max.Y; y++ {
		if r.skip(y) {
			continue
		}
		lo, hi := s.span(y, Rect{b.X - 0.5, b.Y - 0.5, b.W + 1, b.H + 1}, area)
		coreLo, coreHi := s.span(y, core, area)
		for x, i := lo, top.img.PixOffset(lo, y); x < hi; x, i = x+1, i+4 {
			if x == coreLo && coreLo < coreHi {
				if op.Kind == Fill {
					flood(top.img.Pix[i:top.img.PixOffset(coreHi, y)], paint, 1)
				}
				x, i = coreHi-1, top.img.PixOffset(coreHi-1, y)
				continue
			}
			u, v := s.back(x, y)
			cov := b.cover(u, v)
			if op.Kind == Border {
				cov = max(cov-inner.cover(u, v), 0)
			}
			switch {
			case cov == 0 || op.Dash == Solid:
			case op.Kind == Border:
				cov *= dash.cover(line.along(u, v))
			case b.W < b.H:
				cov *= dash.cover(v - b.Y)
			default:
				cov *= dash.cover(u - b.X)
			}
			src := paint
			if len(op.Stops) > 0 && cov > 0 {
				src = r.sample((u-b.X-b.W/2)*gx + (v-b.Y-b.H/2)*gy + 0.5)
			}
			blend(top.img.Pix[i:i+4:i+4], src, cov)
		}
	}
}

func (r *Raster) cast(op Op, s spin, from, area image.Rectangle) {
	if r.inner == nil {
		r.inner = new(Raster)
	}
	sh := &r.spun
	sh.Pix, sh.Stride, sh.Rect = resize(sh.Pix, 4*from.Dx()*from.Dy()), 4*from.Dx(), from
	paint := premul(op.Color)
	op.Turn, op.Color.A = 0, math.MaxUint8
	r.inner.Draw(sh, []Op{op}, from)
	top, reach, hole := r.top(), Rect{float64(from.Min.X) - 0.5, float64(from.Min.Y) - 0.5, float64(from.Dx()) + 1, float64(from.Dy()) + 1}, Rect{W: -1}
	if !op.Shadow.Inset {
		c := op.Box.fit().core()
		hole = Rect{c.X + 1, c.Y + 1, c.W - 2, c.H - 2}
	}
	for y := area.Min.Y; y < area.Max.Y; y++ {
		if r.skip(y) {
			continue
		}
		lo, hi := s.span(y, reach, area)
		holeLo, holeHi := s.span(y, hole, area)
		for x, i := lo, top.img.PixOffset(lo, y); x < hi; x, i = x+1, i+4 {
			if x == holeLo && holeLo < holeHi {
				x, i = holeHi-1, top.img.PixOffset(holeHi-1, y)
				continue
			}
			u, v := s.back(x, y)
			u, v = u-0.5, v-0.5
			left, up := math.Floor(u), math.Floor(v)
			wx, wy := float32(u-left), float32(v-up)
			var cov float32
			for k, w := range [4]float32{(1 - wx) * (1 - wy), wx * (1 - wy), (1 - wx) * wy, wx * wy} {
				if p := image.Pt(int(left)+k&1, int(up)+k>>1); w > 0 && p.In(from) {
					cov += w * float32(sh.Pix[sh.PixOffset(p.X, p.Y)+3])
				}
			}
			blend(top.img.Pix[i:i+4:i+4], paint, cov/math.MaxUint8)
		}
	}
}
