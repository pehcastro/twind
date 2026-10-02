package scene

import (
	"testing"

	"github.com/twind-dev/twind/twi/layout"
)

func TestHitFindsChildrenOutsideTheirParent(t *testing.T) {
	at := func(x, y, w, h int) Node {
		r := layout.Rect{X: x, Y: y, W: w, H: h}
		return Node{Bounds: r, Clip: layout.Rect{W: 20, H: 10}, Opacity: 1}
	}
	inside := func(x, y int) func(*Node) bool {
		return func(n *Node) bool {
			b := n.Bounds
			return x >= b.X && x < b.X+b.W && y >= b.Y && y < b.Y+b.H
		}
	}
	for _, sealed := range []bool{false, true} {
		root := at(0, 0, 20, 10)
		card := at(2, 2, 4, 2)
		card.Children = []Node{at(10, 6, 3, 2)}
		card.Children[0].Position = layout.PositionAbsolute
		root.Children = []Node{card, at(0, 8, 20, 2)}
		if sealed {
			card.Children[0].Enclose()
			root.Children[0].Enclose()
			root.Children[1].Enclose()
			root.Enclose()
		}
		var w Walker
		for _, c := range []struct {
			x, y int
			want *Node
		}{
			{11, 7, &root.Children[0].Children[0]},
			{3, 3, &root.Children[0]},
			{11, 9, &root.Children[1]},
			{15, 4, &root},
		} {
			if got := w.Hit(&root, c.x, c.y, inside(c.x, c.y)); got != c.want {
				t.Errorf("sealed %v: hit at %d,%d is %+v, want %+v", sealed, c.x, c.y, got, c.want)
			}
		}
	}
}
