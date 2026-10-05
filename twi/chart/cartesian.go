package chart

import (
	"image"
	"strconv"
	"strings"

	konst "github.com/pehcastro/twind/internal/konst/chart"
	stylekonst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/text"
	"github.com/pehcastro/twind/twi/theme"
)

func (c *Chart) cartesian(p plot) twi.Node {
	widths := c.rt.Widths()
	axis, room := 0, 0
	names := make([]string, p.points)
	for i, l := range p.labels {
		names[i] = text.Sanitize(l, text.RemoveBidi)
	}
	ticks := make([]string, len(p.ticks))
	for i, t := range p.ticks {
		ticks[i] = number(t)
	}
	if p.horizontal {
		for i := range names {
			axis = max(axis, widths.Width(names[i]))
			for s := range p.top {
				if c.Labelled {
					room = max(room, widths.Width(number(p.top[s][i]))+konst.LabelGap)
				}
			}
		}
	} else {
		for _, t := range ticks {
			axis = max(axis, widths.Width(t))
		}
	}
	w, h := c.Width-axis-1, c.Height-1-konst.LegendGap
	if !p.horizontal {
		h--
	}
	if c.Brush {
		h -= konst.BrushRows + 1
	}
	if c.Labelled && p.kind == Bar && !p.horizontal {
		h = p.fit(h)
	}
	if w-room < konst.MinPlot || h-p.lead < konst.MinPlot {
		panic("chart: no room for a plot")
	}
	m := p.halves(h)
	labels := make([]string, h)
	if p.horizontal {
		m = metrics{top: float32(w - room), half: 0.5, gap: 1, dot: 1}
		for i, name := range names {
			lo, hi := span(i, p.points, h*konst.CellRows)
			labels[(lo+hi)/2/konst.CellRows] = name
		}
	} else {
		for i, t := range p.ticks {
			labels[p.row(t, h)] = ticks[i]
		}
	}
	yAxis := []twi.NodeOption{twi.Class("flex flex-col text-muted-foreground")}
	for _, l := range labels {
		yAxis = append(yAxis, twi.Text(strings.Repeat(konst.Blank, axis-widths.Width(l))+l))
	}
	var tint image.Rectangle
	if c.hover > 0 && p.horizontal {
		lo, hi := span(c.hover-1, p.points, h*konst.CellRows)
		tint = image.Rect(0, lo/konst.CellRows, w, (hi+1)/konst.CellRows)
	} else if c.hover > 0 {
		lo, hi := span(c.hover-1, p.points, w)
		tint = image.Rect(lo, 0, hi, h)
	}
	hover := c.hover
	plot := surface(c.key(0), func(dst *image.RGBA, cell image.Point) {
		m := pixelMetrics(cell)
		if p.horizontal {
			m.top = float32((w - room) * cell.X)
		} else {
			m = p.rows(m, h, float32(cell.Y))
		}
		c.canvas = c.pixels(c.canvas, dst, cell)
		c.trace(c.canvas, p, m, hover)
	}, c.cells(p, m, w, h, tint), w)
	area := []twi.NodeOption{twi.Class("relative flex flex-col"),
		twi.OnPointerMove(func(e *twi.Event) {
			at := e.Offset()
			if p.horizontal {
				c.point(under(at.Y*konst.CellRows+1, p.points, h*konst.CellRows), at)
				return
			}
			c.point(under(at.X, p.points, w), at)
		}),
		twi.OnPointerLeave(func() { c.point(0, image.Point{}) }),
		plot,
	}
	if !p.horizontal {
		area = append(area, twi.Element(twi.Class("text-muted-foreground"), twi.Text(xAxis(widths, names, w))))
	}
	if c.Labelled {
		area = append(area, values(p, m, widths, w, h)...)
	}
	if c.Brush {
		area = append(area, c.brush(w))
	}
	if c.hover > 0 {
		i := c.hover - 1
		rows := make([]entry, len(c.Series))
		for s, series := range c.Series {
			rows[s] = entry{series.Color, series.Label, number(p.values[s][i])}
		}
		place := beside(c.at.X, c.at.X+1, c.at.Y-1, w)
		if !p.horizontal {
			lo, hi := span(i, p.points, w)
			place = beside(lo, hi, 0, w)
		}
		area = append(area, tooltip(widths, names[i], rows, place))
	}
	return twi.Element(twi.Class("flex flex-row gap-1"), twi.Element(yAxis...), twi.Element(area...))
}

