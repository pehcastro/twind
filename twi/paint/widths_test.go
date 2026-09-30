package paint

import (
	"slices"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/text"
)

func flagCard(inner int, align style.TextAlign) scene.Node {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	s := plain()
	s.BorderStyle, s.TextAlign = style.BorderSingle, align
	return scene.New(place(0, 0, inner+2, 3, one), s, scene.Sanitize("🇧🇷ok"))
}

func cellsOf(buf *buffer.Buffer) []string {
	var out []string
	for y := range buf.Height() {
		var row []string
		for _, c := range buf.Row(y) {
			if c.Width == buffer.Continuation {
				row = append(row, "~")
				continue
			}
			row = append(row, c.Grapheme)
		}
		out = append(out, strings.Join(row, ""))
	}
	return out
}

func TestWidthsFlagInACard(t *testing.T) {
	for _, c := range []struct {
		name   string
		inner  int
		widths text.Widths
		align  style.TextAlign
		want   []string
	}{
		{"flag 1", 3, text.Widths{text.Flag: 1}, style.TextLeft, []string{"┌───┐ ", "│🇧🇷ok│ ", "└───┘ "}},
		{"default", 4, text.Widths{}, style.TextLeft, []string{"┌────┐", "│🇧🇷~ok│", "└────┘"}},
		{"flag 1 right", 4, text.Widths{text.Flag: 1}, style.TextRight, []string{"┌────┐", "│ 🇧🇷ok│", "└────┘"}},
	} {
		buf := buffer.New(6, 3)
		n := flagCard(c.inner, c.align)
		(&Painter{Widths: c.widths}).Paint(buf, &n, Plain)
		got := cellsOf(buf)
		t.Logf("%s:\n%s", c.name, strings.Join(got, "\n"))
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: frame\n%s\nwant\n%s", c.name, strings.Join(got, "\n"), strings.Join(c.want, "\n"))
		}
	}
	var reused Painter
	buf := buffer.New(6, 3)
	n := flagCard(4, style.TextLeft)
	for _, w := range []text.Widths{{}, {text.Flag: 1}, {}} {
		reused.Widths = w
		reused.Paint(buf, &n, Plain)
		fresh := buffer.New(6, 3)
		(&Painter{Widths: w}).Paint(fresh, &n, Plain)
		if got, want := cellsOf(buf), cellsOf(fresh); !slices.Equal(got, want) {
			t.Errorf("painter reused across widths %v:\n%s\nwant\n%s", w, strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	}
}

func TestTopLayerOrder(t *testing.T) {
	a := scene.New(place(0, 0, 3, 1, layout.Edges{}), filled(red), scene.Text{})
	b := scene.New(place(0, 0, 3, 1, layout.Edges{}), filled(blue), scene.Text{})
	var p Painter
	buf := buffer.New(3, 1)
	for _, c := range []struct {
		a, b int
		want [3]byte
	}{{1, 2, [3]byte{blue.R, blue.G, blue.B}}, {2, 1, [3]byte{red.R, red.G, red.B}}, {1, 2, [3]byte{blue.R, blue.G, blue.B}}} {
		a.TopLayer, b.TopLayer = c.a, c.b
		root := page(3, 1, "", a, b)
		p.Paint(buf, &root, Composited)
		if got := buf.At(1, 0).Bg.RGBA; [3]byte{got.R, got.G, got.B} != c.want {
			t.Errorf("top layers a %d b %d: cell %+v, want %v on top", c.a, c.b, got, c.want)
		}
	}
}
