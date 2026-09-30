package chart

import (
	"slices"
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/chart"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/text"
	"github.com/twind-dev/twind/twi/theme"
)

type Kind uint8

const (
	Bar Kind = iota + 1
	Line
	Area
)

type Series struct {
	Label  string
	Color  theme.Token
	Values []float64
}

type Chart struct {
	Kind          Kind
	Stacked       bool
	Labels        []string
	Series        []Series
	Width, Height int
	rt            *twi.Runtime
	hover         int
	pixels, halfs *raster
}

func New(rt *twi.Runtime) *Chart { return &Chart{rt: rt} }

type cell struct{ glyph, fg, bg string }

func tones(t theme.Token) [4]string {
	switch t {
	case theme.Chart1:
		return [4]string{"text-chart-1", "bg-chart-1", "text-chart-1/30", "bg-chart-1/30"}
	case theme.Chart2:
		return [4]string{"text-chart-2", "bg-chart-2", "text-chart-2/30", "bg-chart-2/30"}
	case theme.Chart3:
		return [4]string{"text-chart-3", "bg-chart-3", "text-chart-3/30", "bg-chart-3/30"}
	case theme.Chart4:
		return [4]string{"text-chart-4", "bg-chart-4", "text-chart-4/30", "bg-chart-4/30"}
	case theme.Chart5:
		return [4]string{"text-chart-5", "bg-chart-5", "text-chart-5/30", "bg-chart-5/30"}
	}
	panic("chart: a series colour is chart-1 to chart-5, not " + t.String())
}

func cellMetrics() metrics { return metrics{margin: konst.CellMargin, half: 0.5, gap: 1} }

func (p plot) row(v float64, h int) int {
	return min(int(p.y(v, float32(h*konst.CellRows), cellMetrics()))/konst.CellRows, h-1)
}

func (c *Chart) cells(p plot, w, h int) []cell {
	c.halfs = reuse(c.halfs, w, h*konst.CellRows)
	r := c.halfs
	r.ink = slices.Grow(r.ink[:0], len(r.cover))[:len(r.cover)]
	clear(r.ink)
	c.draw(r, p, cellMetrics(), 0)
	grid := make([]bool, h)
	for _, t := range p.ticks {
		grid[p.row(t, h)] = true
	}
	fg := func(ink uint8) string { return tones(c.Series[(ink-1)/2].Color)[2*int(1-ink%2)] }
	bg := func(ink uint8) string { return tones(c.Series[(ink-1)/2].Color)[1+2*int(1-ink%2)] }
	out := make([]cell, w*h)
	for y := range h {
		for x := range w {
			top, bottom := r.ink[2*y*w+x], r.ink[(2*y+1)*w+x]
			faded := top%2 == 0
			var at cell
			switch {
			case top == 0 && bottom == 0 && grid[y]:
				at = cell{konst.Grid, "text-border", ""}
			case top == 0 && bottom == 0:
				at = cell{glyph: konst.Blank}
			case top == bottom && grid[y] && faded:
				at = cell{konst.Grid, "text-border", bg(top)}
			case top == bottom:
				at = cell{konst.Full, fg(top), ""}
			case top == 0:
				at = cell{konst.Lower, fg(bottom), ""}
			case bottom == 0:
				at = cell{konst.Upper, fg(top), ""}
			default:
				at = cell{konst.Upper, fg(top), bg(bottom)}
			}
			out[y*w+x] = at
		}
	}
	return out
}

