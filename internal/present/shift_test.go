package present

import (
	"slices"
	"testing"

	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi/color"
)

func repainted(s *Screen) int {
	n := 0
	for _, r := range s.painter.Repainted() {
		n += r.W * r.H
	}
	return n
}

func sameText(t *testing.T, name string, s *Screen, root scene.Node, g terminal.Graphics) {
	t.Helper()
	fresh, _ := screen(g)
	frame(t, fresh, root)
	for y := range rows {
		if !slices.Equal(s.text.Row(y), fresh.text.Row(y)) {
			t.Fatalf("%s: painted row %d is %q, a fresh paint is %q", name, y, cells(s.text.Row(y)), cells(fresh.text.Row(y)))
		}
	}
}

func TestScrollRepaintsOnlyTheExposedRows(t *testing.T) {
	offsets := []int{1, 2, 3, 2, 1, 0, 1}
	trees := scrolled(t, offsets...)
	for _, g := range []terminal.Graphics{terminal.GraphicsSixel, terminal.GraphicsNone} {
		s, _ := screen(g)
		frame(t, s, scrolled(t, 0)[0])
		for i, root := range trees {
			frame(t, s, root)
			sameText(t, "offset", s, root, g)
			if v := view(root); repainted(s) > 2*v.W {
				t.Errorf("graphics %d, offset %d: %d cells repainted, want at most the exposed row of %d cells and its tile edges", g, offsets[i], repainted(s), v.W)
			}
		}
	}
}

func TestScrollWithAChangedRowRepaintsThatRow(t *testing.T) {
	trees := scrolled(t, 0, 1)
	s, _ := screen(terminal.GraphicsSixel)
	frame(t, s, trees[0])
	root := trees[1]
	list := &root.Children[0].Children[1]
	for i := range list.Children {
		if row := &list.Children[i]; row.Bounds.Y == view(root).Y+5 {
			row.Background = color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 200, G: 30, B: 30, A: 255}}
		}
	}
	frame(t, s, root)
	sameText(t, "a row changed while scrolling", s, root, terminal.GraphicsSixel)
}

func TestScrollUnderStaticLayersRepaintsThem(t *testing.T) {
	s, _ := screen(terminal.GraphicsSixel)
	for _, offset := range []int{0, 1, 2, 3, 2, 1} {
		root := overlaid(offset)
		frame(t, s, root)
		sameText(t, "static layers", s, root, terminal.GraphicsSixel)
	}
}
