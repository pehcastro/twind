package chart

import (
	"encoding/binary"
	"hash/fnv"
	"image"
	"math"
	"slices"
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/chart"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
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

func (p plot) halves(h int) metrics { return p.rows(metrics{half: 0.5, gap: 1}, h, konst.CellRows) }

func (p plot) row(v float64, h int) int { return int(p.y(v, p.halves(h))) / konst.CellRows }

func (c *Chart) column(i, w int) (x0, x1 int) {
	return i * w / len(c.Labels), (i + 1) * w / len(c.Labels)
}

func (c *Chart) cells(p plot, w, h int) []cell {
	c.halfs = reuse(c.halfs, w, h*konst.CellRows)
	r := c.halfs
	r.ink = slices.Grow(r.ink[:0], len(r.cover))[:len(r.cover)]
	clear(r.ink)
	c.draw(r, p, p.halves(h), 0)
	grid := make([]bool, h)
	for _, t := range p.ticks {
		grid[p.row(t, h)] = true
	}
	x0, x1 := 0, 0
	if c.hover > 0 {
		x0, x1 = c.column(c.hover-1, w)
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
			if x >= x0 && x < x1 && at.bg == "" {
				at.bg = "bg-muted/50"
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
	tokens, hover := c.rt.Theme().Tokens, c.hover
	plot := []twi.NodeOption{twi.Class("flex flex-col"), twi.Canvas(c.key(&tokens, hover), func(dst *image.RGBA, cell image.Point) {
		c.paint(dst, p, p.rows(pixelMetrics(cell), h, float32(cell.Y)), hover, func(t theme.Token) color.RGBA { return tokens[t].RGBA })
	})}
	grid := c.cells(p, w, h)
	for y := range h {
		runs := []twi.NodeOption{twi.Class("flex flex-row")}
		for x := 0; x < w; {
			at, end := grid[y*w+x], x+1
			for end < w && grid[y*w+end] == at {
				end++
			}
			glyphs := twi.Text(strings.Repeat(at.glyph, end-x))
			if at.fg != "" || at.bg != "" {
				glyphs = twi.Element(twi.Class(at.fg, at.bg), glyphs)
			}
			runs, x = append(runs, glyphs), end
		}
		plot = append(plot, twi.Element(runs...))
	}
	var xAxis strings.Builder
	for i := range p.points {
		x0, x1 := c.column(i, w)
		label := widths.Truncate(text.Sanitize(c.Labels[i], text.RemoveBidi), max(x1-x0-1, 1))
		pad := x1 - x0 - widths.Width(label)
		xAxis.WriteString(strings.Repeat(konst.Blank, pad/2) + label + strings.Repeat(konst.Blank, pad-pad/2))
	}
	area := []twi.NodeOption{twi.Class("relative flex flex-col"),
		twi.OnPointerMove(func(e *twi.Event) { c.point(c.under(e.Offset().X, w)) }),
		twi.OnPointerLeave(func() { c.point(0) }),
		twi.Element(plot...),
		twi.Element(twi.Class("text-muted-foreground"), twi.Text(xAxis.String())),
	}
	if c.hover > 0 {
		area = append(area, c.tooltip(widths, c.hover-1, w))
	}
	legend := []twi.NodeOption{twi.Class("flex flex-row justify-center gap-2 mt-1")}
	for _, s := range c.Series {
		legend = append(legend, twi.Element(twi.Class("flex flex-row gap-1"), twi.Element(twi.Class(tones(s.Color)[0]), twi.Text(konst.Key)), twi.Text(text.Sanitize(s.Label, text.RemoveBidi))))
	}
	return twi.Element(append([]twi.NodeOption{twi.Class("flex flex-col whitespace-pre"),
		twi.Element(twi.Class("flex flex-row gap-1"), twi.Element(yAxis...), twi.Element(area...)),
		twi.Element(legend...),
	}, options...)...)
}

func (c *Chart) key(tokens *theme.Tokens, hover int) uint64 {
	h := fnv.New64a()
	var word []byte
	put := func(v uint64) {
		word = binary.LittleEndian.AppendUint64(word[:0], v)
		h.Write(word)
	}
	tone := func(t theme.Token) {
		rgb := tokens[t].RGBA
		put(uint64(rgb.R)<<24 | uint64(rgb.G)<<16 | uint64(rgb.B)<<8 | uint64(rgb.A))
	}
	put(uint64(c.Kind))
	put(uint64(len(c.Labels)))
	put(uint64(hover))
	if c.Stacked {
		put(1)
	}
	tone(theme.Border)
	tone(theme.Muted)
	for _, s := range c.Series {
		tone(s.Color)
		for _, v := range s.Values {
			put(math.Float64bits(v))
		}
	}
	return h.Sum64()
}

func (c *Chart) under(x, w int) int {
	for i := range c.Labels {
		if _, x1 := c.column(i, w); x < x1 {
			return i + 1
		}
	}
	return 0
}

func (c *Chart) point(hover int) {
	if hover != c.hover {
		c.hover = hover
		c.rt.Invalidate()
	}
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
	x0, x1 := c.column(i, w)
	x := x1 + 1
	if x+width > w {
		x = max(x0-1-width, 0)
	}
	return twi.Element(append([]twi.NodeOption{twi.At(x, 0)}, tip...)...)
}
