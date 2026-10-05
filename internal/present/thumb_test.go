package present

import (
	"testing"

	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/layout"
	"github.com/pehcastro/twind/twi/scene"
	"github.com/pehcastro/twind/twi/terminal"
)

func TestScrollThumbStaysUnderAnOpenSheet(t *testing.T) {
	const w, h = 20, 10
	fill := func(r, g, b uint8) color.Color {
		return color.Color{Kind: color.Literal, RGBA: color.RGBA{R: r, G: g, B: b, A: 255}}
	}
	page, red := fill(10, 10, 10), fill(200, 0, 0)
	screenRect := layout.Rect{W: w, H: h}
	row := scene.Node{Bounds: layout.Rect{W: w, H: 1}, Clip: screenRect, Opacity: 1, Background: fill(40, 40, 40)}
	scroller := scene.Node{Bounds: screenRect, Padding: screenRect, Clip: screenRect, Scroll: true, Opacity: 1, ScrollContent: layout.Rect{W: w, H: 4 * h}, Children: []scene.Node{row}, Foreground: fill(250, 250, 250)}
	sheet := scene.Node{Bounds: layout.Rect{X: w / 2, W: w / 2, H: h}, Clip: screenRect, Position: layout.PositionFixed, Opacity: 1, Background: red}
	root := func(children ...scene.Node) scene.Node {
		return scene.Node{Bounds: screenRect, Clip: screenRect, Opacity: 1, Background: page, Children: children}
	}
	s, _ := screen(terminal.GraphicsNone)
	if err := s.Frame(root(scroller), w, h); err != nil {
		t.Fatal(err)
	}
	if c := s.want.At(w-1, 0); c.Grapheme == " " {
		t.Fatalf("no sheet: the thumb cell is %+v, want the thumb over the page and the row under it", c)
	}
	s, _ = screen(terminal.GraphicsNone)
	if err := s.Frame(root(scroller, sheet), w, h); err != nil {
		t.Fatal(err)
	}
	for y := range 3 {
		if c := s.want.At(w-1, y); c.Grapheme != " " || c.Bg != red {
			t.Errorf("row %d: the thumb column under the open sheet shows %q on %v, want the sheet's blank red cell", y, c.Grapheme, c.Bg.RGBA)
		}
	}
}

func TestScrollbarsWithoutAScrollerAllocateNothing(t *testing.T) {
	const w, h = 20, 10
	screenRect := layout.Rect{W: w, H: h}
	leaf := scene.Node{Bounds: layout.Rect{W: w, H: 1}, Clip: screenRect, Opacity: 1}
	row := scene.Node{Bounds: layout.Rect{W: w, H: 2}, Clip: screenRect, Opacity: 1, Children: []scene.Node{leaf, leaf}}
	root := scene.Node{Bounds: screenRect, Clip: screenRect, Opacity: 1, Children: []scene.Node{row, row, row}}
	s, _ := screen(terminal.GraphicsNone)
	if err := s.Frame(root, w, h); err != nil {
		t.Fatal(err)
	}
	if allocs := testing.AllocsPerRun(10, func() { s.scrollbars(&root) }); allocs != 0 {
		t.Errorf("scrollbars over a tree with no scroller allocated %v times per frame, want 0", allocs)
	}
}
