package paint

import (
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/layout"
	"github.com/pehcastro/twind/twi/scene"
	"github.com/pehcastro/twind/twi/style"
)

func token(t *testing.T, oklch string) color.RGBA {
	t.Helper()
	c, err := color.Parse(oklch)
	if err != nil {
		t.Fatal(err)
	}
	return c.RGBA
}

func within1(a, b color.RGBA) bool {
	d := func(x, y uint8) bool { return x-y <= 1 || y-x <= 1 }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B) && d(a.A, b.A)
}

func indigoToPink(t *testing.T, dir style.GradientDirection) style.ComputedStyle {
	s := plain()
	s.Gradient = style.Gradient{
		GradientLine: style.GradientLine{Kind: style.GradientLinear, Direction: dir},
		From:         style.GradientStop{Color: literal(token(t, "oklch(51.1% 0.262 276.966)")), Position: 0},
		To:           style.GradientStop{Color: literal(token(t, "oklch(59.2% 0.249 0.584)")), Position: 1},
	}
	return s
}

func TestGradientHorizontal(t *testing.T) {
	s := indigoToPink(t, style.ToRight)
	buf := painted(50, 1, page(50, 1, "", scene.New(place(0, 0, 50, 1, layout.Edges{}), s, scene.Text{})), Composited)
	if got := rows(buf)[0]; got != strings.Repeat(" ", 50) {
		t.Errorf("horizontal gradient row %q, want spaces", got)
	}
	if got, want := buf.At(0, 0).Bg.RGBA, s.Gradient.From.Color.RGBA; !within1(got, want) {
		t.Errorf("first cell %+v, want indigo-600 %+v", got, want)
	}
	if got, want := buf.At(49, 0).Bg.RGBA, s.Gradient.To.Color.RGBA; !within1(got, want) {
		t.Errorf("last cell %+v, want pink-600 %+v", got, want)
	}
	left, right := buf.At(24, 0).Bg.RGBA, buf.At(25, 0).Bg.RGBA
	mid := color.RGBA{R: uint8((int(left.R) + int(right.R) + 1) / 2), G: uint8((int(left.G) + int(right.G) + 1) / 2), B: uint8((int(left.B) + int(right.B) + 1) / 2), A: 255}
	if oklabMidpoint := (color.RGBA{R: 157, G: 69, B: 185, A: 255}); !within1(mid, oklabMidpoint) {
		t.Errorf("middle of the row %+v (cells 24 %+v and 25 %+v), want the OKLab midpoint %+v", mid, left, right, oklabMidpoint)
	}
}

func TestGradientVertical(t *testing.T) {
	s := indigoToPink(t, style.ToBottom)
	tree := page(3, 4, "", scene.New(place(0, 0, 3, 4, layout.Edges{}), s, scene.Text{}))
	buf := painted(3, 4, tree, Composited)
	expect(t, buf, "▀▀▀", "▀▀▀", "▀▀▀", "▀▀▀")
	var halves []color.RGBA
	for y := range 4 {
		halves = append(halves, buf.At(1, y).Fg.RGBA, buf.At(1, y).Bg.RGBA)
	}
	seen := map[color.RGBA]bool{}
	for _, h := range halves {
		seen[h] = true
	}
	if len(seen) != 8 {
		t.Errorf("4-row vertical gradient gave %d distinct colours %+v, want 8", len(seen), halves)
	}
	if !within1(halves[0], s.Gradient.From.Color.RGBA) || !within1(halves[7], s.Gradient.To.Color.RGBA) {
		t.Errorf("top half %+v and bottom half %+v, want the from and to tokens", halves[0], halves[7])
	}
	if c := painted(3, 4, tree, Plain).At(1, 1); c.Grapheme != " " || c.Bg != literal(zinc100) {
		t.Errorf("gradient under Plain: %+v, want the parent's fill", c)
	}
	expect(t, painted(3, 1, page(3, 1, "", scene.New(place(0, 0, 0, 1, layout.Edges{}), s, scene.Text{})), Composited), "   ")
}

