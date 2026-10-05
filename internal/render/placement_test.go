package render_test

import (
	"image"
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/render/testdata/sheet"
	"github.com/pehcastro/twind/twi/buffer"
	"github.com/pehcastro/twind/twi/layout"
	"github.com/pehcastro/twind/twi/paint"
	"github.com/pehcastro/twind/twi/scene"
)

func TestPlacementInsideRelativeParent(t *testing.T) {
	var tree render.Tree
	f := cssFrame(t, 70)
	f.Height = layout.Length{Unit: layout.Cells, Value: 16}
	at := func(x, y int) *image.Point { return &image.Point{X: x, Y: y} }
	for _, step := range []struct {
		name     string
		classes  string
		at       *image.Point
		menu     image.Point
		row      image.Point
		cascades int
	}{
		{"in flow", sheet.Button, nil, image.Pt(5, 2), image.Pt(5, 3), -1},
		{"placed at (30, 7)", sheet.Button, at(30, 7), image.Pt(35, 9), image.Pt(5, 2), 2},
		{"same place", sheet.Button, at(30, 7), image.Pt(35, 9), image.Pt(5, 2), 0},
		{"moved to (10, 3)", sheet.Button, at(10, 3), image.Pt(15, 5), image.Pt(5, 2), 2},
		{"popup classes placed", sheet.Button + " " + sheet.Popup, at(30, 7), image.Pt(35, 9), image.Pt(5, 2), -1},
		{"fixed dialog placed", "fixed top-4 left-10 w-20 border", at(3, 1), image.Pt(3, 1), image.Pt(5, 2), -1},
		{"back in flow", sheet.Button, nil, image.Pt(5, 2), image.Pt(5, 3), -1},
	} {
		menu := node(step.classes, text("Copy"))
		menu.At = step.at
		before := tree.Cascades()
		root, err := tree.Scene(node(sheet.Page, node(sheet.Context, menu, text("row"))), f)
		if err != nil {
			t.Fatal(err)
		}
		buf := buffer.New(f.Width, f.Height.Value)
		paint.Paint(buf, root, paint.Composited)
		rows := rowsOf(buf)
		t.Logf("%s, 70x16:\n%s", step.name, strings.Join(rows, "\n"))
		context := root.Children[0]
		got := func(n scene.Node) image.Point { return image.Pt(n.Bounds.X, n.Bounds.Y) }
		if m := got(context.Children[0]); m != step.menu {
			t.Errorf("%s: menu at %v, want %v", step.name, m, step.menu)
		}
		if r := got(context.Children[1]); r != step.row {
			t.Errorf("%s: sibling at %v, want %v", step.name, r, step.row)
		}
		if w := context.Children[0].Bounds.W; step.at != nil && (w <= 0 || w >= 40) {
			t.Errorf("%s: menu %d wide, want its content width", step.name, w)
		}
		if step.cascades >= 0 && tree.Cascades()-before != step.cascades {
			t.Errorf("%s: %d cascades, want %d", step.name, tree.Cascades()-before, step.cascades)
		}
	}
}