func (c *Chart) cells(p plot, m metrics, w, h int, tint image.Rectangle) []cell {
	c.halfs = halves(c.halfs, w, h)
	c.trace(c.halfs, p, m, 0)
	var grid []bool
	if !p.horizontal {
		grid = make([]bool, h)
		for _, t := range p.ticks {
			grid[p.row(t, h)] = true
		}
	}
	return c.glyphs(c.halfs, grid, tint)
}

func xAxis(widths text.Widths, names []string, w int) string {
	var out strings.Builder
	col := 0
	for i, name := range names {
		lo, hi := span(i, len(names), w)
		label := widths.Truncate(name, w)
		width := widths.Width(label)
		start := min(max(lo+(hi-lo-width)/2, 0), w-width)
		if col > 0 && start <= col {
			continue
		}
		out.WriteString(strings.Repeat(konst.Blank, start-col) + label)
		col = start + width
	}
	return out.String()
}

func values(p plot, m metrics, widths text.Widths, w, h int) []twi.NodeOption {
	length := float32(w)
	if p.horizontal {
		length = float32(h * konst.CellRows)
	}
	var out []twi.NodeOption
	last := len(p.top) - 1
	for s := range p.top {
		if p.stacked && s != last {
			continue
		}
		for i, v := range p.top[s] {
			label := number(v)
			width := widths.Width(label)
			lo, hi := p.bar(i, s, length, m)
			end := p.y(v, m)
			x, y := int((lo+hi)/2)-width/2, int(end)/konst.CellRows-1
			switch {
			case p.horizontal:
				x, y = ceil32(end)+konst.LabelGap, int((lo+hi)/2)/konst.CellRows
			case v < 0:
				y = (ceil32(end) + 1) / konst.CellRows
			}
			out = append(out, twi.Element(twi.At(max(x, 0), max(y, 0)), twi.Class("text-foreground"), twi.Text(label)))
		}
	}
	return out
}

func (c *Chart) trace(r *raster, p plot, m metrics, hover int) {
	w, h := float32(r.w), float32(r.h)
	length := w
	if p.horizontal {
		length = h
	}
	band := length / float32(p.points)
	if hover > 0 && p.kind == Bar {
		lo, hi := float32(hover-1)*band, float32(hover)*band
		if p.horizontal {
			r.span(0, w, func(float32) (float32, float32) { return lo, hi })
		} else {
			r.span(lo, hi, func(float32) (float32, float32) { return 0, h })
		}
		r.commit(paint{token: theme.Muted, top: konst.CursorAlpha, bottom: konst.CursorAlpha})
	}
	if m.grid > 0 && !p.horizontal && r.dst != nil {
		for _, t := range p.ticks {
			y := min(max(round(p.y(t, m)-m.grid/2), 0), h-m.grid)
			r.span(0, w, func(float32) (float32, float32) { return y, y + m.grid })
			r.commit(paint{token: theme.Border, top: konst.GridAlpha, bottom: konst.GridAlpha})
		}
	}
	if hover > 0 && p.kind != Bar && r.dst != nil {
		x := round((float32(hover) - 0.5) * band)
		r.span(x, x+m.grid, func(float32) (float32, float32) { return 0, h })
		r.commit(paint{token: theme.Border, top: 1, bottom: 1})
	}
	if p.kind == Area && p.points > 1 {
		c.areas(r, p, m)
	}
	last := len(c.Series) - 1
	for s, series := range c.Series {
		switch p.kind {
		case Bar:
			for i, v := range p.top[s] {
				lo, hi := p.bar(i, s, length, m)
				end, base := p.y(v, m), p.y(p.base[s][i], m)
				far, near := m.radius, m.radius
				if p.stacked && s != last {
					far = 0
				}
				if p.stacked && s != 0 {
					near = 0
				}
				fill := paint{token: series.Color, ink: c.ink(s, solid), top: 1, bottom: 1}
				if p.values[s][i] < 0 {
					fill.token, fill.ink = c.negative(s), c.ink(s, below)
				}
				if p.horizontal {
					left, right := near, far
					if end < base {
						end, base, left, right = base, end, far, near
					}
					r.bar(base, end, lo, hi, corners{left, right, right, left})
					fill.top, fill.bottom = c.dithered(konst.DitherFlat, konst.DitherFlat)
				} else {
					if end > base {
						end, base, far, near = base, end, near, far
					}
					r.bar(lo, hi, end, base, corners{far, far, near, near})
					fill.top, fill.bottom = c.dithered(1, konst.DitherFloor)
					fill.from, fill.to = m.top, p.y(0, m)
				}
				fill.dither = c.Dither
				r.commit(fill)
			}
		case Line, Area:
			r.points = p.series(s, w, m, r.points)
			r.curve = p.curve(r.curve, r.points)
			r.stroke(r.curve, m.half)
			solid := paint{token: series.Color, ink: c.ink(s, solid), top: 1, bottom: 1}
			r.commit(solid)
			if hover > 0 && m.active > 0 && r.dst != nil {
				r.disc(r.points[hover-1], m.active)
				r.commit(solid)
			}
		default:
			panic("chart: kind " + strconv.Itoa(int(p.kind)) + " is not a cartesian kind")
		}
	}
}

