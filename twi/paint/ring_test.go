package paint

import (
	"testing"

	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/theme"
)

func ring(c color.RGBA, spread style.Pixels) style.Shadow {
	return style.Shadow{Spread: spread, Color: literal(c)}
}

func focusRing(c color.RGBA, halo uint8) []style.Shadow {
	soft := c
	soft.A = halo
	return []style.Shadow{ring(c, 1), ring(soft, 3)}
}

func ringed(x, y, w int, rings ...style.Shadow) scene.Node {
	s := filled(white)
	s.Shadows = rings
	return scene.New(place(x, y, w, 1, layout.Edges{}), s, scene.Text{})
}

func edgeAt(t *testing.T, buf *buffer.Buffer, x, y int, glyph string, fg, bg color.RGBA) {
	t.Helper()
	if c := buf.At(x, y); c.Grapheme != glyph || c.Attr != 0 || c.Fg != literal(fg) || c.Bg != literal(bg) {
		t.Errorf("cell %d,%d: %q attr %d fg %+v bg %+v, want %q fg %+v bg %+v", x, y, c.Grapheme, c.Attr, c.Fg.RGBA, c.Bg.RGBA, glyph, fg, bg)
	}
}

func TestRingShare(t *testing.T) {
	both := func(a, b scene.Node, check func(*buffer.Buffer)) {
		t.Helper()
		check(painted(10, 5, page(10, 5, "", a, b), Composited))
		check(painted(10, 5, page(10, 5, "", b, a), Composited))
	}
	tall := func(y int, r style.Shadow) scene.Node {
		s := filled(white)
		s.Shadows = []style.Shadow{r}
		return scene.New(place(2, y, 4, 2, layout.Edges{}), s, scene.Text{})
	}
	both(tall(0, ring(zinc800, 1)), tall(3, ring(zinc800, 1)), func(buf *buffer.Buffer) {
		t.Helper()
		for x := 2; x < 6; x++ {
			edgeAt(t, buf, x, 2, "─", zinc800, zinc100)
		}
	})
	both(ringed(2, 1, 4, ring(zinc800, 1)), ringed(2, 3, 4, ring(zinc800, 1)), func(buf *buffer.Buffer) {
		t.Helper()
		if c := buf.At(3, 2); c.Grapheme != " " {
			t.Errorf("between two one-row boxes with thin rings: %+v, want the page, a one-row box rings its sides only", c)
		}
	})
	for _, later := range []struct {
		first, last scene.Node
		ink         color.RGBA
	}{
		{tall(0, ring(red, 1)), tall(3, ring(blue, 1)), blue},
		{tall(3, ring(blue, 1)), tall(0, ring(red, 1)), red},
	} {
		buf := painted(10, 5, page(10, 5, "", later.first, later.last), Composited)
		for x := 2; x < 6; x++ {
			edgeAt(t, buf, x, 2, "─", later.ink, zinc100)
		}
	}
	both(ringed(2, 1, 4, focusRing(blue, 128)...), ringed(2, 3, 4, ring(zinc800, 1)), func(buf *buffer.Buffer) {
		t.Helper()
		for x := 2; x < 6; x++ {
			edgeAt(t, buf, x, 2, "▀", blue, zinc100)
		}
	})
	both(ringed(2, 1, 4, ring(zinc800, 1)), ringed(2, 3, 4, focusRing(blue, 128)...), func(buf *buffer.Buffer) {
		t.Helper()
		for x := 2; x < 6; x++ {
			edgeAt(t, buf, x, 2, "▄", blue, zinc100)
		}
	})
	both(ringed(1, 1, 2, focusRing(blue, 128)...), ringed(4, 1, 2, ring(zinc800, 1)), func(buf *buffer.Buffer) {
		t.Helper()
		edgeAt(t, buf, 3, 1, "▌", blue, zinc100)
	})
	both(ringed(1, 1, 2, ring(zinc800, 1)), ringed(4, 1, 2, focusRing(blue, 128)...), func(buf *buffer.Buffer) {
		t.Helper()
		edgeAt(t, buf, 3, 1, "▐", blue, zinc100)
	})
	buf := painted(10, 5, page(10, 5, "", ringed(1, 1, 2, ring(zinc800, 1)), ringed(4, 1, 2, ring(zinc800, 1))), Composited)
	if c := buf.At(3, 1); c.Grapheme != " " || c.Bg != literal(zinc100) {
		t.Errorf("between two one-row boxes side by side with thin rings: %+v, want the page", c)
	}
}

