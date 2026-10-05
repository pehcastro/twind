package chart

import (
	"image"
	"math"

	konst "github.com/pehcastro/twind/internal/konst/chart"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/text"
	"github.com/pehcastro/twind/twi/theme"
)

const turn = 2 * math.Pi

func toward(c point, radius, angle float32) point {
	s, k := math.Sincos(float64(angle))
	return point{c.x + radius*float32(s), c.y - radius*float32(k)}
}

func angleOf(c, q point) float32 {
	a := float32(math.Atan2(float64(q.x-c.x), float64(c.y-q.y)))
	if a < 0 {
		a += turn
	}
	return a
}

func arc(out []point, c point, radius, from, to float32) []point {
	n := min(max(ceil32(float32(math.Abs(float64(to-from)))*radius/konst.CurveStep), 1), konst.MaxArcSteps)
	for k := range n + 1 {
		out = append(out, toward(c, radius, from+(to-from)*float32(k)/float32(n)))
	}
	return out
}

type wheel struct {
	centre               point
	radius, hole, grow   float32
	band, thick, largest float32
}

func (p plot) wheel(w, h float32, unit point, cols int) wheel {
	g := wheel{centre: point{w / 2, h / 2}, radius: min(w, h) / 2 * konst.PolarFill}
	switch p.kind {
	case Pie, Donut:
		g.radius /= 1 + konst.PieGrow
		g.grow = g.radius * konst.PieGrow
		if p.kind == Donut {
			g.hole = g.radius * konst.DonutHole
		}
	case Radial:
		g.hole = g.radius * konst.RadialHole
		g.band = (g.radius - g.hole) / float32(len(p.values))
		g.thick = g.band * (1 - konst.RadialGap)
		for _, v := range p.values {
			g.largest = max(g.largest, float32(v[0]))
		}
	case Radar:
		g.radius = min(w/2-(float32(cols)+konst.RadarCols)*unit.x, h/2-konst.RadarRows*unit.y)
	}
	if g.radius <= 0 {
		panic("chart: no room for the wheel")
	}
	return g
}

func (p plot) slices() []float32 {
	total := 0.0
	for _, v := range p.values {
		total += v[0]
	}
	out := make([]float32, len(p.values)+1)
	sum := 0.0
	for s, v := range p.values {
		sum += v[0]
		if total > 0 {
			out[s+1] = float32(turn * sum / total)
		}
	}
	return out
}

func (p plot) axis(i int) float32 { return turn * float32(i) / float32(p.points) }

func (g wheel) reach(p plot, v float64) float32 {
	return g.radius * float32(v/p.ticks[len(p.ticks)-1])
}

func (c *Chart) spin(r *raster, p plot, m metrics, unit point, cols, hover int) {
	g := p.wheel(float32(r.w), float32(r.h), unit, cols)
	flat := paint{top: 1, bottom: 1, dither: c.Dither}
	if c.Dither {
		flat.top, flat.bottom = konst.DitherFlat, konst.DitherFlat
	}
	sector := func(radius, from, to float32) {
		r.ring = arc(r.ring[:0], g.centre, radius, from, to)
		if g.hole > 0 {
			r.ring = arc(r.ring, g.centre, g.hole, to, from)
		} else {
			r.ring = append(r.ring, g.centre)
		}
		r.polygon(r.ring)
	}
	switch p.kind {
	case Pie, Donut:
		edges := p.slices()
		for s := range p.values {
			if edges[s+1] > edges[s] {
				sector(g.radius, edges[s], turn)
				flat.token, flat.ink = c.Series[s].Color, c.ink(s, solid)
				r.commit(flat)
			}
		}
		if s := hover - 1; s >= 0 && edges[s+1] > edges[s] {
			sector(g.radius+g.grow, edges[s], edges[s+1])
			flat.token, flat.ink = c.Series[s].Color, c.ink(s, solid)
			r.commit(flat)
		}
	case Radial:
		for s, v := range p.values {
			mid := g.radius - g.band*(float32(s)+0.5)
			outer, inner := mid+g.thick/2, mid-g.thick/2
			r.ring = arc(r.ring[:0], g.centre, outer, 0, turn)
			r.lower = arc(r.lower[:0], g.centre, inner, 0, turn)
			r.polygon(r.ring, r.lower)
			r.commit(paint{token: theme.Muted, ink: c.quiet(theme.Muted), top: 1, bottom: 1})
			if v[0] == 0 {
				continue
			}
			sweep := turn * float32(v[0]) / g.largest
			if sweep < turn {
				cap := min(g.thick/2/mid, sweep/2)
				r.round(toward(g.centre, mid, cap), g.thick/2)
				r.round(toward(g.centre, mid, sweep-cap), g.thick/2)
				r.ring = arc(r.ring[:0], g.centre, outer, cap, sweep-cap)
				r.ring = arc(r.ring, g.centre, inner, sweep-cap, cap)
				r.polygon(r.ring)
			} else {
				r.polygon(r.ring, r.lower)
			}
			flat.token, flat.ink = c.Series[s].Color, c.ink(s, solid)
			r.commit(flat)
		}
	case Radar:
		for _, t := range p.ticks[1:] {
			r.ring = r.ring[:0]
			for i := range p.points + 1 {
				r.ring = append(r.ring, toward(g.centre, g.reach(p, t), p.axis(i)))
			}
			r.stroke(r.ring, m.grid/2)
		}
		for i := range p.points {
			r.stroke([]point{g.centre, toward(g.centre, g.radius, p.axis(i))}, m.grid/2)
		}
		r.commit(paint{token: theme.Border, ink: c.quiet(theme.Border), top: 1, bottom: 1})
		for s, series := range p.values {
			r.ring = r.ring[:0]
			for i, v := range series {
				r.ring = append(r.ring, toward(g.centre, g.reach(p, v), p.axis(i)))
			}
			r.polygon(r.ring)
			fill := paint{token: c.Series[s].Color, ink: c.ink(s, half), top: konst.RadarFill, bottom: konst.RadarFill}
			if c.Dither {
				fill.ink, fill.dither = c.ink(s, solid), true
			}
			r.commit(fill)
			if hover > 0 && r.dst != nil {
				r.disc(r.ring[hover-1], m.active)
				r.commit(paint{token: c.Series[s].Color, top: 1, bottom: 1})
			}
		}
	}
}

