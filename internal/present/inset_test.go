package present

import (
	"image"
	"math"
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/paint"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/layout"
	"github.com/pehcastro/twind/twi/scene"
	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/terminal"
)

func highlights(page, fill color.RGBA, radius style.Radius) scene.Node {
	root := flatPage(page, "")
	for y, text := range []string{"ab", "cd"} {
		item := flatPage(fill, text)
		at := layout.Rect{X: 2, Y: 2 + y, W: 12, H: 1}
		item.Bounds, item.Padding, item.Content = at, at, layout.Rect{X: 4, Y: 2 + y, W: 10, H: 1}
		item.Border.Radius = radius
		root.Children = append(root.Children, item)
	}
	return root
}

func TestOneRowHighlightsLeaveAGap(t *testing.T) {
	page, fill := color.RGBA{R: 7, G: 6, B: 10, A: 255}, color.RGBA{R: 28, G: 18, B: 43, A: 255}
	inset := int(math.Round(konst.OneRowInsetCell * float64(wt.Y)))
	if inset < 1 {
		t.Fatalf("inset %d px at %d px cells, want at least one", inset, wt.Y)
	}
	seam, column := 3*wt.Y, 10*wt.X+wt.X/2
	for _, g := range []terminal.Graphics{terminal.GraphicsKitty, terminal.GraphicsSixel, terminal.GraphicsGDI} {
		s, _ := screen(g)
		p := &painted{}
		s.Paint = p.paint
		frame(t, s, highlights(page, fill, style.RadiusMd))
		img := s.image()
		for y := seam - inset; y < seam+inset; y++ {
			if c := color.RGBA(img.RGBAAt(column, y)); c != page && (g == terminal.GraphicsKitty || c.A != 0) {
				t.Errorf("graphics %d: pixel row %d between two highlights is %v, want the page %v or transparent over the page", g, y, c, page)
			}
		}
		if c := color.RGBA(img.RGBAAt(column, seam-wt.Y/2)); c != fill {
			t.Errorf("graphics %d: the middle of a highlight is %v, want %v", g, c, fill)
		}
		if g != terminal.GraphicsKitty {
			if bg := s.shown.At(4, 2).Bg.RGBA; bg != fill {
				t.Errorf("graphics %d: text cell on a highlight has bg %v, want the fill %v, not a mean with the inset rows", g, bg, fill)
			}
		}
		if g == terminal.GraphicsGDI {
			a := cellAlpha(p.tile(t, image.Rect(0, 2, 8, 3)), wt, 4)
			if !all(a[:inset*wt.X], 255) || !all(a[len(a)-inset*wt.X:], 255) || !all(a[inset*wt.X:len(a)-inset*wt.X], 0) {
				t.Errorf("GDI text cell on a highlight: alphas top %v bottom %v, want the inset rows painted and the rest left to the console", a[:wt.X], a[len(a)-wt.X:])
			}
		}
	}
	for _, radius := range []style.Radius{style.RadiusNone, style.RadiusNone.With(style.CornerTopLeft, style.RadiusLg).With(style.CornerTopRight, style.RadiusLg)} {
		s, _ := screen(terminal.GraphicsKitty)
		frame(t, s, highlights(page, fill, radius))
		if c := color.RGBA(s.image().RGBAAt(column, seam)); c != fill {
			t.Errorf("radius %x: the seam is %v, want the fill: a box that is not rounded on every corner joins its neighbours", radius, c)
		}
	}
}

func TestGDISymbolOverhangIsNotCoveredByTheOverlay(t *testing.T) {
	white, dark := color.RGBA{R: 255, G: 255, B: 255, A: 255}, color.RGBA{R: 24, G: 24, B: 27, A: 255}
	page := ramp(white, "")
	card := flatPage(dark, "abc ✕ de")
	card.Bounds, card.Padding, card.Content = layout.Rect{X: 2, Y: 1, W: 12, H: 3}, layout.Rect{X: 2, Y: 1, W: 12, H: 3}, layout.Rect{X: 3, Y: 2, W: 10, H: 1}
	card.Border.Radius = style.RadiusLg
	card.Foreground = color.Color{Kind: color.Literal, RGBA: white}
	page.Children = []scene.Node{card}
	s, _, p := gdiScreen()
	frame(t, s, page)
	left, right := p.tile(t, image.Rect(0, 2, 8, 3)), p.tile(t, image.Rect(8, 2, 16, 3))
	if a := cellAlpha(left, wt, 3); !all(a, 0) {
		t.Errorf("text cell 3 on the flat card has overlay alphas %v, want it left to the console", a[:4])
	}
	for x, tile := range map[int]terminal.Tile{6: left, 8: right} {
		if a := cellAlpha(tile, wt, x%8); !all(a, 0) {
			t.Errorf("cell %d beside the ✕ has overlay alphas %v, want it left to the console so the symbol's overhang shows", x, a[:4])
		}
		if bg := s.shown.At(x, 2).Bg.RGBA; bg != dark {
			t.Errorf("cell %d beside the ✕ has bg %v, want the card %v", x, bg, dark)
		}
	}
	if a := cellAlpha(right, wt, 12-8); !all(a, 255) {
		t.Errorf("blank cell 12, away from the ✕, has overlay alphas %v, want it painted", a[:4])
	}
}
