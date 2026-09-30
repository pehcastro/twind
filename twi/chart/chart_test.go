package chart

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/chart"
	paintkonst "github.com/twind-dev/twind/internal/konst/paint"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/theme"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen -o twir_gen_test.go -func styles

func threeSeries(kind Kind, stacked bool) func(rt *twi.Runtime) *Chart {
	return func(rt *twi.Runtime) *Chart {
		c := New(rt)
		c.Kind, c.Stacked, c.Width, c.Height = kind, stacked, 96, 26
		c.Labels = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
		c.Series = []Series{
			{Label: "Desktop", Color: theme.Chart1, Values: visitors},
			{Label: "Mobile", Color: theme.Chart2, Values: []float64{80, 200, 120, 190, 130, 140, 90, 160, 110, 150, 100, 170}},
			{Label: "Tablet", Color: theme.Chart3, Values: []float64{40, 60, 55, 30, 70, 65, 45, 80, 50, 75, 35, 60}},
		}
		return c
	}
}

func chartDriver(t *testing.T, build func(rt *twi.Runtime) *Chart) (*drive.Driver, *Chart) {
	t.Helper()
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	var c *Chart
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		c = build(rt)
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col h-full p-1 bg-background text-foreground"), c.Node())
		}
	}, drive.Size(100, 30), drive.Styles(sheet))
	t.Cleanup(func() {
		if err := d.Err(); err != nil {
			t.Error(err)
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	return d, c
}

func TestCellsUseOnlyConsoleGlyphs(t *testing.T) {
	for _, k := range []struct {
		name    string
		kind    Kind
		stacked bool
	}{{"bar", Bar, false}, {"stacked bar", Bar, true}, {"line", Line, false}, {"stacked area", Area, true}} {
		d, _ := chartDriver(t, threeSeries(k.kind, k.stacked))
		frame := d.Frame().Text()
		t.Logf("%s chart, 100x30, no graphics:\n%s", k.name, frame)
		if !strings.ContainsAny(frame, konst.Upper+konst.Lower+konst.Full) {
			t.Errorf("%s chart drew no block cells", k.name)
		}
		if i := strings.IndexFunc(frame, func(r rune) bool {
			return r >= utf8.RuneSelf && !strings.ContainsRune(paintkonst.ConsoleGlyphs, r)
		}); i >= 0 {
			r, _ := utf8.DecodeRuneInString(frame[i:])
			t.Errorf("%s chart draws %q, outside the console-safe set", k.name, r)
		}
	}
}

func TestCellsDrawBarsInHalfCells(t *testing.T) {
	c := &Chart{Kind: Bar, Labels: []string{"a", "b"}, Series: []Series{{Color: theme.Chart1, Values: []float64{4, 2}}}}
	cells := c.cells(c.model(), 5, 5)
	want := []string{"▄▄───", "██───", "██─▄▄", "██─██", "██─██"}
	for y, row := range want {
		var got strings.Builder
		for x := range 5 {
			got.WriteString(cells[y*5+x].glyph)
		}
		if got.String() != row {
			t.Errorf("row %d = %q, want %q (bar 4 from half row 1, bar 2 from half row 5, a cell between bands, grid where empty)", y, got.String(), row)
		}
	}
	if at := cells[0]; at.fg != "text-chart-1" || at.bg != "" {
		t.Errorf("bar top cell classes %q %q, want text-chart-1 on nothing", at.fg, at.bg)
	}
}

func TestCellsKeepSteepLinesConnected(t *testing.T) {
	c := &Chart{Kind: Line, Labels: []string{"a", "b", "c"}, Series: []Series{{Color: theme.Chart1, Values: []float64{0, 100, 0}}}}
	const w, h = 30, 10
	cells := c.cells(c.model(), w, h)
	for x := 5; x < 25; x++ {
		lit := 0
		for y := range h {
			if g := cells[y*w+x].glyph; g != konst.Blank && g != konst.Grid {
				lit++
			}
		}
		if lit == 0 {
			t.Errorf("column %d has no line cell between the first and last point", x)
		}
	}
	for y := range h {
		lit := 0
		for x := 5; x <= 15; x++ {
			if g := cells[y*w+x].glyph; g != konst.Blank && g != konst.Grid {
				lit++
			}
		}
		if lit == 0 {
			t.Errorf("row %d has no cell of the rising line from 5 to 15", y)
		}
	}
}

func TestTooltipNamesThePointUnderThePointer(t *testing.T) {
	d, c := chartDriver(t, threeSeries(Bar, false))
	lines := strings.Split(d.Frame().Text(), "\n")
	axis := len(lines) - 1
	for axis >= 0 && !strings.Contains(lines[axis], "Jan") {
		axis--
	}
	for _, k := range []struct {
		label, desktop, mobile string
		left                   bool
	}{{"Jan", "186", "80", false}, {"Apr", "73", "190", false}, {"Dec", "230", "170", true}} {
		x := strings.Index(lines[axis], k.label)
		x = utf8.RuneCountInString(lines[axis][:x]) + 1
		d.Move(x, axis-3)
		frame := d.Frame().Text()
		if c.hover == 0 || c.Labels[c.hover-1] != k.label {
			t.Fatalf("pointer over %s at %d,%d hovers point %d:\n%s", k.label, x, axis-3, c.hover, frame)
		}
		row := regexp.MustCompile(`■ Desktop +` + k.desktop + `\b`)
		if !row.MatchString(frame) || !regexp.MustCompile(`■ Mobile +`+k.mobile+`\b`).MatchString(frame) {
			t.Errorf("tooltip over %s does not name Desktop %s and Mobile %s:\n%s", k.label, k.desktop, k.mobile, frame)
		}
		for _, line := range strings.Split(frame, "\n") {
			if loc := row.FindStringIndex(line); loc != nil {
				at := utf8.RuneCountInString(line[:loc[0]])
				if k.left != (at < x) {
					t.Errorf("tooltip over %s starts at column %d with the pointer at %d: want it on the %v side", k.label, at, x, map[bool]string{true: "left", false: "right"}[k.left])
				}
			}
		}
		t.Logf("pointer over %s, 100x30:\n%s", k.label, frame)
	}
	d.Move(0, 0)
	if frame := d.Frame().Text(); c.hover != 0 || regexp.MustCompile(`Desktop +\d`).MatchString(frame) {
		t.Errorf("tooltip stays after the pointer leaves the chart (hover %d):\n%s", c.hover, frame)
	}
}
