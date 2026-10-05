package present

import (
	"testing"

	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/style"
)

func pill(label string, fg color.RGBA) scene.Node {
	root := flatPage(ink(7, 6, 10, 255), "")
	item := flatPage(ink(240, 240, 240, 255), label)
	at := layout.Rect{X: 1, Y: 2, W: 9, H: 1}
	item.Bounds, item.Padding, item.Content = at, at, layout.Rect{X: 2, Y: 2, W: 5, H: 1}
	item.Border.Radius = style.RadiusFull
	item.Foreground = color.Color{Kind: color.Literal, RGBA: fg}
	root.Children = []scene.Node{item}
	return root
}

func TestOneRowPillSpacesMatchTheirWords(t *testing.T) {
	s, out := screen(terminal.GraphicsSixel)
	m := &term{}
	frame(t, s, pill("ab cd", ink(0, 0, 0, 255)))
	m.write(t, out.last())
	img := s.image()
	seen := func(x, py int) color.RGBA {
		c := m.cells[2][x]
		if p := color.RGBA(img.RGBAAt(x*wt.X+wt.X/2, py)); c.image && p.A == 255 {
			return p
		}
		return c.bg
	}
	for _, py := range []int{2 * wt.Y, 3*wt.Y - 1} {
		for x := 2; x < 7; x++ {
			if got, want := seen(x, py), seen(2, py); got != want {
				t.Errorf("pixel row %d: cell %d shows %v, the letter at cell 2 shows %v: the space must match the words around it", py, x, got, want)
			}
		}
	}
	for _, x := range []int{1, 7, 9} {
		if c := m.cells[2][x]; !c.image || c.text != "" {
			t.Errorf("padding cell %d is %+v, want left to the image so the pill's rounded ends show", x, c)
		}
	}
	frame(t, s, pill("ab cd", ink(200, 0, 0, 255)))
	if s.imageBytes != 0 {
		t.Errorf("a text colour change sent %d image bytes, want none", s.imageBytes)
	}
}

func TestSymbolBesideAnInsetRowSendsNoImageOnATextChange(t *testing.T) {
	s, _ := screen(terminal.GraphicsSixel)
	frame(t, s, pill("ab ✕", ink(0, 0, 0, 255)))
	if c := s.shown.At(6, 2); c.Grapheme != " " {
		t.Fatalf("cell 6 beside the ✕ is %+v, want a written space; the test proves nothing", c)
	}
	frame(t, s, pill("ab ✕", ink(200, 0, 0, 255)))
	if s.imageBytes != 0 {
		t.Errorf("a text colour change sent %d image bytes, want none", s.imageBytes)
	}
}
