package chart

import (
	"encoding/binary"
	"hash/fnv"
	"image"
	"math"
	"strconv"
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
	Pie
	Donut
	Radial
	Radar
	Map
)

type Series struct {
	Label    string
	Color    theme.Token
	Negative theme.Token
	Values   []float64
}

type Chart struct {
	Kind              Kind
	Stacked           bool
	Horizontal        bool
	Step              bool
	Labelled          bool
	Dither            bool
	Brush             bool
	From, To          int
	Labels            []string
	Series            []Series
	Width, Height     int
	rt                *twi.Runtime
	hover             int
	at                image.Point
	anchor            int
	dragged           bool
	canvas, halfs     *raster
	strip, stripHalfs *raster
	world             *atlas
}

func New(rt *twi.Runtime) *Chart { return &Chart{rt: rt} }

type level uint8

const (
	solid level = iota
	half
	faint
	below
	levels
)

type tone struct {
	fg, bg string
	faded  bool
}

func shade(t theme.Token, l level) tone {
	var classes [3][2]string
	switch t {
	case theme.Chart1:
		classes = [3][2]string{{"text-chart-1", "bg-chart-1"}, {"text-chart-1/60", "bg-chart-1/60"}, {"text-chart-1/30", "bg-chart-1/30"}}
	case theme.Chart2:
		classes = [3][2]string{{"text-chart-2", "bg-chart-2"}, {"text-chart-2/60", "bg-chart-2/60"}, {"text-chart-2/30", "bg-chart-2/30"}}
	case theme.Chart3:
		classes = [3][2]string{{"text-chart-3", "bg-chart-3"}, {"text-chart-3/60", "bg-chart-3/60"}, {"text-chart-3/30", "bg-chart-3/30"}}
	case theme.Chart4:
		classes = [3][2]string{{"text-chart-4", "bg-chart-4"}, {"text-chart-4/60", "bg-chart-4/60"}, {"text-chart-4/30", "bg-chart-4/30"}}
	case theme.Chart5:
		classes = [3][2]string{{"text-chart-5", "bg-chart-5"}, {"text-chart-5/60", "bg-chart-5/60"}, {"text-chart-5/30", "bg-chart-5/30"}}
	case theme.Muted:
		classes = [3][2]string{{"text-muted", "bg-muted"}, {"text-muted", "bg-muted"}, {"text-muted", "bg-muted"}}
	case theme.Border:
		classes = [3][2]string{{"text-border", "bg-border"}, {"text-border", "bg-border"}, {"text-border", "bg-border"}}
	default:
		panic("chart: a series colour is chart-1 to chart-5, not " + t.String())
	}
	if l == below {
		l = solid
	}
	return tone{classes[l][0], classes[l][1], l == faint}
}

func (c *Chart) palette() []tone {
	tones := make([]tone, 0, int(levels)*len(c.Series)+2)
	for l := range levels {
		for _, s := range c.Series {
			t := s.Color
			if l == below && s.Negative != 0 {
				t = s.Negative
			}
			tones = append(tones, shade(t, l))
		}
	}
	return append(tones, shade(theme.Muted, solid), shade(theme.Border, solid))
}

func (c *Chart) ink(s int, l level) uint8 { return uint8(int(l)*len(c.Series) + s + 1) }

func (c *Chart) quiet(t theme.Token) uint8 {
	if t == theme.Border {
		return uint8(int(levels)*len(c.Series) + 2)
	}
	return uint8(int(levels)*len(c.Series) + 1)
}

func (c *Chart) negative(s int) theme.Token {
	if t := c.Series[s].Negative; t != 0 {
		return t
	}
	return c.Series[s].Color
}

type cell struct{ glyph, fg, bg string }

func (c *Chart) glyphs(r *raster, grid []bool, tint image.Rectangle) []cell {
	tones := c.palette()
	w, h := r.w, r.h/konst.CellRows
	out := make([]cell, w*h)
	for y := range h {
		for x := range w {
			top, bottom := r.ink[2*y*w+x], r.ink[(2*y+1)*w+x]
			var at cell
			switch {
			case top == 0 && bottom == 0 && grid != nil && grid[y]:
				at = cell{konst.Grid, "text-border", ""}
			case top == 0 && bottom == 0:
				at = cell{glyph: konst.Blank}
			case top == bottom && grid != nil && grid[y] && tones[top-1].faded:
				at = cell{konst.Grid, "text-border", tones[top-1].bg}
			case top == bottom:
				at = cell{konst.Full, tones[top-1].fg, ""}
			case top == 0:
				at = cell{konst.Lower, tones[bottom-1].fg, ""}
			case bottom == 0:
				at = cell{konst.Upper, tones[top-1].fg, ""}
			default:
				at = cell{konst.Upper, tones[top-1].fg, tones[bottom-1].bg}
			}
			if image.Pt(x, y).In(tint) && at.bg == "" {
				at.bg = "bg-muted/50"
			}
			out[y*w+x] = at
		}
	}
	return out
}

func halves(r *raster, w, h int) *raster {
	r = reuse(r, w, h*konst.CellRows)
	r.dot = 1
	if len(r.ink) != len(r.cover) {
		r.ink = make([]uint8, len(r.cover))
	}
	clear(r.ink)
	return r
}

