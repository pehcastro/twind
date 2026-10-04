package chart

import (
	"cmp"
	"image"
	"math"
	"slices"
	"sort"
	"strconv"

	konst "github.com/twind-dev/twind/internal/konst/chart"
	stylekonst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/theme"
)

type point struct{ x, y float32 }

type interval struct{ lo, hi float32 }

type reach struct{ lo, hi int }

type raster struct {
	w, h                        int
	cover                       []float32
	reach                       []reach
	top, bottom                 int
	spans                       [][]interval
	points, curve, lower, under []point
	dst                         *image.RGBA
	token                       func(theme.Token) color.RGBA
	ink                         []uint8
}

type paint struct {
	token       theme.Token
	ink         uint8
	top, bottom float32
	from, to    float32
}

func reuse(r *raster, w, h int) *raster {
	if r != nil && r.w == w && r.h == h {
		return r
	}
	return &raster{w: w, h: h, cover: make([]float32, w*h), reach: make([]reach, h), top: h, spans: make([][]interval, h*konst.Samples)}
}

func pixelMetrics(cell image.Point) metrics {
	f := float32(cell.Y) / stylekonst.RemPixels
	return metrics{
		half: konst.StrokeWidth * f / 2, active: konst.ActiveDotRadius * f,
		radius: konst.BarRadius * f, gap: max(1, round(konst.BarGap*f)), grid: max(1, round(konst.GridWidth*f)),
	}
}

func (c *Chart) paint(dst *image.RGBA, p plot, m metrics, hover int, token func(theme.Token) color.RGBA) {
	c.pixels = reuse(c.pixels, dst.Rect.Dx(), dst.Rect.Dy())
	c.pixels.dst, c.pixels.token = dst, token
	c.draw(c.pixels, p, m, hover)
}

func (c *Chart) draw(r *raster, p plot, m metrics, hover int) {
	w, h := float32(r.w), float32(r.h)
	whole := func(float32) (float32, float32) { return 0, h }
	if hover > 0 && p.kind == Bar {
		band := w / float32(p.points)
		r.span(float32(hover-1)*band, float32(hover)*band, whole)
		r.commit(paint{token: theme.Muted, top: konst.CursorAlpha, bottom: konst.CursorAlpha})
	}
	if m.grid > 0 {
		for _, t := range p.ticks {
			y := min(max(round(p.y(t, m)-m.grid/2), 0), h-m.grid)
			r.span(0, w, func(float32) (float32, float32) { return y, y + m.grid })
			r.commit(paint{token: theme.Border, top: konst.GridAlpha, bottom: konst.GridAlpha})
		}
	}
	if hover > 0 && p.kind != Bar {
		x := round((float32(hover) - 0.5) * w / float32(p.points))
		r.span(x, x+m.grid, whole)
		r.commit(paint{token: theme.Border, top: 1, bottom: 1})
	}
	if p.kind == Area && p.points > 1 {
		c.areas(r, p, m)
	}
	last := len(c.Series) - 1
	for s, series := range c.Series {
		solid := paint{token: series.Color, ink: uint8(2*s + 1), top: 1, bottom: 1}
		switch p.kind {
		case Bar:
			for i, v := range p.top[s] {
				x0, x1 := p.bar(i, s, w, m)
				y0, y1 := p.y(v, m), p.y(p.base[s][i], m)
				far, near := m.radius, m.radius
				if p.stacked && s != last {
					far = 0
				}
				if p.stacked && s != 0 {
					near = 0
				}
				if y0 > y1 {
					y0, y1, far, near = y1, y0, near, far
				}
				r.bar(x0, x1, y0, y1, far, near)
				r.commit(solid)
			}
		case Line, Area:
			r.points = p.series(s, w, m, r.points)
			r.curve = natural(r.curve, r.points)
			r.stroke(r.curve, m.half)
			r.commit(solid)
			if hover > 0 && m.active > 0 {
				r.disc(r.points[hover-1], m.active)
				r.commit(solid)
			}
		default:
			panic("chart: unknown kind " + strconv.Itoa(int(p.kind)))
		}
	}
}

func (c *Chart) areas(r *raster, p plot, m metrics) {
	w, h := float32(r.w), float32(r.h)
	for s, series := range c.Series {
		r.points = p.series(s, w, m, r.points)
		r.curve = natural(r.curve, r.points)
		if p.stacked && s > 0 {
			r.under = p.series(s-1, w, m, r.under)
			r.lower = natural(r.lower, r.under)
		} else {
			zero := p.y(0, m)
			r.lower = append(r.lower[:0], point{r.points[0].x, zero}, point{r.points[len(r.points)-1].x, zero})
		}
		top, bottom := h, float32(0)
		for _, pt := range r.curve {
			top = min(top, pt.y)
		}
		for _, pt := range r.lower {
			bottom = max(bottom, pt.y)
		}
		r.span(r.points[0].x, r.points[len(r.points)-1].x, func(x float32) (float32, float32) { return at(r.curve, x), at(r.lower, x) })
		r.commit(paint{token: series.Color, ink: uint8(2*s + 2), top: konst.FillTop * konst.FillOpacity, bottom: konst.FillBottom * konst.FillOpacity, from: top, to: bottom})
	}
}

