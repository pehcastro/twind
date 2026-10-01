package paint

import (
	"testing"

	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/style"
)

func TestCornerHidesThePage(t *testing.T) {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	text := "xxxxxxxxxxxx中中中中中中xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	corners := [4][2]int{{1, 1}, {10, 1}, {1, 3}, {10, 3}}
	rounded := [4]string{"╭", "╮", "╰", "╯"}
	for _, look := range []Look{Composited, Plain, Glyphs} {
		buf := painted(12, 5, page(12, 5, text, card(place(1, 1, 10, 3, one), style.RadiusMd)), look)
		for i, at := range corners {
			c, want := buf.At(at[0], at[1]), buffer.Cell{Grapheme: rounded[i], Fg: literal(zinc800), Bg: literal(white)}
			switch look {
			case Composited:
				want.Bg = literal(zinc100)
			case Plain:
			case Glyphs:
				want = buffer.Cell{Grapheme: " ", Bg: literal(white)}
			}
			if c != want {
				t.Errorf("look %d corner %v: %+v, want %+v", look, at, c, want)
			}
		}
		if c := buf.At(0, 1); c.Grapheme == "中" || c.Width != buffer.Narrow {
			t.Errorf("look %d: the wide glyph under the corner kept its head %+v", look, c)
		}
	}
	clipped := place(1, 1, 10, 3, one)
	clipped.Clip = layout.Rect{W: 10, H: 5}
	buf := painted(12, 5, page(12, 5, text, card(clipped, style.RadiusMd)), Composited)
	if c := buf.At(10, 3); c.Grapheme != "x" {
		t.Errorf("a corner outside the clip was written: %+v", c)
	}
	top := layout.Edges{Top: 1}
	buf = painted(12, 5, page(12, 5, text, card(place(1, 1, 10, 3, top), style.RadiusMd)), Composited)
	if c := buf.At(1, 1); c.Grapheme != "─" {
		t.Errorf("a top-only border's first cell %+v, want the line", c)
	}
}