func (r *raster) round(c point, radius float32) {
	from := max(ceil32((c.y-radius)*konst.Samples-0.5), 0)
	to := min(floor((c.y+radius)*konst.Samples-0.5), r.h*konst.Samples-1)
	for k := from; k <= to; k++ {
		y := (float32(k)+0.5)/konst.Samples - c.y
		d := float32(math.Sqrt(float64(max(radius*radius-y*y, 0))))
		r.add(k, interval{c.x - d, c.x + d})
	}
}

func (c *Chart) aim(p plot, w, h float32, unit point, cols int, q point) int {
	g := p.wheel(w, h, unit, cols)
	d, a := float32(math.Hypot(float64(q.x-g.centre.x), float64(q.y-g.centre.y))), angleOf(g.centre, q)
	switch p.kind {
	case Pie, Donut:
		edges := p.slices()
		if d > g.radius+g.grow || d < g.hole {
			return 0
		}
		for s := range p.values {
			if a >= edges[s] && a < edges[s+1] {
				return s + 1
			}
		}
	case Radial:
		if s := int((g.radius - d) / g.band); d >= g.hole && d <= g.radius {
			return min(s, len(p.values)-1) + 1
		}
	case Radar:
		if d <= g.radius+unit.y {
			return int(math.Round(float64(a/p.axis(1))))%p.points + 1
		}
	}
	return 0
}

func (c *Chart) polar(p plot) twi.Node {
	widths := c.rt.Widths()
	names := make([]string, p.points)
	cols := 0
	for i, l := range p.labels {
		names[i] = text.Sanitize(l, text.RemoveBidi)
		cols = max(cols, widths.Width(names[i]))
	}
	if p.kind == Radar && p.points < konst.RadarAxes {
		panic("chart: a radar needs three labels or more")
	}
	if p.kind != Radar {
		cols = 0
	}
	w, h := c.Width, c.Height-1-konst.LegendGap
	cellUnit := point{1, konst.CellRows}
	c.halfs = halves(c.halfs, w, h)
	c.spin(c.halfs, p, metrics{half: 0.5, grid: 1, active: 1}, cellUnit, cols, 0)
	hover := c.hover
	area := []twi.NodeOption{twi.Class("relative flex flex-col"),
		twi.OnPointerMove(func(e *twi.Event) {
			at := e.Offset()
			q := point{float32(at.X) + 0.5, (float32(at.Y) + 0.5) * konst.CellRows}
			c.point(c.aim(p, float32(w), float32(h*konst.CellRows), cellUnit, cols, q), at)
		}),
		twi.OnPointerLeave(func(*twi.Event) { c.point(0, image.Point{}) }),
		surface(c.key(0), func(dst *image.RGBA, cell image.Point) {
			c.canvas = c.pixels(c.canvas, dst, cell)
			c.spin(c.canvas, p, pixelMetrics(cell), point{float32(cell.X), float32(cell.Y)}, cols, hover)
		}, c.glyphs(c.halfs, nil, image.Rectangle{}), w),
	}
	g := p.wheel(float32(w), float32(h*konst.CellRows), cellUnit, cols)
	switch p.kind {
	case Radar:
		for i, name := range names {
			a := p.axis(i)
			at := toward(g.centre, g.radius+konst.RadarLabel*cellUnit.y, a)
			x, width := int(at.x), widths.Width(name)
			switch s := math.Sin(float64(a)); {
			case s < -konst.RadarSide:
				x -= width + 1
			case s <= konst.RadarSide:
				x -= width / 2
			default:
				x++
			}
			area = append(area, twi.Element(twi.At(max(x, 0), max(int(at.y/cellUnit.y), 0)), twi.Class("text-muted-foreground"), twi.Text(name)))
		}
	case Donut:
		total := 0.0
		for _, v := range p.values {
			total += v[0]
		}
		row := int(math.Round(float64(g.centre.y/cellUnit.y))) - 1
		for i, line := range [2]string{number(total), names[0]} {
			class := "font-bold text-foreground"
			if i > 0 {
				class = "text-muted-foreground"
			}
			x := int(g.centre.x) - widths.Width(line)/2
			area = append(area, twi.Element(twi.At(x, row+i), twi.Class(class), twi.Text(line)))
		}
	}
	if c.hover > 0 {
		i := c.hover - 1
		var rows []entry
		title := ""
		if p.kind == Radar {
			title = names[i]
			for s, series := range c.Series {
				rows = append(rows, entry{series.Color, series.Label, number(p.values[s][i])})
			}
		} else {
			rows = []entry{{c.Series[i].Color, c.Series[i].Label, number(p.values[i][0])}}
		}
		area = append(area, tooltip(widths, title, rows, beside(c.at.X, c.at.X+1, c.at.Y-1, w)))
	}
	return twi.Element(twi.Class("flex flex-row justify-center"), twi.Element(area...))
}