func (c *Chart) Node(options ...twi.NodeOption) twi.Node {
	p := c.model()
	widths := c.rt.Widths()
	ticks, axis := make([]string, len(p.ticks)), 0
	for i, t := range p.ticks {
		ticks[i] = number(t)
		axis = max(axis, widths.Width(ticks[i]))
	}
	w, h := c.Width-axis-1, c.Height-2-konst.LegendGap
	if w < konst.MinPlot || h < konst.MinPlot || p.points == 0 {
		panic("chart: no room for a plot or no points")
	}
	labels := make([]string, h)
	for i, t := range p.ticks {
		labels[p.row(t, h)] = ticks[i]
	}
	yAxis := []twi.NodeOption{twi.Class("flex flex-col text-muted-foreground")}
	for _, l := range labels {
		yAxis = append(yAxis, twi.Text(strings.Repeat(konst.Blank, axis-widths.Width(l))+l))
	}
	grid := c.cells(p, w, h)
	plotArea := []twi.NodeOption{twi.Class("relative flex flex-row")}
	for i := range p.points {
		x0, x1 := i*w/p.points, (i+1)*w/p.points
		rows := []twi.NodeOption{twi.Class("flex flex-col")}
		if i == c.hover-1 {
			rows[0] = twi.Class("flex flex-col bg-muted/50")
		}
		for y := range h {
			runs := []twi.NodeOption{twi.Class("flex flex-row")}
			for x := x0; x < x1; {
				at, end := grid[y*w+x], x+1
				for end < x1 && grid[y*w+end] == at {
					end++
				}
				glyphs := twi.Text(strings.Repeat(at.glyph, end-x))
				if at.fg != "" || at.bg != "" {
					glyphs = twi.Element(twi.Class(at.fg, at.bg), glyphs)
				}
				runs, x = append(runs, glyphs), end
			}
			rows = append(rows, twi.Element(runs...))
		}
		label := widths.Truncate(text.Sanitize(c.Labels[i], text.RemoveBidi), x1-x0)
		plotArea = append(plotArea, twi.Element(twi.Class("flex flex-col"), twi.OnPointerEnter(func() { c.point(i + 1) }),
			twi.Element(rows...),
			twi.Element(twi.Class("flex flex-row justify-center text-muted-foreground"), twi.Text(label)),
		))
	}
	if c.hover > 0 {
		plotArea = append(plotArea, c.tooltip(widths, c.hover-1, w))
	}
	legend := []twi.NodeOption{twi.Class("flex flex-row justify-center gap-2 mt-1")}
	for _, s := range c.Series {
		legend = append(legend, twi.Element(twi.Class("flex flex-row gap-1"), twi.Element(twi.Class(tones(s.Color)[0]), twi.Text(konst.Key)), twi.Text(text.Sanitize(s.Label, text.RemoveBidi))))
	}
	return twi.Element(append([]twi.NodeOption{twi.Class("flex flex-col whitespace-pre"), twi.OnPointerLeave(func() { c.point(0) }),
		twi.Element(twi.Class("flex flex-row gap-1"), twi.Element(yAxis...), twi.Element(plotArea...)),
		twi.Element(legend...),
	}, options...)...)
}

func (c *Chart) point(hover int) {
	c.hover = hover
	c.rt.Invalidate()
}

func (c *Chart) tooltip(widths text.Widths, i, w int) twi.Node {
	label := text.Sanitize(c.Labels[i], text.RemoveBidi)
	width := widths.Width(label)
	tip := []twi.NodeOption{twi.Class("z-50 flex flex-col rounded-md border bg-popover px-1 text-popover-foreground shadow-md"), twi.Element(twi.Class("font-medium"), twi.Text(label))}
	for _, s := range c.Series {
		name, value := text.Sanitize(s.Label, text.RemoveBidi), number(s.Values[i])
		width = max(width, widths.Width(konst.Key)+1+widths.Width(name)+konst.TooltipGap+widths.Width(value))
		tip = append(tip, twi.Element(twi.Class("flex flex-row justify-between gap-2"),
			twi.Element(twi.Class("flex flex-row gap-1"), twi.Element(twi.Class(tones(s.Color)[0]), twi.Text(konst.Key)), twi.Element(twi.Class("text-muted-foreground"), twi.Text(name))),
			twi.Element(twi.Class("font-medium"), twi.Text(value)),
		))
	}
	width += konst.TooltipChrome
	x0, x1 := i*w/len(c.Labels), (i+1)*w/len(c.Labels)
	x := x1 + 1
	if x+width > w {
		x = max(x0-1-width, 0)
	}
	return twi.Element(append([]twi.NodeOption{twi.At(x, 0)}, tip...)...)
}