func surface(key uint64, paint func(dst *image.RGBA, cell image.Point), grid []cell, w int, options ...twi.NodeOption) twi.Node {
	rows := append([]twi.NodeOption{twi.Class("flex flex-col"), twi.Canvas(key, paint)}, options...)
	for y := range len(grid) / w {
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
		rows = append(rows, twi.Element(runs...))
	}
	return twi.Element(rows...)
}

func (c *Chart) Node(options ...twi.NodeOption) twi.Node {
	p := c.model()
	var body twi.Node
	switch p.kind {
	case Bar, Line, Area:
		body = c.cartesian(p)
	case Pie, Donut, Radial, Radar:
		body = c.polar(p)
	case Map:
		body = c.choropleth(p)
	default:
		panic("chart: unknown kind " + strconv.Itoa(int(p.kind)))
	}
	legend := []twi.NodeOption{twi.Class("flex flex-row justify-center gap-2 mt-1")}
	for _, s := range c.Series {
		legend = append(legend, twi.Element(twi.Class("flex flex-row gap-1"), twi.Element(twi.Class(shade(s.Color, solid).fg), twi.Text(konst.Key)), twi.Text(text.Sanitize(s.Label, text.RemoveBidi))))
	}
	if p.kind == Map {
		lo, hi := p.extent()
		keys := []twi.NodeOption{twi.Class("flex flex-row")}
		for _, l := range []level{faint, half, solid} {
			keys = append(keys, twi.Element(twi.Class(shade(c.Series[0].Color, l).fg), twi.Text(konst.Key)))
		}
		legend = append(legend, twi.Element(twi.Class("flex flex-row gap-1 text-muted-foreground"), twi.Text(number(lo)), twi.Element(keys...), twi.Text(number(hi))))
	}
	return twi.Element(append([]twi.NodeOption{twi.Class("flex flex-col whitespace-pre"), body, twi.Element(legend...)}, options...)...)
}

func (c *Chart) key(salt uint64) uint64 {
	tokens := c.rt.Theme().Tokens
	h := fnv.New64a()
	var word []byte
	put := func(v uint64) {
		word = binary.LittleEndian.AppendUint64(word[:0], v)
		h.Write(word)
	}
	tone := func(t theme.Token) {
		if t == 0 {
			put(0)
			return
		}
		rgb := tokens[t].RGBA
		put(uint64(rgb.R)<<24 | uint64(rgb.G)<<16 | uint64(rgb.B)<<8 | uint64(rgb.A))
	}
	flags := 0
	for i, on := range []bool{c.Stacked, c.Horizontal, c.Step, c.Labelled, c.Dither, c.Brush} {
		if on {
			flags |= 1 << i
		}
	}
	for _, v := range []int{int(c.Kind), flags, len(c.Labels), c.From, c.To, c.hover} {
		put(uint64(v))
	}
	put(salt)
	tone(theme.Border)
	tone(theme.Muted)
	for _, s := range c.Series {
		tone(s.Color)
		tone(s.Negative)
		for _, v := range s.Values {
			put(math.Float64bits(v))
		}
	}
	return h.Sum64()
}

func (c *Chart) colours() func(theme.Token) color.RGBA {
	tokens := c.rt.Theme().Tokens
	return func(t theme.Token) color.RGBA { return tokens[t].RGBA }
}

func (c *Chart) point(hover int, at image.Point) {
	if hover != c.hover {
		c.hover, c.at = hover, at
		c.rt.Invalidate()
	}
}

type entry struct {
	color       theme.Token
	name, value string
}

func tooltip(widths text.Widths, title string, rows []entry, place func(width int) image.Point) twi.Node {
	width := widths.Width(title)
	tip := []twi.NodeOption{twi.Class("z-50 flex flex-col rounded-md border bg-popover px-1 text-popover-foreground shadow-md")}
	if title != "" {
		tip = append(tip, twi.Element(twi.Class("font-medium"), twi.Text(title)))
	}
	for _, e := range rows {
		name := text.Sanitize(e.name, text.RemoveBidi)
		width = max(width, widths.Width(konst.Key)+1+widths.Width(name)+konst.TooltipGap+widths.Width(e.value))
		tip = append(tip, twi.Element(twi.Class("flex flex-row justify-between gap-2"),
			twi.Element(twi.Class("flex flex-row gap-1"), twi.Element(twi.Class(shade(e.color, solid).fg), twi.Text(konst.Key)), twi.Element(twi.Class("text-muted-foreground"), twi.Text(name))),
			twi.Element(twi.Class("font-medium"), twi.Text(e.value)),
		))
	}
	at := place(width + konst.TooltipChrome)
	return twi.Element(append([]twi.NodeOption{twi.At(at.X, at.Y)}, tip...)...)
}

func beside(x0, x1, y, w int) func(width int) image.Point {
	return func(width int) image.Point {
		x := x1 + 1
		if x+width > w {
			x = max(x0-1-width, 0)
		}
		return image.Pt(x, max(y, 0))
	}
}
