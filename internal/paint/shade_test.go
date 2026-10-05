package paint

import (
	"slices"
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/buffer"
	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/style"
)

func shadowMdPair(alpha uint8) []style.Shadow {
	ink := literal(color.RGBA{A: alpha})
	return []style.Shadow{{Y: 4, Blur: 6, Spread: -1, Color: ink}, {Y: 2, Blur: 4, Spread: -2, Color: ink}}
}

func shaded(id terminal.Identity, root scene.Node) *buffer.Buffer {
	p := Painter{Profile: color.TrueColor, Identity: id}
	buf := buffer.New(root.Bounds.W, root.Bounds.H)
	p.Paint(buf, &root, Composited)
	return buf
}

func cardAt(x, y int, text string, shadows ...style.Shadow) scene.Node {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	return page(14, 7, text, card(place(x, y, 8, 4, one), style.RadiusLg, shadows...))
}

func TestShadeZedTwoSteps(t *testing.T) {
	soft := []string{
		"              ",
		" ░▒╭──────╮▒░ ",
		" ░▒│      │▒░ ",
		" ░▒│      │▒░ ",
		" ░▒╰──────╯▒░ ",
		"  ░▒▒▒▒▒▒▒▒░  ",
		"   ░░░░░░░░   ",
	}
	md := shadowMdPair(26)
	for _, shadows := range [][]style.Shadow{md, {md[1], md[0]}} {
		buf := shaded(terminal.IdentityZed, cardAt(3, 1, "", shadows...))
		t.Logf("zed:\n%s", strings.Join(rows(buf), "\n"))
		expect(t, buf, soft...)
	}
	buf := shaded(terminal.IdentityZed, cardAt(3, 1, "", md...))
	ink := over(literal(color.RGBA{A: 52}), literal(zinc100))
	for _, at := range [][2]int{{11, 2}, {12, 2}, {2, 3}, {5, 5}, {5, 6}, {2, 5}} {
		if c := buf.At(at[0], at[1]); c.Fg != ink || c.Bg != literal(zinc100) {
			t.Errorf("shade at %v: fg %+v bg %+v, want the shadow at twice its alpha %+v over the page", at, c.Fg.RGBA, c.Bg.RGBA, ink.RGBA)
		}
	}
}

func TestShadeZedKeepsText(t *testing.T) {
	buf := shaded(terminal.IdentityZed, cardAt(3, 1, "abcdefghijklmn", shadowMdPair(26)...))
	if got := rows(buf)[0]; got != "abcdefghijklmn" {
		t.Errorf("row 0 %q, want the page text untouched", got)
	}
	ink := plain()
	ink.Color = literal(zinc950)
	root := cardAt(3, 1, "", shadowMdPair(26)...)
	root.Children = append([]scene.Node{scene.New(place(0, 1, 14, 6, layout.Edges{}), ink, scene.Sanitize("xyz\nxyz\nxyz\nxyz\nxyz\nxyz"))}, root.Children...)
	buf = shaded(terminal.IdentityZed, root)
	for y := 1; y < 7; y++ {
		if got := rows(buf)[y][:3]; got != "xyz" {
			t.Errorf("row %d starts %q, want the text under the shadow kept", y, got)
		}
	}
}

func TestShadeZedTransparentShadow(t *testing.T) {
	buf := shaded(terminal.IdentityZed, cardAt(3, 1, "", shadowMdPair(0)...))
	expect(t, buf, "              ", "   ╭──────╮   ", "   │      │   ", "   │      │   ", "   ╰──────╯   ", "              ", "              ")
}

func TestShadeZedRingUnchanged(t *testing.T) {
	ring := style.Shadow{Spread: 2, Color: literal(zinc800)}
	zed, other := shaded(terminal.IdentityZed, cardAt(3, 1, "", ring)), shaded(terminal.IdentityOther, cardAt(3, 1, "", ring))
	for y := range 7 {
		if !slices.Equal(zed.Row(y), other.Row(y)) {
			t.Errorf("ring row %d: zed %q, other %q", y, rows(zed)[y], rows(other)[y])
		}
	}
}

func TestShadeZedMovedLeavesNoTail(t *testing.T) {
	p := Painter{Profile: color.TrueColor, Identity: terminal.IdentityZed}
	buf := buffer.New(14, 7)
	first, second := cardAt(5, 1, "", shadowMdPair(26)...), cardAt(2, 0, "", shadowMdPair(26)...)
	p.Paint(buf, &first, Composited)
	p.Paint(buf, &second, Composited)
	fresh := shaded(terminal.IdentityZed, second)
	for y := range 7 {
		if !slices.Equal(buf.Row(y), fresh.Row(y)) {
			t.Errorf("row %d after the move %q, fresh paint %q", y, rows(buf)[y], rows(fresh)[y])
		}
	}
}

func TestShadeOtherUnchanged(t *testing.T) {
	buf := shaded(terminal.IdentityOther, cardAt(3, 1, "", shadowMdPair(26)...))
	expect(t, buf, "              ", "  ▐╭──────╮▌  ", "  ▐│      │▌  ", "  ▐│      │▌  ", "  ▐╰──────╯▌  ", "   ▀▀▀▀▀▀▀▀   ", "              ")
	half := color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 219, G: 219, B: 220, A: 255}}
	for _, at := range [][2]int{{2, 1}, {11, 4}, {3, 5}, {10, 5}} {
		if c := buf.At(at[0], at[1]); c.Fg != half || c.Bg != literal(zinc100) {
			t.Errorf("half block at %v: fg %+v bg %+v, want today's %+v over the page", at, c.Fg.RGBA, c.Bg.RGBA, half.RGBA)
		}
	}
}
