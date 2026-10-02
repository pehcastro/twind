package present

import (
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/terminal"
)

func TestSymbolOverhangIsNotCoveredByAnImage(t *testing.T) {
	white, dark := color.RGBA{R: 255, G: 255, B: 255, A: 255}, color.RGBA{R: 24, G: 24, B: 27, A: 255}
	page := flatPage(white, "")
	card := flatPage(dark, "abc ✕ de")
	card.Bounds, card.Padding, card.Content = layout.Rect{X: 2, Y: 1, W: 14, H: 3}, layout.Rect{X: 2, Y: 1, W: 14, H: 3}, layout.Rect{X: 3, Y: 2, W: 12, H: 1}
	card.Border.Radius = style.RadiusLg
	card.Foreground = color.Color{Kind: color.Literal, RGBA: white}
	page.Children = []scene.Node{card}
	s, _ := screen(terminal.GraphicsSixel)
	frame(t, s, page)
	if s.imageBytes == 0 {
		t.Fatal("the rounded card sent no image; the test proves nothing")
	}
	if c := s.shown.At(7, 2); c.Grapheme != "✕" {
		t.Fatalf("cell 7,2 holds %q, want the ✕", c.Grapheme)
	}
	for _, x := range []int{6, 8} {
		if c := s.shown.At(x, 2); c.Grapheme != " " || c.Bg != s.shown.At(7, 2).Bg {
			t.Errorf("cell %d beside the ✕ is %+v, want a space written in the card colour so the image does not cover the symbol's overhang", x, c)
		}
	}
	if c := s.shown.At(12, 2); c.Grapheme != "" {
		t.Errorf("blank cell 12, away from the ✕, is %+v, want left to the image", c)
	}
}
