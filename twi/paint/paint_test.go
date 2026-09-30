package paint

import (
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
)

var (
	zinc950 = color.RGBA{R: 9, G: 9, B: 11, A: 255}
	zinc800 = color.RGBA{R: 39, G: 39, B: 42, A: 255}
	zinc100 = color.RGBA{R: 244, G: 244, B: 245, A: 255}
	white   = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	red     = color.RGBA{R: 251, G: 44, B: 54, A: 255}
	blue    = color.RGBA{R: 43, G: 127, B: 255, A: 255}
)

func literal(c color.RGBA) color.Color { return color.Color{Kind: color.Literal, RGBA: c} }

func plain() style.ComputedStyle { return style.ComputedStyle{Opacity: 1} }

func filled(c color.RGBA) style.ComputedStyle {
	s := plain()
	s.Background = literal(c)
	return s
}

func cells(n int) layout.Length { return layout.Length{Unit: layout.Cells, Value: n} }

func sized(w, h int, border layout.Edges) *layout.Box {
	b := &layout.Box{Style: layout.Style{Width: cells(w), Height: cells(h), Border: border}}
	layout.Layout(b, w, cells(h))
	return b
}

func rows(buf *buffer.Buffer) []string {
	var out []string
	for y := range buf.Height() {
		var line strings.Builder
		for _, c := range buf.Row(y) {
			line.WriteString(c.Grapheme)
		}
		out = append(out, line.String())
	}
	return out
}

func expect(t *testing.T, buf *buffer.Buffer, want ...string) {
	t.Helper()
	got := rows(buf)
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: %q, want %q", i, got[i], want[i])
		}
	}
}

func boxed(radius style.Radius, c color.Color) *buffer.Buffer {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	box := sized(6, 3, one)
	s := style.ComputedStyle{BorderStyle: style.BorderSingle, BorderColor: color.Color{Kind: color.Current}, Radius: radius, Color: c, Opacity: 1}
	buf := buffer.New(8, 4)
	Paint(buf, scene.New(box, s, scene.Text{}), Plain)
	return buf
}

func TestBorderCurrentColor(t *testing.T) {
	buf := boxed(style.RadiusLg, color.Color{Kind: color.Literal, RGBA: zinc100})
	for _, at := range [][2]int{{0, 0}, {3, 0}, {0, 1}, {5, 2}} {
		if c := buf.At(at[0], at[1]); c.Fg.Kind != color.Literal || c.Fg.RGBA != zinc100 {
			t.Errorf("border cell %v: fg %+v, want zinc-100", at, c.Fg)
		}
	}
}

func TestTextWide(t *testing.T) {
	box := sized(5, 1, layout.Edges{})
	buf := buffer.New(7, 2)
	Paint(buf, scene.New(box, plain(), scene.Sanitize("中文a")), Composited)
	expect(t, buf, "中文a  ", "       ")
	want := []buffer.Width{buffer.Wide, buffer.Continuation, buffer.Wide, buffer.Continuation, buffer.Narrow, buffer.Narrow, buffer.Narrow}
	for x, w := range want {
		if got := buf.At(x, 0).Width; got != w {
			t.Errorf("cell %d: width %d, want %d", x, got, w)
		}
	}
}

func TestTextWideAtEdge(t *testing.T) {
	box := sized(1, 1, layout.Edges{})
	buf := buffer.New(3, 1)
	Paint(buf, scene.New(box, plain(), scene.Sanitize("中")), Composited)
	expect(t, buf, "   ")
	if buf.At(1, 0).Width != buffer.Narrow {
		t.Errorf("cell 1 is %d, a wide glyph spilled out of a 1-cell box", buf.At(1, 0).Width)
	}
}

func TestTextTab(t *testing.T) {
	box := sized(12, 1, layout.Edges{})
	buf := buffer.New(12, 1)
	Paint(buf, scene.New(box, plain(), scene.Sanitize("ab\tc")), Composited)
	expect(t, buf, "ab      c   ")
}