func at(poly []point, x float32) float32 {
	i := max(min(sort.Search(len(poly), func(i int) bool { return poly[i].x >= x }), len(poly)-1), 1)
	a, b := poly[i-1], poly[i]
	if b.x == a.x {
		return b.y
	}
	return a.y + (b.y-a.y)*(x-a.x)/(b.x-a.x)
}

func natural(out, pts []point) []point {
	out = out[:0]
	if len(pts) < 3 {
		return append(out, pts...)
	}
	cx0, cx1 := controls(pts, func(p point) float32 { return p.x })
	cy0, cy1 := controls(pts, func(p point) float32 { return p.y })
	out = append(out, pts[0])
	for i := range len(pts) - 1 {
		a, e := pts[i], pts[i+1]
		c0, c1 := point{cx0[i], cy0[i]}, point{cx1[i], cy1[i]}
		chord := math.Hypot(float64(e.x-a.x), float64(e.y-a.y))
		steps := min(max(int(math.Ceil(chord/konst.CurveStep)), 1), konst.MaxCurveSteps)
		for k := 1; k <= steps; k++ {
			t := float32(k) / float32(steps)
			u := 1 - t
			out = append(out, point{
				u*u*u*a.x + 3*u*u*t*c0.x + 3*u*t*t*c1.x + t*t*t*e.x,
				u*u*u*a.y + 3*u*u*t*c0.y + 3*u*t*t*c1.y + t*t*t*e.y,
			})
		}
	}
	return out
}

func controls(pts []point, axis func(point) float32) (first, second []float32) {
	n := len(pts) - 1
	x := func(i int) float32 { return axis(pts[i]) }
	a, b, r := make([]float32, n), make([]float32, n), make([]float32, n)
	a[0], b[0], r[0] = 0, 2, x(0)+2*x(1)
	for i := 1; i < n-1; i++ {
		a[i], b[i], r[i] = 1, 4, 4*x(i)+2*x(i+1)
	}
	a[n-1], b[n-1], r[n-1] = 2, 7, 8*x(n-1)+x(n)
	for i := 1; i < n; i++ {
		m := a[i] / b[i-1]
		b[i] -= m
		r[i] -= m * r[i-1]
	}
	a[n-1] = r[n-1] / b[n-1]
	for i := n - 2; i >= 0; i-- {
		a[i] = (r[i] - a[i+1]) / b[i]
	}
	b[n-1] = (x(n) + a[n-1]) / 2
	for i := range n - 1 {
		b[i] = 2*x(i+1) - a[i+1]
	}
	return a, b
}

func floor(v float32) int { return int(math.Floor(float64(v))) }

func (r *raster) mark(y, x0, x1 int) {
	e := &r.reach[y]
	if e.lo < e.hi {
		x0, x1 = min(x0, e.lo), max(x1, e.hi)
	}
	*e = reach{x0, x1}
	r.top, r.bottom = min(r.top, y), max(r.bottom, y+1)
}

func (r *raster) span(x0, x1 float32, edges func(x float32) (top, bottom float32)) {
	x0, x1 = max(x0, 0), min(x1, float32(r.w))
	first, last := floor(x0), ceil32(x1)
	lo, hi := r.h, 0
	for px := first; px < last; px++ {
		var samples [konst.Samples]interval
		inner, outer := interval{0, float32(r.h)}, interval{float32(r.h), 0}
		for k := range samples {
			x := float32(px) + (float32(k)+0.5)/konst.Samples
			s := interval{1, 0}
			if x >= x0 && x < x1 {
				s.lo, s.hi = edges(x)
				s = interval{max(s.lo, 0), min(s.hi, float32(r.h))}
			}
			samples[k] = s
			inner = interval{max(inner.lo, s.lo), min(inner.hi, s.hi)}
			if s.lo < s.hi {
				outer = interval{min(outer.lo, s.lo), max(outer.hi, s.hi)}
			}
		}
		full, through := ceil32(inner.lo), floor(inner.hi)
		for py := floor(outer.lo); float32(py) < outer.hi; py++ {
			at := &r.cover[py*r.w+px]
			if py >= full && py < through {
				*at++
				continue
			}
			for _, s := range samples {
				if c := min(s.hi, float32(py+1)) - max(s.lo, float32(py)); c > 0 {
					*at += c / konst.Samples
				}
			}
		}
		if outer.lo < outer.hi {
			lo, hi = min(lo, floor(outer.lo)), max(hi, ceil32(outer.hi))
		}
	}
	for py := lo; py < hi; py++ {
		r.mark(py, first, last)
	}
}

func ceil32(v float32) int { return int(math.Ceil(float64(v))) }

func (r *raster) bar(x0, x1, y0, y1, top, bottom float32) {
	if y1 <= y0 || x1 <= x0 {
		return
	}
	limit := min((x1-x0)/2, (y1-y0)/2)
	top, bottom = min(top, limit), min(bottom, limit)
	arc := func(radius, x float32) float32 {
		d := max(x0+radius-x, x-(x1-radius), 0)
		if radius == 0 || d == 0 {
			return 0
		}
		return radius - float32(math.Sqrt(float64(max(radius*radius-d*d, 0))))
	}
	r.span(x0, x1, func(x float32) (float32, float32) { return y0 + arc(top, x), y1 - arc(bottom, x) })
}