func (c *Chart) dithered(top, bottom float32) (float32, float32) {
	if c.Dither {
		return top, bottom
	}
	return 1, 1
}

func (p plot) curve(out, pts []point) []point {
	if p.step {
		return steps(out, pts)
	}
	return natural(out, pts)
}

func (c *Chart) areas(r *raster, p plot, m metrics) {
	h := float32(r.h)
	for s, series := range c.Series {
		r.points = p.series(s, float32(r.w), m, r.points)
		r.curve = p.curve(r.curve, r.points)
		if p.stacked && s > 0 {
			r.under = p.series(s-1, float32(r.w), m, r.under)
			r.lower = p.curve(r.lower, r.under)
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
		fill := paint{token: series.Color, ink: c.ink(s, faint), top: konst.FillTop * konst.FillOpacity, bottom: konst.FillBottom * konst.FillOpacity, from: top, to: bottom}
		if c.Dither {
			fill.ink, fill.top, fill.bottom, fill.dither = c.ink(s, solid), konst.FillTop, konst.FillBottom, true
		}
		r.commit(fill)
	}
}

func (c *Chart) brush(w int) twi.Node {
	full := c.window(0, 0)
	full.kind, full.lead = Area, 0
	n := full.points
	from, to := c.From, c.To
	if to == 0 {
		to = n
	}
	c.stripHalfs = halves(c.stripHalfs, w, konst.BrushRows)
	r := c.stripHalfs
	c.trace(r, full, metrics{top: 0.5, bottom: konst.BrushRows*konst.CellRows - 0.5, half: 0.5, gap: 1, dot: 1}, 0)
	x0, _ := span(from, n, w)
	_, x1 := span(to-1, n, w)
	grid := c.glyphs(r, nil, image.Rect(x0, 0, x1, konst.BrushRows))
	pick := func(e *twi.Event) int { return under(min(max(e.Offset().X, 0), w-1), n, w) }
	return surface(c.key(konst.BrushSalt), func(dst *image.RGBA, cell image.Point) {
		c.strip = c.pixels(c.strip, dst, cell)
		r, m := c.strip, pixelMetrics(cell)
		m.top, m.bottom, m.grid = m.half+1, float32(dst.Rect.Dy())-m.half-1, 0
		W, H := float32(r.w), float32(r.h)
		lo, hi := float32(from)*W/float32(n), float32(to)*W/float32(n)
		whole := func(float32) (float32, float32) { return 0, H }
		r.span(lo, hi, whole)
		r.commit(paint{token: theme.Muted, top: 1, bottom: 1})
		c.trace(r, full, m, 0)
		r.span(0, lo, whole)
		r.span(hi, W, whole)
		r.commit(paint{top: konst.BrushDim, bottom: konst.BrushDim, erase: true})
		edge := max(1, round(konst.BrushEdge*float32(cell.Y)/stylekonst.RemPixels))
		r.span(lo, lo+edge, whole)
		r.span(hi-edge, hi, whole)
		r.commit(paint{token: theme.Border, top: 1, bottom: 1})
	}, grid, w, twi.Class("mt-1"),
		twi.OnPointerDown(func(e *twi.Event) {
			e.PreventDefault()
			c.anchor, c.dragged = pick(e), false
		}),
		twi.OnPointerMove(func(e *twi.Event) {
			e.StopPropagation()
			i := pick(e)
			if c.anchor == 0 || i == c.anchor && !c.dragged {
				return
			}
			c.dragged = true
			c.zoom(min(i, c.anchor)-1, max(i, c.anchor), n)
		}),
		twi.OnPointerUp(func(*twi.Event) {
			if c.anchor != 0 && !c.dragged {
				c.zoom(0, n, n)
			}
			c.anchor = 0
		}),
	)
}

func (c *Chart) zoom(from, to, n int) {
	if to-from < konst.MinWindow {
		to = min(from+konst.MinWindow, n)
		from = max(to-konst.MinWindow, 0)
	}
	if to == n {
		to = 0
	}
	c.From, c.To, c.hover = from, to, 0
	c.rt.Invalidate()
}