func TestTextColors(t *testing.T) {
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{{Class: "panel", Decls: []style.Declaration{
		{Property: style.PropBackground, Color: color.Color{Kind: color.Literal, RGBA: zinc950}},
		{Property: style.PropColor, Color: color.Color{Kind: color.Literal, RGBA: zinc100}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	parent := sheet.Compute(style.ComputedStyle{}, []string{"panel"})
	child := sheet.Compute(parent, nil)
	leaf := &layout.Box{Measure: func(int) (int, int) { return 2, 1 }}
	root := &layout.Box{Style: layout.Style{Width: cells(6), Height: cells(3), Padding: layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}}, Children: []*layout.Box{leaf}}
	layout.Layout(root, 6, cells(3))
	buf := buffer.New(6, 3)
	page := scene.New(root, parent, scene.Text{})
	page.Children = []scene.Node{scene.New(leaf, child, scene.Sanitize("hi"))}
	Paint(buf, page, Composited)
	expect(t, buf, "      ", " hi   ", "      ")
	for x := range 2 {
		c := buf.At(1+x, 1)
		if c.Bg.Kind != color.Literal || c.Bg.RGBA != zinc950 {
			t.Errorf("text cell %d: bg %+v, want zinc-950 %+v", x, c.Bg, zinc950)
		}
		if c.Fg.Kind != color.Literal || c.Fg.RGBA != zinc100 {
			t.Errorf("text cell %d: fg %+v, want inherited zinc-100 %+v", x, c.Fg, zinc100)
		}
	}
}

func TestBackgroundTransparent(t *testing.T) {
	under := sized(4, 1, layout.Edges{})
	over := sized(2, 1, layout.Edges{})
	buf := buffer.New(4, 1)
	page := scene.New(under, filled(zinc950), scene.Text{})
	page.Children = []scene.Node{scene.New(over, filled(color.RGBA{}), scene.Text{})}
	Paint(buf, page, Composited)
	if c := buf.At(0, 0); c.Bg.RGBA != zinc950 {
		t.Errorf("transparent background replaced zinc-950 with %+v", c.Bg)
	}
}

func TestHiddenPaintsNothing(t *testing.T) {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	box := sized(4, 3, one)
	s := filled(zinc950)
	s.Visibility, s.BorderStyle = style.Hidden, style.BorderSingle
	buf := buffer.New(4, 3)
	Paint(buf, scene.New(box, s, scene.Sanitize("x")), Composited)
	for y := range 3 {
		for x := range 4 {
			if c := buf.At(x, y); c != (buffer.Cell{Grapheme: " "}) {
				t.Errorf("hidden node painted %+v at %d,%d", c, x, y)
			}
		}
	}
}

func TestPaintOffScreen(t *testing.T) {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	box := sized(6, 3, one)
	s := filled(zinc950)
	s.BorderStyle = style.BorderSingle
	buf := buffer.New(3, 2)
	Paint(buf, scene.New(box, s, scene.Sanitize("wide text")), Plain)
	expect(t, buf, "┌──", "│wi")
}

func TestTextHostile(t *testing.T) {
	corpus, err := os.ReadFile("../text/testdata/hostile.txt")
	if err != nil {
		t.Fatal(err)
	}
	box := sized(60, 4, layout.Edges{})
	cases := 0
	for line := range strings.Lines(string(corpus)) {
		fields := strings.Split(strings.TrimSuffix(line, "\n"), "\t")
		var raw strings.Builder
		for _, field := range fields[1 : len(fields)-1] {
			part, err := strconv.Unquote(field)
			if err != nil {
				t.Fatalf("%s: %v", fields[0], err)
			}
			raw.WriteString(part)
		}
		want, err := strconv.Unquote(fields[len(fields)-1])
		if err != nil {
			t.Fatalf("%s: %v", fields[0], err)
		}
		buf, clean := buffer.New(60, 4), buffer.New(60, 4)
		Paint(buf, scene.New(box, plain(), scene.Sanitize(raw.String())), Composited)
		Paint(clean, scene.New(box, plain(), scene.Sanitize(want)), Composited)
		if got, sanitised := rows(buf), rows(clean); !slices.Equal(got, sanitised) {
			t.Errorf("%s: painted %q, want the sanitised %q", fields[0], got, sanitised)
		}
		read := 0
		for y := range buf.Height() {
			for _, c := range buf.Row(y) {
				read++
				for _, r := range c.Grapheme {
					if r < 0x20 || (r >= 0x7f && r <= 0x9f) || (r >= 0x202a && r <= 0x202e) || (r >= 0x2066 && r <= 0x2069) {
						t.Errorf("%s: cell grapheme %q holds %U", fields[0], c.Grapheme, r)
					}
				}
			}
		}
		if read != 60*4 {
			t.Fatalf("%s: read %d cells", fields[0], read)
		}
		cases++
	}
	if cases == 0 {
		t.Fatal("no hostile cases read")
	}
	t.Logf("%d hostile cases, every cell read", cases)
}

func TestBorderCornerEach(t *testing.T) {
	left := style.RadiusNone.With(style.CornerTopLeft, style.RadiusMd).With(style.CornerBottomLeft, style.RadiusMd)
	expect(t, boxed(left, literal(white)), "╭────┐  ", "│    │  ", "╰────┘  ")
	bottomRight := style.RadiusNone.With(style.CornerBottomRight, style.RadiusSm)
	expect(t, boxed(bottomRight, literal(white)), "┌────┐  ", "│    │  ", "└────╯  ")
}
