package present

import (
	"image"
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/terminal"
)

var pageInk = ink(9, 9, 11, 255)

func pageWithCard(text string) scene.Node {
	p := flatPage(pageInk, text)
	c := box(2, 2, 12, 5, ink(24, 24, 27, 255))
	c.Border.Radius = style.RadiusLg
	c.Shadows = []style.Shadow{{Y: 8, Blur: 16, Color: color.Color{Kind: color.Literal, RGBA: ink(0, 0, 0, 90)}}}
	p.Children = []scene.Node{c}
	return p
}

func TestPagePixelsAreTransparentInSixel(t *testing.T) {
	s, out := screen(terminal.GraphicsSixel)
	frame(t, s, pageWithCard(""))
	img := s.image()
	for _, at := range []image.Point{{0, 0}, {20, 40}, {20 + 12*wt.X - 1, 40}, {cols*wt.X - 1, rows*wt.Y - 1}} {
		if got := img.RGBAAt(at.X, at.Y); got.A != 0 {
			t.Errorf("pixel %v where only the page is: %+v, want alpha 0", at, got)
		}
	}
	if got := img.RGBAAt(80, 80); got.A != 255 || got.R != 24 {
		t.Errorf("pixel 80,80 inside the card: %+v, want the opaque card colour", got)
	}
	for i := 3; i < len(img.Pix); i += 4 {
		if a := img.Pix[i]; a != 0 && a != 255 {
			t.Fatalf("pixel %d,%d has alpha %d, want 0 or 255: sixel cannot blend", i/4%img.Rect.Dx(), i/4/img.Rect.Dx(), a)
		}
	}
	m := &term{}
	m.write(t, out.last())
	for _, at := range [][2]int{{2, 2}, {60, 20}, {0, 0}} {
		if got := m.cells[at[1]][at[0]].bg; got != pageInk {
			t.Errorf("cell %v under the image shows %+v, want the exact page colour %+v", at, got, pageInk)
		}
	}
	frame(t, s, flatPage(pageInk, ""))
	m.write(t, out.last())
	if c := m.cells[2][2]; c.image || c.bg != pageInk {
		t.Errorf("cell 2,2 after the card left: %+v, want erased to the page colour with no image left", c)
	}
}

func TestTextOnThePageTakesThePageColour(t *testing.T) {
	s, _ := screen(terminal.GraphicsSixel)
	frame(t, s, pageWithCard("abc"))
	if got := s.shown.At(0, 0).Bg; got != (color.Color{Kind: color.Literal, RGBA: pageInk}) {
		t.Errorf("text cell 0,0 on the page has bg %+v, want the exact page colour %+v", got, pageInk)
	}
	img, corner := s.image(), s.sample(2, 2)
	if corner.RGBA.A == 255 || corner.RGBA.A == 0 {
		t.Fatalf("corner cell 2,2 sampled %+v, want a partial alpha: it is part page, part card", corner)
	}
	var sum [3]int
	for p := range wt.X * wt.Y {
		px := img.RGBAAt(2*wt.X+p%wt.X, 2*wt.Y+p/wt.X)
		if px.A == 0 {
			px.R, px.G, px.B = pageInk.R, pageInk.G, pageInk.B
		}
		sum[0], sum[1], sum[2] = sum[0]+int(px.R), sum[1]+int(px.G), sum[2]+int(px.B)
	}
	n := wt.X * wt.Y
	want := color.RGBA{R: register(uint8((sum[0] + n/2) / n)), G: register(uint8((sum[1] + n/2) / n)), B: register(uint8((sum[2] + n/2) / n))}
	if got := (color.RGBA{R: corner.RGBA.R, G: corner.RGBA.G, B: corner.RGBA.B}); got != want {
		t.Errorf("corner cell 2,2 sampled %+v, want %+v: its pixels flattened over the page", got, want)
	}
}