func TestGradientTextTakesCellMean(t *testing.T) {
	s := indigoToPink(t, style.ToBottom)
	buf := painted(3, 4, page(3, 4, "", scene.New(place(0, 0, 3, 4, layout.Edges{}), s, scene.Sanitize("a"))), Composited)
	bare := painted(3, 4, page(3, 4, "", scene.New(place(0, 0, 3, 4, layout.Edges{}), s, scene.Text{})), Composited)
	top, bottom := bare.At(0, 0).Fg.RGBA, bare.At(0, 0).Bg.RGBA
	mean := color.RGBA{R: uint8((int(top.R) + int(bottom.R)) / 2), G: uint8((int(top.G) + int(bottom.G)) / 2), B: uint8((int(top.B) + int(bottom.B)) / 2), A: 255}
	if c := buf.At(0, 0); c.Grapheme != "a" || !within1(c.Bg.RGBA, mean) {
		t.Errorf("text over a half-block cell: %q bg %+v, want a on the mean %+v", c.Grapheme, c.Bg.RGBA, mean)
	}
}

func TestAlign(t *testing.T) {
	for _, tc := range []struct {
		align style.TextAlign
		text  string
		want  []string
	}{
		{style.TextLeft, "abc", []string{"abc" + strings.Repeat(" ", 17)}},
		{style.TextRight, "abc", []string{strings.Repeat(" ", 17) + "abc"}},
		{style.TextCenter, "abc", []string{strings.Repeat(" ", 8) + "abc" + strings.Repeat(" ", 9)}},
		{style.TextJustify, "abc", []string{"abc" + strings.Repeat(" ", 17)}},
		{style.TextRight, "abc\nde", []string{strings.Repeat(" ", 17) + "abc", strings.Repeat(" ", 18) + "de"}},
		{style.TextRight, "日本", []string{strings.Repeat(" ", 16) + "日本"}},
		{style.TextCenter, strings.Repeat("x", 25), []string{strings.Repeat("x", 20), strings.Repeat(" ", 7) + "xxxxx" + strings.Repeat(" ", 8)}},
	} {
		s := plain()
		s.TextAlign = tc.align
		buf := painted(20, 2, page(20, 2, "", scene.New(place(0, 0, 20, 2, layout.Edges{}), s, scene.Sanitize(tc.text))), Composited)
		got := rows(buf)
		for i, want := range tc.want {
			if got[i] != want {
				t.Errorf("align %d %q row %d: %q, want %q", tc.align, tc.text, i, got[i], want)
			}
		}
	}
}

func TestAlignTruncate(t *testing.T) {
	long := "abcdefghijklmnopqrstuvwxyz0123"
	n := scene.New(place(0, 0, 10, 2, layout.Edges{}), plain(), scene.Sanitize(long))
	n.Truncate = true
	buf := painted(10, 2, page(10, 2, "", n), Composited)
	expect(t, buf, "abcdefghi…", "          ")
	if buf.At(9, 0).Grapheme != "…" {
		t.Errorf("x 9 holds %q, want the ellipsis", buf.At(9, 0).Grapheme)
	}
	short := scene.New(place(0, 0, 10, 1, layout.Edges{}), plain(), scene.Sanitize("fits"))
	short.Truncate = true
	expect(t, painted(10, 1, page(10, 1, "", short), Composited), "fits      ")
}

func TestPill(t *testing.T) {
	green100, green700 := token(t, "oklch(96.2% 0.044 156.743)"), token(t, "oklch(52.7% 0.154 150.069)")
	pill := func(h int) scene.Node {
		s := filled(green100)
		s.Color, s.Radius = literal(green700), style.RadiusFull
		box := &layout.Box{
			BorderBox: layout.Rect{X: 1, Y: 0, W: 8, H: h}, PaddingBox: layout.Rect{X: 1, Y: 0, W: 8, H: h},
			ContentBox: layout.Rect{X: 2, Y: 0, W: 6, H: h}, Clip: layout.Rect{W: 100, H: 100},
		}
		return scene.New(box, s, scene.Sanitize("Active"))
	}
	buf := painted(10, 1, page(10, 1, "", pill(1)), Composited)
	expect(t, buf, "  Active  ")
	for _, x := range []int{1, 8} {
		if c := buf.At(x, 0); c.Bg != literal(green100) {
			t.Errorf("end of the pill at x %d: bg %+v, want the pill colour", x, c.Bg)
		}
	}
	if c := buf.At(2, 0); c.Fg != literal(green700) || c.Bg != literal(green100) {
		t.Errorf("label cell: fg %+v bg %+v, want green-700 on green-100", c.Fg, c.Bg)
	}
	expect(t, painted(10, 1, page(10, 1, "", pill(1)), Plain), "  Active  ")
	expect(t, painted(10, 2, page(10, 2, "", pill(2)), Composited), "  Active  ", "          ")
	clipped := pill(1)
	clipped.Clip = layout.Rect{X: 2, W: 6, H: 1}
	expect(t, painted(10, 1, page(10, 1, "", clipped), Composited), "  Active  ")
}
