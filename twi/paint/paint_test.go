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
	zinc100 = color.RGBA{R: 244, G: 244, B: 245, A: 255}
)

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
	s := style.ComputedStyle{BorderStyle: style.BorderSingle, BorderColor: color.Color{Kind: color.Current}, Radius: radius, Color: c}
	buf := buffer.New(8, 4)
	Paint(buf, []scene.Node{scene.New(box, s, "")})
	return buf
}

func TestBorderRounded(t *testing.T) {
	buf := boxed(style.RadiusLg, color.Color{})
	expect(t, buf, "╭────╮  ", "│    │  ", "╰────╯  ", "        ")
}

func TestBorderSingle(t *testing.T) {
	buf := boxed(style.RadiusNone, color.Color{})
	expect(t, buf, "┌────┐  ", "│    │  ", "└────┘  ", "        ")
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
	Paint(buf, []scene.Node{scene.New(box, style.ComputedStyle{}, "中文ab")})
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
	Paint(buf, []scene.Node{scene.New(box, style.ComputedStyle{}, "中")})
	expect(t, buf, "   ")
	if buf.At(1, 0).Width != buffer.Narrow {
		t.Errorf("cell 1 is %d, a wide glyph spilled out of a 1-cell box", buf.At(1, 0).Width)
	}
}

func TestTextTab(t *testing.T) {
	box := sized(12, 1, layout.Edges{})
	buf := buffer.New(12, 1)
	Paint(buf, []scene.Node{scene.New(box, style.ComputedStyle{}, "ab\tc")})
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
	Paint(buf, []scene.Node{scene.New(root, parent, ""), scene.New(leaf, child, "hi")})
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
	Paint(buf, []scene.Node{
		scene.New(under, style.ComputedStyle{Background: color.Color{Kind: color.Literal, RGBA: zinc950}}, ""),
		scene.New(over, style.ComputedStyle{Background: color.Color{Kind: color.Literal}}, ""),
	})
	if c := buf.At(0, 0); c.Bg.RGBA != zinc950 {
		t.Errorf("transparent background replaced zinc-950 with %+v", c.Bg)
	}
}

func TestHiddenPaintsNothing(t *testing.T) {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	box := sized(4, 3, one)
	s := style.ComputedStyle{Visibility: style.Hidden, BorderStyle: style.BorderSingle, Background: color.Color{Kind: color.Literal, RGBA: zinc950}}
	buf := buffer.New(4, 3)
	Paint(buf, []scene.Node{scene.New(box, s, "x")})
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
	s := style.ComputedStyle{BorderStyle: style.BorderSingle, Background: color.Color{Kind: color.Literal, RGBA: zinc950}}
	buf := buffer.New(3, 2)
	Paint(buf, []scene.Node{scene.New(box, s, "wide text")})
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
		Paint(buf, []scene.Node{scene.New(box, style.ComputedStyle{}, raw.String())})
		Paint(clean, []scene.Node{scene.New(box, style.ComputedStyle{}, want)})
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
