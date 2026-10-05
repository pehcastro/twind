package render_test

import (
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/twi/style"
)

func TestHitMatchesAWalkOfEveryNode(t *testing.T) {
	moved, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "row", Decls: []style.Declaration{{Property: style.PropDisplay, Display: style.DisplayFlex}, {Property: style.PropDirection, Direction: style.Row}}},
		{Class: "w-6", Decls: []style.Declaration{{Property: style.PropWidth, Length: style.Length{Unit: style.Cells, Value: 6}}}},
		{Class: "shrink-0", Decls: []style.Declaration{{Property: style.PropShrink}}},
		{Class: "slide", Decls: []style.Declaration{{Property: style.PropTranslateX, Length: style.Length{Unit: style.Cells, Value: 5}}, {Property: style.PropTranslateY, Length: style.Length{Unit: style.Cells, Value: 2}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		sheet  style.Sheet
		root   render.Node
		scroll []int
	}{
		{"absolute menu in an overflow-hidden card", topSheet(t), node("page", node("card", text("card"), menuNode(0)), node("later", text("later"))), nil},
		{"top-layer menu", topSheet(t), node("page", node("card", text("card"), menuNode(1)), node("later", text("later"))), nil},
		{"menu in a scrolled container", topSheet(t), node("page", node("scroller", text("a"), node("relative", text("b"), menuNode(0)), text("c"), text("d"), text("e"), text("f")), node("later", text("later"))), []int{0}},
		{"fixed backdrop in a scroller", topSheet(t), node("page", node("card", text("above")), node("scroller", node("backdrop"), text("a"), text("b"), text("c"), text("d"), text("e"))), []int{1}},
		{"translated track", moved, node("row", node("row slide", node("w-6 shrink-0", text("one")), node("w-6 shrink-0", text("two")), node("w-6 shrink-0", text("three")))), nil},
	}
	for _, c := range cases {
		var tree render.Tree
		f := render.Frame{Sheet: c.sheet, Width: 16, Height: layout.Length{Unit: layout.Cells, Value: 8}}
		if _, err := tree.Scene(c.root, f); err != nil {
			t.Fatal(err)
		}
		if c.scroll != nil {
			tree.ScrollBy(c.scroll, 0, 1)
		}
		sc, err := tree.Scene(c.root, f)
		if err != nil {
			t.Fatal(err)
		}
		var w scene.Walker
		for y := range 8 {
			for x := range 16 {
				hits := func(n *scene.Node) bool {
					in := func(r layout.Rect) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }
					return in(n.Bounds) && in(n.Clip) && n.Visibility == style.Visible && n.PointerEvents != style.PointerNone
				}
				var want *scene.Node
				new(scene.Walker).Walk(&sc, func(n *scene.Node) {
					if hits(n) {
						want = n
					}
				}, func(_ *scene.Node, inside func()) { inside() })
				if got := w.Hit(&sc, x, y, hits); got != want {
					t.Errorf("%s: hit at %d,%d is %v, a walk of every node finds %v", c.name, x, y, bounds(got), bounds(want))
				}
			}
		}
	}
}

func bounds(n *scene.Node) any {
	if n == nil {
		return nil
	}
	return n.Bounds
}