func TestRingOverContent(t *testing.T) {
	buf := painted(10, 3, page(10, 3, "abcdefghij\n\nklmnopqrst", ringed(2, 1, 4, focusRing(blue, 128)...)), Composited)
	expect(t, buf, "abcdefghij", " ▐    ▌   ", "klmnopqrst")
}

func TestPillRing(t *testing.T) {
	green100 := color.RGBA{R: 220, G: 252, B: 231, A: 255}
	pill := func(x int, fill bool, rings ...style.Shadow) scene.Node {
		s := plain()
		if fill {
			s = filled(green100)
		}
		s.Radius, s.Shadows = style.RadiusFull, rings
		return scene.New(place(x, 1, 4, 1, layout.Edges{}), s, scene.Text{})
	}
	buf := painted(12, 3, page(12, 3, "", pill(4, true, tokenRing(zinc800, theme.Border))), Composited)
	expect(t, buf, "            ", "            ", "            ")
	for _, x := range []int{3, 8} {
		if c := buf.At(x, 1); c.Bg != literal(zinc100) {
			t.Errorf("beside a one-row ringed pill, cell %d: bg %+v, want the page", x, c.Bg)
		}
	}
	for x := 4; x < 8; x++ {
		if c := buf.At(x, 1); c.Bg != literal(green100) {
			t.Errorf("pill cell %d: bg %+v, want the pill fill across the whole box", x, c.Bg)
		}
	}
	expect(t, painted(12, 3, page(12, 3, "", pill(4, false, ring(zinc800, 1))), Composited), "            ", "            ", "            ")
	var p Painter
	moved := painted(20, 3, page(20, 3, "", pill(9, true)), Composited)
	buf = buffer.New(20, 3)
	before, after := page(20, 3, "", pill(9, true, ring(zinc800, 1))), page(20, 3, "", pill(9, true))
	p.Paint(buf, &before, Composited)
	p.Paint(buf, &after, Composited)
	if got, want := buf.At(7, 1), moved.At(7, 1); got != want {
		t.Errorf("after the ring went, cell 7,1 holds %+v, want %+v", got, want)
	}
}

func TestFocusInvalid(t *testing.T) {
	invalid := painted(10, 3, page(10, 3, "", ringed(2, 1, 4, ring(red, 1))), Composited)
	expect(t, invalid, "          ", "          ", "          ")
	focused := painted(10, 3, page(10, 3, "", ringed(2, 1, 4, focusRing(red, 51)...)), Composited)
	expect(t, focused, "  ▄▄▄▄    ", " ▐    ▌   ", "  ▀▀▀▀    ")
	edgeAt(t, focused, 3, 0, "▄", red, zinc100)
	edgeAt(t, focused, 6, 1, "▌", red, zinc100)
	edgeAt(t, focused, 3, 2, "▀", red, zinc100)
	edgeAt(t, focused, 1, 1, "▐", red, zinc100)
	bare := scene.New(place(0, 0, 10, 3, layout.Edges{}), plain(), scene.Text{})
	bare.Children = []scene.Node{ringed(2, 1, 4, focusRing(red, 51)...)}
	expect(t, painted(10, 3, bare, Composited), "  ▄▄▄▄    ", " ▐    ▌   ", "  ▀▀▀▀    ")
}
