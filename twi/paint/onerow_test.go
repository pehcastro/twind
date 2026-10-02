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

func bgAcross(t *testing.T, name string, buf *buffer.Buffer, to int, inside, outside color.Color) {
	t.Helper()
	for x := 1; x <= to+1; x++ {
		want := inside
		if x == 1 || x > to {
			want = outside
		}
		if c := buf.At(x, 1).Bg; c != want {
			t.Errorf("%s cell %d: bg %+v, want %+v", name, x, c.RGBA, want.RGBA)
		}
	}
}

func tokenRing(c color.RGBA, token theme.Token) style.Shadow {
	r := ring(c, 1)
	r.Token = token
	return r
}

func oneRowBox(s style.ComputedStyle, shadows ...style.Shadow) scene.Node {
	s.Color, s.Shadows = literal(zinc950), shadows
	return scene.New(place(2, 1, 6, 1, layout.Edges{}), s, scene.Sanitize("ab"))
}

func TestOneRowOutlineButton(t *testing.T) {
	for _, fill := range []color.RGBA{white, {R: 255, G: 255, B: 255, A: 77}} {
		for _, token := range []theme.Token{theme.Border, theme.Input} {
			buf := painted(12, 3, page(12, 3, "", oneRowBox(filled(fill), tokenRing(zinc800, token))), Composited)
			expect(t, buf, "            ", "  ab        ", "            ")
			bgAcross(t, "outline button", buf, 7, over(literal(fill), literal(zinc100)), literal(zinc100))
		}
	}
}

func TestOneRowFillLikeTheGroundCountsAsUnfilled(t *testing.T) {
	tint := zinc800
	tint.A = 89
	want := over(literal(tint), literal(zinc100))
	for _, fill := range []color.RGBA{zinc100, {R: 244, G: 244, B: 245, A: 128}} {
		buf := painted(12, 3, page(12, 3, "", oneRowBox(filled(fill), tokenRing(zinc800, theme.Border))), Composited)
		expect(t, buf, "            ", "  ab        ", "            ")
		bgAcross(t, "fill like the page", buf, 7, want, literal(zinc100))
	}
	split := page(12, 3, "", scene.New(place(0, 1, 5, 1, layout.Edges{}), filled(white), scene.Text{}), oneRowBox(filled(zinc100), tokenRing(zinc800, theme.Border)))
	buf := painted(12, 3, split, Composited)
	for x := 2; x < 8; x++ {
		if c := buf.At(x, 1); c.Bg != literal(zinc100) {
			t.Errorf("fill matching only part of the ground, cell %d: bg %+v, want the fill alone", x, c.Bg.RGBA)
		}
	}
}

func TestOneRowColouredRingTintsTheFill(t *testing.T) {
	tinted := func(c color.RGBA) color.Color {
		c.A = 89
		return over(literal(c), literal(white))
	}
	for _, c := range []struct {
		name  string
		rings []style.Shadow
		want  color.Color
	}{
		{"invalid", []style.Shadow{tokenRing(red, theme.Destructive)}, tinted(red)},
		{"explicit", []style.Shadow{tokenRing(blue, 0)}, tinted(blue)},
		{"focused, the halo carries the colour", focusRing(red, 51), literal(white)},
	} {
		buf := painted(12, 3, page(12, 3, "", oneRowBox(filled(white), c.rings...)), Composited)
		bgAcross(t, c.name, buf, 7, c.want, literal(zinc100))
	}
}

func TestOneRowBorderedInput(t *testing.T) {
	s := filled(white)
	s.Color, s.BorderStyle, s.BorderColor = literal(zinc950), style.BorderSingle, literal(zinc800)
	input := func() scene.Node {
		return scene.New(place(2, 1, 8, 1, layout.Edges{Left: 1, Right: 1}), s, scene.Sanitize("ab"))
	}
	buf := painted(12, 3, page(12, 3, "", input()), Composited)
	expect(t, buf, "            ", "   ab       ", "            ")
	bgAcross(t, "bordered input", buf, 9, literal(white), literal(zinc100))
	expect(t, painted(12, 3, page(12, 3, "", input()), Plain), "            ", "  │ab    │  ", "            ")
}

func TestOneRowOutlineBadge(t *testing.T) {
	faint, night, dim := white, color.RGBA{R: 5, G: 4, B: 6, A: 255}, color.RGBA{R: 36, G: 31, B: 46, A: 255}
	faint.A = 26
	on := func(ground, line color.RGBA) scene.Node {
		root := scene.New(place(0, 0, 12, 3, layout.Edges{}), filled(ground), scene.Text{})
		root.Children = []scene.Node{oneRowBox(plain(), tokenRing(line, theme.Border))}
		return root
	}
	for _, c := range []struct {
		name         string
		line, ground color.RGBA
		alpha        uint8
	}{
		{"light", zinc800, zinc100, 89},
		{"dark, a faint line", faint, zinc950, 54},
		{"dark, a line close to the page", dim, night, 172},
	} {
		buf := painted(12, 3, on(c.ground, c.line), Composited)
		expect(t, buf, "            ", "  ab        ", "            ")
		tint := c.line
		tint.A = c.alpha
		bgAcross(t, c.name, buf, 7, over(literal(tint), literal(c.ground)), literal(c.ground))
		if got := buf.At(2, 1); got.Fg != literal(zinc950) {
			t.Errorf("%s badge label: fg %+v, want the text colour over the tint", c.name, got.Fg.RGBA)
		}
	}
}

func TestRingOfTheFillColourGrowsTheFillByWholeCells(t *testing.T) {
	s := filled(zinc800)
	s.Color, s.Radius, s.Shadows = literal(white), style.RadiusFull, []style.Shadow{ring(zinc800, 4)}
	chip := scene.New(place(3, 1, 2, 1, layout.Edges{}), s, scene.Sanitize("CN"))
	buf := painted(8, 3, page(8, 3, "", chip), Composited)
	expect(t, buf, "        ", "   CN   ", "        ")
	for y := range 3 {
		for x := range 8 {
			want := literal(zinc100)
			if y == 1 && x >= 2 && x < 6 {
				want = literal(zinc800)
			}
			if c := buf.At(x, y); c.Bg != want {
				t.Errorf("cell %d,%d bg %+v, want %+v: a 4px ring of the fill colour is half a cell wide and a quarter row tall", x, y, c.Bg.RGBA, want.RGBA)
			}
		}
	}
}

func TestOneRowRuleKeepsItsLine(t *testing.T) {
	buf := painted(6, 3, page(6, 3, "", card(place(1, 1, 4, 1, layout.Edges{Bottom: 1}), style.RadiusNone)), Composited)
	expect(t, buf, "      ", " ──── ", "      ")
}
