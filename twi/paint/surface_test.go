package paint

import (
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
)

var shadowMd = color.RGBA{A: 18}

func place(x, y, w, h int, border layout.Edges) *layout.Box {
	inner := layout.Rect{X: x + border.Left, Y: y + border.Top, W: w - border.Left - border.Right, H: h - border.Top - border.Bottom}
	return &layout.Box{
		Style:     layout.Style{Border: border},
		BorderBox: layout.Rect{X: x, Y: y, W: w, H: h}, PaddingBox: inner, ContentBox: inner,
		Clip: layout.Rect{W: 100, H: 100},
	}
}

func page(w, h int, text string, children ...scene.Node) scene.Node {
	s := filled(zinc100)
	s.Color = literal(zinc950)
	n := scene.New(place(0, 0, w, h, layout.Edges{}), s, scene.Sanitize(text))
	n.Children = children
	return n
}

func card(box *layout.Box, radius style.Radius, shadows ...style.Shadow) scene.Node {
	s := filled(white)
	s.BorderStyle, s.BorderColor, s.Radius = style.BorderSingle, literal(zinc800), radius
	s.Shadows = shadows
	return scene.New(box, s, scene.Text{})
}

func painted(w, h int, root scene.Node, look Look) *buffer.Buffer {
	buf := buffer.New(w, h)
	Paint(buf, root, look)
	return buf
}

func TestHairline(t *testing.T) {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	buf := painted(12, 6, page(12, 6, "", card(place(1, 1, 10, 4, one), style.RadiusNone)), Composited)
	expect(t, buf, "            ", "  ▁▁▁▁▁▁▁▁  ", " ▕        ▏ ", " ▕        ▏ ", "  ▔▔▔▔▔▔▔▔  ", "            ")
	for _, at := range [][2]int{{2, 1}, {9, 1}, {1, 2}, {10, 3}, {4, 4}, {9, 4}} {
		c := buf.At(at[0], at[1])
		if c.Bg != literal(zinc100) || c.Fg != literal(zinc800) {
			t.Errorf("hairline cell %v: bg %+v fg %+v, want the parent's bg under the border colour", at, c.Bg, c.Fg)
		}
	}
	if c := buf.At(2, 2); c.Bg != literal(white) {
		t.Errorf("fill cell bg %+v, want white", c.Bg)
	}
}

func TestHairlineRounded(t *testing.T) {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	text := "abcdefghijkl\nmnopqrstuvwx\nABCDEFGHIJKL\nMNOPQRSTUVWX\nyz"
	bare := painted(12, 6, page(12, 6, text), Composited)
	buf := painted(12, 6, page(12, 6, text, card(place(1, 1, 10, 4, one), style.RadiusLg)), Composited)
	expect(t, buf, "abcdefghijkl", "mn▁▁▁▁▁▁▁▁wx", "A▕        ▏L", "M▕        ▏X", "yz▔▔▔▔▔▔▔▔  ")
	for _, at := range [][2]int{{1, 1}, {10, 1}, {1, 4}, {10, 4}} {
		if got, want := buf.At(at[0], at[1]), bare.At(at[0], at[1]); got != want {
			t.Errorf("corner %v: %+v, want the parent's cell %+v", at, got, want)
		}
	}
}

func TestHairlineOneEdge(t *testing.T) {
	buf := painted(6, 3, page(6, 3, "", card(place(1, 0, 4, 2, layout.Edges{Bottom: 1}), style.RadiusLg)), Composited)
	expect(t, buf, "      ", " ▔▔▔▔ ", "      ")
}

func TestHairlinePlain(t *testing.T) {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	square := painted(6, 3, page(6, 3, "", card(place(0, 0, 6, 3, one), style.RadiusNone)), Plain)
	expect(t, square, "┌────┐", "│    │", "└────┘")
	round := painted(6, 3, page(6, 3, "", card(place(0, 0, 6, 3, one), style.RadiusLg)), Plain)
	expect(t, round, "╭────╮", "│    │", "╰────╯")
}

func TestShadow(t *testing.T) {
	md := style.Shadow{X: 1, Y: 1, Color: literal(shadowMd)}
	bare := painted(12, 7, page(12, 7, ""), Composited)
	buf := painted(12, 7, page(12, 7, "", card(place(1, 1, 6, 3, layout.Edges{}), style.RadiusNone, md)), Composited)
	expect(t, buf, "            ", "            ", "       ▏    ", "       ▏    ", "  ▔▔▔▔▔     ", "            ")
	shade := color.RGBA{R: 227, G: 227, B: 228, A: 255}
	for y := range 7 {
		for x := range 12 {
			got, want := buf.At(x, y), bare.At(x, y)
			inCard := x >= 1 && x < 7 && y >= 1 && y < 4
			ring := x == 7 && y >= 2 && y < 4 || y == 4 && x >= 2 && x < 7
			switch {
			case inCard:
			case ring:
				if got.Bg != want.Bg || !near(got.Fg, shade) {
					t.Errorf("ring %d,%d: bg %+v fg %+v, want the page bg under black at alpha 18 %+v", x, y, got.Bg, got.Fg, shade)
				}
			case got != want:
				t.Errorf("cell %d,%d outside the ring changed: %+v, want %+v", x, y, got, want)
			}
		}
	}
}

