package paint

import (
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/buffer"
	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/style"
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

func TestOneFrameFilledBorderedBox(t *testing.T) {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	muted, line := color.RGBA{R: 39, G: 39, B: 42, A: 255}, color.RGBA{R: 255, G: 255, B: 255, A: 26}
	s := filled(muted)
	s.BorderStyle, s.BorderColor, s.Radius = style.BorderSingle, literal(line), style.RadiusLg
	root := scene.New(place(0, 0, 12, 6, layout.Edges{}), filled(zinc950), scene.Text{})
	root.Children = []scene.Node{scene.New(place(1, 1, 10, 4, one), s, scene.Text{})}
	rim := over(literal(line), literal(zinc950))
	for _, profile := range []color.Profile{color.TrueColor, color.ANSI16} {
		p := Painter{Profile: profile}
		buf := buffer.New(12, 6)
		p.Paint(buf, &root, Composited)
		t.Logf("profile %d:\n%s", profile, strings.Join(rows(buf), "\n"))
		expect(t, buf, "            ", " ╭────────╮ ", " │        │ ", " │        │ ", " ╰────────╯ ", "            ")
		for y := range 6 {
			for x := range 12 {
				c := buf.At(x, y)
				inside := x >= 2 && x < 10 && y >= 2 && y < 4
				ring := !inside && x >= 1 && x < 11 && y >= 1 && y < 5
				switch {
				case inside && c.Bg.RGBA.ANSI16() == zinc950.ANSI16() && (profile == color.ANSI16 || c.Bg != literal(muted)):
					t.Errorf("profile %d cell %d,%d inside the line: bg %+v, want the fill", profile, x, y, c.Bg)
				case !inside && c.Bg != literal(zinc950):
					t.Errorf("profile %d cell %d,%d outside the line: bg %+v, want the page; fill here reads as a second frame", profile, x, y, c.Bg)
				case ring && profile == color.TrueColor && c.Fg != rim:
					t.Errorf("ring cell %d,%d: fg %+v, want the border over the page %+v", x, y, c.Fg, rim)
				case ring && profile == color.ANSI16 && c.Fg.RGBA.ANSI16() == c.Bg.RGBA.ANSI16():
					t.Errorf("profile %d ring cell %d,%d: the line and the page share index %d", profile, x, y, c.Fg.RGBA.ANSI16())
				}
			}
		}
	}
}

func TestBorderRounded(t *testing.T) {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	text := "abcdefghijkl\nmnopqrstuvwx\nABCDEFGHIJKL\nMNOPQRSTUVWX\nyz"
	buf := painted(12, 6, page(12, 6, text, card(place(1, 1, 10, 4, one), style.RadiusLg)), Composited)
	expect(t, buf, "abcdefghijkl", "m╭────────╮x", "A│        │L", "M│        │X", "y╰────────╯ ")
	square := painted(12, 6, page(12, 6, "", card(place(1, 1, 10, 4, one), style.RadiusNone)), Composited)
	expect(t, square, "            ", " ┌────────┐ ", " │        │ ", " │        │ ", " └────────┘ ", "            ")
	for y := range 6 {
		for x := range 12 {
			want := literal(zinc100)
			if x >= 2 && x < 10 && y >= 2 && y < 4 {
				want = literal(white)
			}
			if c := square.At(x, y); c.Bg != want {
				t.Errorf("cell %d,%d: bg %+v, want %+v; the fill stays inside the line", x, y, c.Bg, want)
			}
		}
	}
}

func TestFilledBorderedBoxWithNoInsideKeepsItsFill(t *testing.T) {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	for _, c := range []struct {
		box  layout.Rect
		want []string
	}{
		{layout.Rect{X: 1, Y: 1, W: 5, H: 2}, []string{"       ", " ╭───╮ ", " ╰───╯ ", "       "}},
		{layout.Rect{X: 1, Y: 1, W: 2, H: 3}, []string{"       ", " ╭╮    ", " ││    ", " ╰╯    "}},
	} {
		buf := painted(7, 4, page(7, 4, "", card(place(c.box.X, c.box.Y, c.box.W, c.box.H, one), style.RadiusLg)), Composited)
		expect(t, buf, c.want...)
		for y := range 4 {
			for x := range 7 {
				want := literal(zinc100)
				if x >= c.box.X && x < c.box.X+c.box.W && y >= c.box.Y && y < c.box.Y+c.box.H {
					want = literal(white)
				}
				if got := buf.At(x, y).Bg; got != want {
					t.Errorf("box %+v cell %d,%d: bg %+v, want %+v; a box with no inside shows its fill under the line and nowhere else", c.box, x, y, got, want)
				}
			}
		}
	}
}

func TestBorderOneEdge(t *testing.T) {
	buf := painted(6, 3, page(6, 3, "", card(place(1, 0, 4, 2, layout.Edges{Bottom: 1}), style.RadiusLg)), Composited)
	expect(t, buf, "      ", " ──── ", "      ")
}

func TestBorderPlain(t *testing.T) {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	square := painted(6, 3, page(6, 3, "", card(place(0, 0, 6, 3, one), style.RadiusNone)), Plain)
	expect(t, square, "┌────┐", "│    │", "└────┘")
	round := painted(6, 3, page(6, 3, "", card(place(0, 0, 6, 3, one), style.RadiusLg)), Plain)
	expect(t, round, "╭────╮", "│    │", "╰────╯")
}

func TestShadow(t *testing.T) {
	md := style.Shadow{X: 8, Y: 16, Color: literal(shadowMd)}
	bare := painted(12, 7, page(12, 7, ""), Composited)
	buf := painted(12, 7, page(12, 7, "", card(place(1, 1, 6, 3, layout.Edges{}), style.RadiusNone, md)), Composited)
	expect(t, buf, "            ", "            ", "            ", "            ", "            ", "            ")
	shade := color.RGBA{R: 227, G: 227, B: 228, A: 255}
	for y := range 7 {
		for x := range 12 {
			got, want := buf.At(x, y), bare.At(x, y)
			inCard := x >= 1 && x < 7 && y >= 1 && y < 4
			ring := x == 7 && y >= 2 && y < 4 || y == 4 && x >= 2 && x < 7
			switch {
			case inCard:
			case ring:
				if !near(got.Bg, shade) {
					t.Errorf("ring %d,%d: bg %+v, want a whole cell of the page under black at alpha 18 %+v", x, y, got.Bg, shade)
				}
			case got != want:
				t.Errorf("cell %d,%d outside the ring changed: %+v, want %+v", x, y, got, want)
			}
		}
	}
}

func TestShadowStockMd(t *testing.T) {
	black := literal(color.RGBA{A: 26})
	md := []style.Shadow{{Y: 4, Blur: 6, Spread: -1, Color: black}, {Y: 2, Blur: 4, Spread: -2, Color: black}}
	buf := painted(12, 7, page(12, 7, "", card(place(1, 1, 6, 3, layout.Edges{}), style.RadiusNone, md...)), Composited)
	expect(t, buf, "            ", "▐      ▌    ", "▐      ▌    ", "▐      ▌    ", " ▀▀▀▀▀▀     ", "            ")
	if side, below := buf.At(7, 2).Fg, buf.At(3, 4).Fg; !near(side, color.RGBA{R: 219, G: 219, B: 220}) || !near(below, color.RGBA{R: 219, G: 219, B: 220}) {
		t.Errorf("side %+v below %+v: want the first layer alone, the second reaches under a quarter cell", side, below)
	}
}

func TestShadowNegativeSpread(t *testing.T) {
	sunk := style.Shadow{Y: 4, Spread: -12, Color: literal(shadowMd)}
	bare := painted(8, 5, page(8, 5, ""), Composited)
	buf := painted(8, 5, page(8, 5, "", card(place(1, 1, 4, 2, layout.Edges{}), style.RadiusNone, sunk)), Composited)
	for y := range 5 {
		for x := range 8 {
			inCard := x >= 1 && x < 5 && y >= 1 && y < 3
			if !inCard && buf.At(x, y) != bare.At(x, y) {
				t.Errorf("a shadow hidden under its box by spread painted %d,%d: %+v", x, y, buf.At(x, y))
			}
		}
	}
}

func TestShadowStacked(t *testing.T) {
	md := style.Shadow{X: 8, Y: 16, Color: literal(shadowMd)}
	buf := painted(6, 4, page(6, 4, "", card(place(0, 0, 3, 2, layout.Edges{}), style.RadiusNone, md, md)), Composited)
	if c := buf.At(3, 1); c.Grapheme != " " || !near(c.Bg, color.RGBA{R: 211, G: 211, B: 212}) {
		t.Errorf("two stacked shadows: %+v, want a cell at twice the shade", c)
	}
}

func TestShadowKeepsText(t *testing.T) {
	md := style.Shadow{X: 8, Y: 16, Color: literal(shadowMd)}
	text := strings.Repeat("abcdef\n", 4)
	bare := painted(6, 4, page(6, 4, text), Composited)
	buf := painted(6, 4, page(6, 4, text, card(place(0, 0, 3, 2, layout.Edges{}), style.RadiusNone, md)), Composited)
	for _, at := range [][2]int{{3, 1}, {1, 2}} {
		if got, want := buf.At(at[0], at[1]), bare.At(at[0], at[1]); got.Grapheme != want.Grapheme || got.Bg == want.Bg {
			t.Errorf("text under the shadow at %v: %+v, want %q kept on a shaded cell", at, got, want.Grapheme)
		}
	}
}

func TestShadowCurrentColor(t *testing.T) {
	s := filled(white)
	s.Color = literal(red)
	s.Shadows = []style.Shadow{{X: 8, Y: 16, Color: color.Color{Kind: color.Current}}}
	buf := painted(6, 4, page(6, 4, "", scene.New(place(0, 0, 3, 2, layout.Edges{}), s, scene.Text{})), Composited)
	if c := buf.At(3, 1); c.Bg != literal(red) {
		t.Errorf("currentColor shadow %+v, want a cell in the text colour", c)
	}
}

func TestShadowInset(t *testing.T) {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	s := filled(white)
	s.BorderStyle, s.BorderColor = style.BorderSingle, literal(zinc800)
	s.InsetShadows = []style.Shadow{{Y: 16, Color: literal(shadowMd), Inset: true}}
	buf := painted(6, 5, page(6, 5, "", scene.New(place(0, 0, 6, 5, one), s, scene.Text{})), Composited)
	expect(t, buf, "┌────┐", "│    │", "│    │", "│    │", "└────┘")
	if c := buf.At(2, 1); !near(c.Bg, color.RGBA{R: 237, G: 237, B: 237}) {
		t.Errorf("inset shade %+v, want black at alpha 18 over white", c)
	}
}

func TestShadowPlain(t *testing.T) {
	md := style.Shadow{X: 8, Y: 16, Color: literal(shadowMd)}
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
	md := style.Shadow{X: 8, Y: 16, Color: literal(shadowMd)}
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
	md := style.Shadow{X: 16, Y: 24, Spread: 8, Color: literal(shadowMd)}
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
