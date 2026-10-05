package paint

import (
	"math"
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/layout"
	"github.com/pehcastro/twind/twi/scene"
	"github.com/pehcastro/twind/twi/style"
)

func faded(opacity float64, children ...scene.Node) scene.Node {
	group := scene.New(place(1, 0, 8, 4, layout.Edges{}), plain(), scene.Text{})
	group.Opacity, group.Children = opacity, children
	return group
}

func TestFadingPanelHidesThePage(t *testing.T) {
	text := strings.Repeat("abcdefghij\n", 4)
	for _, opacity := range []float64{0.05, 0.3, 0.59, 0.6, 0.95} {
		panel := scene.New(place(1, 0, 8, 4, layout.Edges{}), filled(zinc800), scene.Sanitize("hi"))
		buf := painted(10, 4, page(10, 4, text, faded(opacity, panel)), Composited)
		t.Logf("opacity %.2f:\n%s", opacity, strings.Join(rows(buf), "\n"))
		ink := literal(zinc800)
		ink.RGBA.A = uint8(math.Round(math.MaxUint8 * opacity))
		want := over(ink, literal(zinc100))
		for y := range 4 {
			for x := 1; x < 9; x++ {
				c := buf.At(x, y)
				if y == 0 && (x == 1 || x == 2) {
					continue
				}
				if c.Grapheme != " " || !near(c.Bg, want.RGBA) {
					t.Errorf("opacity %.2f cell %d,%d: %q bg %+v, want a blank with the panel at %.2f over the page %+v", opacity, x, y, c.Grapheme, c.Bg, opacity, want.RGBA)
				}
			}
		}
		if c := buf.At(1, 0); c.Grapheme != "h" {
			t.Errorf("opacity %.2f: the panel's own text %q, want h", opacity, c.Grapheme)
		}
	}
}

func TestFadingBackdropKeepsThePage(t *testing.T) {
	text := strings.Repeat("abcdefghij\n", 4)
	backdrop := scene.New(place(1, 0, 8, 4, layout.Edges{}), filled(color.RGBA{A: 128}), scene.Text{})
	buf := painted(10, 4, page(10, 4, text, faded(0.5, backdrop)), Composited)
	expect(t, buf, "abcdefghij", "abcdefghij")
}

func TestFadingShadowDrawsWhatTheSettledOneDraws(t *testing.T) {
	md := []style.Shadow{{Y: 4, Blur: 6, Spread: -1, Color: literal(color.RGBA{A: 26})}, {Y: 2, Blur: 4, Spread: -2, Color: literal(color.RGBA{A: 26})}}
	text := strings.Repeat("qqq ─qq│qq q\n", 7)
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	settled := painted(12, 7, page(12, 7, text, card(place(2, 1, 8, 4, one), style.RadiusLg, md...)), Composited)
	group := scene.New(place(2, 1, 8, 4, layout.Edges{}), plain(), scene.Text{})
	group.Opacity, group.Children = 0.5, []scene.Node{card(place(2, 1, 8, 4, one), style.RadiusLg, md...)}
	fading := painted(12, 7, page(12, 7, text, group), Composited)
	t.Logf("settled:\n%s\nat half opacity:\n%s", strings.Join(rows(settled), "\n"), strings.Join(rows(fading), "\n"))
	for y := range 7 {
		for x := range 12 {
			if got, want := fading.At(x, y).Grapheme, settled.At(x, y).Grapheme; got != want {
				t.Errorf("cell %d,%d at half opacity holds %q, settled %q: the shadow jumps when the fade ends", x, y, got, want)
			}
		}
	}
}

func TestDropShadowEdgeCoversThePage(t *testing.T) {
	lg := []style.Shadow{
		{Y: 10, Blur: 15, Spread: -3, Color: literal(color.RGBA{A: 26})},
		{Y: 4, Blur: 6, Spread: -4, Color: literal(color.RGBA{A: 26})},
	}
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	bare := painted(12, 7, page(12, 7, "", card(place(2, 1, 8, 4, one), style.RadiusLg, lg...)), Composited)
	busy := painted(12, 7, page(12, 7, strings.Repeat("╮──abcdefg─╮\n", 7), card(place(2, 1, 8, 4, one), style.RadiusLg, lg...)), Composited)
	t.Logf("over a blank page:\n%s\nover text:\n%s", strings.Join(rows(bare), "\n"), strings.Join(rows(busy), "\n"))
	edges := 0
	for y := range 7 {
		for x := range 12 {
			inside := x >= 2 && x < 10 && y >= 1 && y < 5
			if b := bare.At(x, y); !inside && b.Grapheme != " " {
				edges++
				if c := busy.At(x, y); c.Grapheme != b.Grapheme {
					t.Errorf("shadow edge %d,%d over the page holds %q, want %q as over a blank page", x, y, c.Grapheme, b.Grapheme)
				}
			}
		}
	}
	if edges == 0 {
		t.Fatal("shadow-lg drew no edge glyph over a blank page; the test proves nothing")
	}
}