func TestShadowStacked(t *testing.T) {
	md := style.Shadow{X: 1, Y: 1, Color: literal(shadowMd)}
	buf := painted(6, 4, page(6, 4, "", card(place(0, 0, 3, 2, layout.Edges{}), style.RadiusNone, md, md)), Composited)
	if c := buf.At(3, 1); c.Grapheme != "▏" || !near(c.Fg, color.RGBA{R: 211, G: 211, B: 212}) {
		t.Errorf("two stacked shadows: %+v, want ▏ at twice the shade", c)
	}
}

func TestShadowKeepsText(t *testing.T) {
	md := style.Shadow{X: 1, Y: 1, Color: literal(shadowMd)}
	text := strings.Repeat("abcdef\n", 4)
	bare := painted(6, 4, page(6, 4, text), Composited)
	buf := painted(6, 4, page(6, 4, text, card(place(0, 0, 3, 2, layout.Edges{}), style.RadiusNone, md)), Composited)
	for _, at := range [][2]int{{3, 1}, {1, 2}} {
		if got, want := buf.At(at[0], at[1]), bare.At(at[0], at[1]); got != want {
			t.Errorf("text under the shadow at %v: %+v, want %+v", at, got, want)
		}
	}
}

func TestShadowCurrentColor(t *testing.T) {
	s := filled(white)
	s.Color = literal(red)
	s.Shadows = []style.Shadow{{X: 1, Y: 1, Color: color.Color{Kind: color.Current}}}
	buf := painted(6, 4, page(6, 4, "", scene.New(place(0, 0, 3, 2, layout.Edges{}), s, scene.Text{})), Composited)
	if c := buf.At(3, 1); c.Grapheme != "▏" || c.Fg != literal(red) {
		t.Errorf("currentColor shadow %+v, want ▏ in the text colour", c)
	}
}

func TestShadowInset(t *testing.T) {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	s := filled(white)
	s.BorderStyle, s.BorderColor = style.BorderSingle, literal(zinc800)
	s.InsetShadows = []style.Shadow{{Y: 1, Color: literal(shadowMd), Inset: true}}
	buf := painted(6, 5, page(6, 5, "", scene.New(place(0, 0, 6, 5, one), s, scene.Text{})), Composited)
	expect(t, buf, " ▁▁▁▁ ", "▕▔▔▔▔▏", "▕    ▏", "▕    ▏", " ▔▔▔▔ ")
	if c := buf.At(2, 1); c.Bg != literal(white) || !near(c.Fg, color.RGBA{R: 237, G: 237, B: 237}) {
		t.Errorf("inset shade %+v, want black at alpha 18 over white", c)
	}
}

func TestShadowPlain(t *testing.T) {
	md := style.Shadow{X: 1, Y: 1, Color: literal(shadowMd)}
	bare := painted(6, 4, page(6, 4, ""), Plain)
	buf := painted(6, 4, page(6, 4, "", scene.New(place(0, 0, 3, 2, layout.Edges{}), style.ComputedStyle{Opacity: 1, Shadows: []style.Shadow{md}}, scene.Text{})), Plain)
	for y := range 4 {
		for x := range 6 {
			if buf.At(x, y) != bare.At(x, y) {
				t.Errorf("plain look painted a shadow at %d,%d: %+v", x, y, buf.At(x, y))
			}
		}
	}
}

func TestGlyphsLeavesSurfacesToPixels(t *testing.T) {
	md := style.Shadow{X: 1, Y: 1, Color: literal(shadowMd)}
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	bar := indigoToPink(t, style.ToRight)
	bar.Radius = style.RadiusFull
	root := page(10, 5, strings.Repeat("abcdefghij\n", 5),
		card(place(1, 0, 5, 3, one), style.RadiusLg, md),
		scene.New(place(1, 4, 6, 1, layout.Edges{}), bar, scene.Text{}),
	)
	buf := painted(10, 5, root, Glyphs)
	expect(t, buf, "a     ghij", "a     ghij", "a     ghij", "abcdefghij", "a      hij")
	if c := buf.At(3, 1); c.Bg != literal(white) {
		t.Errorf("card cell %+v, want the card fill kept for occlusion", c)
	}
}

func TestRootBackgroundPaintsTheCanvas(t *testing.T) {
	buf := painted(6, 4, page(6, 2, "ab"), Composited)
	if c := buf.At(5, 3); c.Bg != literal(zinc100) {
		t.Errorf("cell below a 2-row page: %+v, want the page background, as a browser paints the canvas", c)
	}
}

func TestShadowClip(t *testing.T) {
	md := style.Shadow{X: 2, Y: 2, Spread: 1, Color: literal(shadowMd)}
	box := place(0, 0, 4, 2, layout.Edges{})
	box.Clip = layout.Rect{W: 5, H: 2}
	bare := painted(6, 3, page(6, 3, ""), Composited)
	buf := painted(6, 3, page(6, 3, "", card(box, style.RadiusNone, md)), Composited)
	expect(t, buf, "      ", "      ", "      ")
	if c := buf.At(4, 1); !near(c.Bg, color.RGBA{R: 227, G: 227, B: 228}) {
		t.Errorf("inner shadow cell bg %+v, want the page shaded", c.Bg)
	}
	for _, at := range [][2]int{{4, 0}, {5, 1}, {1, 2}} {
		if got, want := buf.At(at[0], at[1]), bare.At(at[0], at[1]); got != want {
			t.Errorf("clipped cell %v: %+v, want %+v", at, got, want)
		}
	}
}