func (r *raster) disc(c point, radius float32) {
	r.span(c.x-radius, c.x+radius, func(x float32) (float32, float32) {
		d := float32(math.Sqrt(float64(max(radius*radius-(x-c.x)*(x-c.x), 0))))
		return c.y - d, c.y + d
	})
}

func (r *raster) stroke(poly []point, half float32) {
	if len(poly) == 1 {
		r.disc(poly[0], half)
		return
	}
	rows := r.h * konst.Samples
	first, last := rows, -1
	for i := 1; i < len(poly); i++ {
		a, e := poly[i-1], poly[i]
		from := max(int(math.Ceil(float64((min(a.y, e.y)-half)*konst.Samples-0.5))), 0)
		to := min(floor((max(a.y, e.y)+half)*konst.Samples-0.5), rows-1)
		for k := from; k <= to; k++ {
			if iv, ok := capsule(a, e, (float32(k)+0.5)/konst.Samples, half); ok {
				r.spans[k] = append(r.spans[k], iv)
				first, last = min(first, k), max(last, k)
			}
		}
	}
	for k := first; k <= last; k++ {
		spans, y := r.spans[k], k/konst.Samples
		slices.SortFunc(spans, func(a, b interval) int { return cmp.Compare(a.lo, b.lo) })
		row := r.cover[y*r.w:][:r.w]
		for i := 0; i < len(spans); {
			run := spans[i]
			for i++; i < len(spans) && spans[i].lo <= run.hi; i++ {
				run.hi = max(run.hi, spans[i].hi)
			}
			run.lo, run.hi = max(run.lo, 0), min(run.hi, float32(r.w))
			if run.lo >= run.hi {
				continue
			}
			r.mark(y, floor(run.lo), int(math.Ceil(float64(run.hi))))
			for px := floor(run.lo); float32(px) < run.hi; px++ {
				row[px] += (min(run.hi, float32(px+1)) - max(run.lo, float32(px))) / konst.Samples
			}
		}
		r.spans[k] = spans[:0]
	}
}

func capsule(a, e point, y, radius float32) (interval, bool) {
	out := interval{math.MaxFloat32, -math.MaxFloat32}
	grow := func(lo, hi float32) {
		if lo <= hi {
			out = interval{min(out.lo, lo), max(out.hi, hi)}
		}
	}
	for _, end := range [2]point{a, e} {
		if d := y - end.y; d*d <= radius*radius {
			w := float32(math.Sqrt(float64(radius*radius - d*d)))
			grow(end.x-w, end.x+w)
		}
	}
	dx, dy, v := e.x-a.x, e.y-a.y, y-a.y
	length2 := dx*dx + dy*dy
	if length2 > 0 {
		reach := radius * float32(math.Sqrt(float64(length2)))
		lo, hi := float32(-math.MaxFloat32), float32(math.MaxFloat32)
		if dy != 0 {
			p, q := a.x+(v*dx-reach)/dy, a.x+(v*dx+reach)/dy
			lo, hi = min(p, q), max(p, q)
		} else if v*v > radius*radius {
			lo, hi = 1, 0
		}
		if dx != 0 {
			p, q := a.x-v*dy/dx, a.x+(length2-v*dy)/dx
			lo, hi = max(lo, min(p, q)), min(hi, max(p, q))
		} else if t := v * dy; t < 0 || t > length2 {
			lo, hi = 1, 0
		}
		grow(lo, hi)
	}
	return out, out.lo <= out.hi
}

func (r *raster) commit(p paint) {
	var c color.RGBA
	if r.dst != nil {
		c = r.token(p.token)
	}
	for y := r.top; y < r.bottom; y++ {
		e := r.reach[y]
		r.reach[y] = reach{}
		alpha := p.top
		if p.to > p.from {
			alpha += (p.bottom - p.top) * min(max((float32(y)+0.5-p.from)/(p.to-p.from), 0), 1)
		}
		row := r.cover[y*r.w:][:r.w]
		var line []uint8
		if r.dst != nil {
			line = r.dst.Pix[r.dst.PixOffset(r.dst.Rect.Min.X, r.dst.Rect.Min.Y+y):]
		}
		for x := e.lo; x < e.hi; x++ {
			cover := min(row[x], 1)
			row[x] = 0
			switch {
			case cover <= 0:
			case line == nil:
				if p.ink != 0 && cover >= konst.CellThreshold {
					r.ink[y*r.w+x] = p.ink
				}
			default:
				a := uint32(cover*alpha*float32(c.A) + 0.5)
				px := line[4*x:][:4]
				for i, v := range [4]uint8{c.R, c.G, c.B, math.MaxUint8} {
					px[i] = uint8((uint32(v)*a + uint32(px[i])*(math.MaxUint8-a) + math.MaxUint8/2) / math.MaxUint8)
				}
			}
		}
	}
	r.top, r.bottom = r.h, 0
}
