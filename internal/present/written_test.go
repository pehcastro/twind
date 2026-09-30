package present

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/twind-dev/twind/internal/present/demo"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/terminal"
)

const chunk = 1 << 10

func written(t *testing.T) []string {
	var lines []string
	record := func(name string, g terminal.Graphics, p color.Profile, margins bool, trees ...scene.Node) {
		s, out := screen(g)
		s.Profile, s.Margins = p, margins
		for _, root := range trees {
			frame(t, s, root)
		}
		for i, f := range out.frames {
			var sums []string
			for at := 0; at < len(f); at += chunk {
				sum := sha256.Sum256(f[at:min(at+chunk, len(f))])
				sums = append(sums, hex.EncodeToString(sum[:4]))
			}
			lines = append(lines, fmt.Sprintf("%s %d %d %s", name, i, len(f), strings.Join(sums, ",")))
		}
	}
	dialog := tree(t, demo.Dialog())
	record("dialog-sixel", terminal.GraphicsSixel, color.TrueColor, false, dialog, tree(t, withText(demo.Dialog(), "Dashboard  Projects  Settings", "Dashboard")))
	record("dialog-kitty", terminal.GraphicsKitty, color.TrueColor, false, dialog)
	record("dialog-iterm2", terminal.GraphicsITerm2, color.TrueColor, false, dialog)
	record("dialog-ansi256", terminal.GraphicsSixel, color.ANSI256, false, dialog)
	record("hover", terminal.GraphicsSixel, color.TrueColor, false, tree(t, demo.List(0)), tree(t, demo.List(1)), tree(t, demo.List(2)), tree(t, demo.List(1)))
	steps := scrolled(t, 0, 1, 2, 1, 0, 10)
	record("scroll-margins", terminal.GraphicsSixel, color.TrueColor, true, steps...)
	record("scroll-plain", terminal.GraphicsSixel, color.TrueColor, false, steps...)
	record("scroll-overlaid", terminal.GraphicsSixel, color.TrueColor, true, overlaid(0), overlaid(1), overlaid(2), overlaid(1))
	page, group := flatPage(color.RGBA{R: 255, G: 255, B: 255, A: 255}, ""), flatPage(color.RGBA{}, "")
	a, b := flatPage(color.RGBA{A: 255}, ""), flatPage(color.RGBA{R: 200, A: 128}, "")
	group.Opacity, group.Bounds = 0.5, layout.Rect{W: 10, H: 4}
	a.Bounds, b.Bounds = layout.Rect{W: 6, H: 2}, layout.Rect{X: 3, Y: 1, W: 6, H: 2}
	group.Children, page.Children = []scene.Node{a, b}, []scene.Node{group}
	record("group", terminal.GraphicsSixel, color.TrueColor, false, page)
	return lines
}

func TestWrittenBytesUnchanged(t *testing.T) {
	fixture, err := os.ReadFile("testdata/written.txt")
	if err != nil {
		t.Fatal(err)
	}
	want, got := strings.Split(strings.TrimSpace(string(fixture)), "\n"), written(t)
	if len(got) != len(want) {
		t.Fatalf("%d frames written, want %d", len(got), len(want))
	}
	for i := range got {
		g, w := strings.Fields(got[i]), strings.Fields(strings.TrimSpace(want[i]))
		if g[2] != w[2] {
			t.Errorf("%s frame %s: %s bytes, want %s", g[0], g[1], g[2], w[2])
		}
		gs, ws := strings.Split(g[3], ","), strings.Split(w[3], ",")
		for c := range min(len(gs), len(ws)) {
			if gs[c] != ws[c] {
				t.Errorf("%s frame %s: first difference in bytes %d to %d", g[0], g[1], c*chunk, (c+1)*chunk)
				break
			}
		}
	}
}
