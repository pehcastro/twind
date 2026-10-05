package present

import (
	"testing"

	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/style"
)

func outlined(fg color.RGBA) scene.Node {
	root := pill("Button", fg)
	item := &root.Children[0]
	item.Background = color.Color{Kind: color.Literal, RGBA: ink(13, 13, 18, 255)}
	item.Content = layout.Rect{X: 2, Y: 2, W: 6, H: 1}
	item.Border = scene.Border{Style: style.BorderSingle, Radius: style.RadiusMd, Top: true, Right: true, Bottom: true, Left: true, Color: color.Color{Kind: color.Literal, RGBA: ink(120, 110, 160, 255)}}
	return root
}

func percent(c color.RGBA) color.RGBA {
	round := func(v uint8) uint8 { return uint8(((int(v)*100+127)/255*255 + 50) / 100) }
	return color.RGBA{R: round(c.R), G: round(c.G), B: round(c.B), A: c.A}
}

func TestOneRowRingRunsOverItsText(t *testing.T) {
	s, out := screen(terminal.GraphicsSixel)
	m := &term{}
	check := func(name string) {
		t.Helper()
		img := s.image()
		for x := 2; x < 8; x++ {
			middle := 2*wt.Y + wt.Y/2
			for py := 2 * wt.Y; py < 3*wt.Y; py++ {
				for px := x * wt.X; px < (x+1)*wt.X; px++ {
					want, edge := color.RGBA(img.RGBAAt(px, py)), img.RGBAAt(px, py) != img.RGBAAt(px, middle)
					if want.A == 0 {
						want = s.pageBg.RGBA
					}
					got := m.seen(px, py)
					switch {
					case edge && got != percent(want):
						t.Fatalf("%s: pixel %d,%d over the text cell %d shows %v, want the ring row %v", name, px, py, x, got, want)
					case !edge && m.layer.RGBAAt(px, py).A != 0:
						t.Fatalf("%s: pixel %d,%d in the middle of text cell %d is covered by the image, want the glyph left visible", name, px, py, x)
					case !edge && got != want:
						t.Fatalf("%s: pixel %d,%d in the middle of text cell %d shows %v, want the fill %v", name, px, py, x, got, want)
					}
				}
			}
		}
	}
	frame(t, s, outlined(ink(240, 240, 240, 255)))
	m.write(t, out.last())
	check("first frame")
	frame(t, s, outlined(ink(200, 0, 0, 255)))
	m.write(t, out.last())
	check("label recoloured")
	sent := len(out.frames)
	frame(t, s, outlined(ink(200, 0, 0, 255)))
	if len(out.frames) != sent {
		t.Errorf("an unchanged frame wrote %q, want nothing", out.last())
	}
}
