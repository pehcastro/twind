package scene

import (
	"image"
	"slices"
	"testing"

	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/text"
)

func TestTopLayerPaintsLastInOpenOrder(t *testing.T) {
	white := paint(255, 255, 255, 255)
	first := box(2, 1, 6, 3, white)
	first.HidesOverflow, first.TopLayer = true, 2
	first.Children = []Node{box(2, 1, 6, 1, paint(0, 0, 255, 255))}
	first.Children[0].Clip = first.Padding
	second := box(20, 2, 8, 3, white)
	second.Position, second.TopLayer = layout.PositionAbsolute, 1
	clipped := track(style.RadiusLg, first)
	clipped.Bounds, clipped.Padding, clipped.Content = layout.Rect{X: 2, Y: 1, W: 10, H: 4}, layout.Rect{X: 2, Y: 1, W: 10, H: 4}, layout.Rect{X: 2, Y: 1, W: 10, H: 4}
	later := box(0, 2, 40, 3, paint(255, 0, 0, 255))
	later.Position, later.ZIndex = layout.PositionRelative, 10
	root := page(clipped, track(style.RadiusNone, second), later)
	root.Children[0].Children[0].Clip = viewport
	root.Children[1].Children[0].Clip = viewport

	var order []*Node
	Walk(&root, func(n *Node) { order = append(order, n) }, func(_ *Node, inside func()) { inside() })
	at := func(n *Node) int { return slices.Index(order, n) }
	top, open, inner := &root.Children[0].Children[0], &root.Children[1].Children[0], &root.Children[0].Children[0].Children[0]
	if got := []int{at(&root.Children[2]), at(open), at(top), at(inner)}; !slices.IsSorted(got) || slices.Contains(got, -1) {
		t.Errorf("paint order of the later z-10, top layer 1, top layer 2 and its row: %v, want rising", got)
	}
	f := record(root)
	var visuals []image.Rectangle
	for _, b := range f.Layers[len(f.Layers)-1].Boxes {
		visuals = append(visuals, b.Visual)
	}
	if want := []image.Rectangle{f.pixels(open.Bounds), f.pixels(top.Bounds), f.pixels(inner.Bounds)}; len(visuals) < 3 || !slices.Equal(visuals[len(visuals)-3:], want) {
		t.Errorf("the last layer draws %v, want it to end with %v", visuals, want)
	}
	for _, l := range f.Layers {
		for _, op := range l.Ops {
			if op.Kind == raster.Clip && op.Box.Radii != [4]float64{} {
				t.Errorf("a rounded mask %+v reached the top layer from the card it escapes", op.Box)
			}
		}
	}
}

func TestFixedEscapesTheCardMask(t *testing.T) {
	dialog := box(2, 1, 6, 3, paint(255, 255, 255, 255))
	dialog.HidesOverflow, dialog.Position = true, layout.PositionFixed
	dialog.Children = []Node{box(2, 1, 6, 1, paint(0, 0, 255, 255))}
	dialog.Children[0].Clip = dialog.Padding
	card := track(style.RadiusLg, dialog)
	card.Bounds, card.Padding, card.Content = layout.Rect{X: 2, Y: 1, W: 10, H: 4}, layout.Rect{X: 2, Y: 1, W: 10, H: 4}, layout.Rect{X: 2, Y: 1, W: 10, H: 4}
	card.Children[0].Clip = viewport
	for _, l := range record(page(card)).Layers {
		for _, op := range l.Ops {
			if op.Kind == raster.Clip && op.Box.Radii != [4]float64{} {
				t.Errorf("a rounded mask %+v reached a fixed dialog's row from the card it escapes", op.Box)
			}
		}
	}
}

func TestTopLayerLeavesTheScrollerLayer(t *testing.T) {
	root := scroller(4)
	menu := box(8, 5, 10, 4, paint(255, 255, 255, 255))
	menu.TopLayer = 1
	view := &root.Children[0]
	view.Children = append(view.Children, menu)
	f := record(root)
	for i, l := range f.Layers {
		for _, b := range l.Boxes {
			if b.Visual.Add(l.Origin) == f.pixels(menu.Bounds) && (l.scroller || l.Clip != f.pixels(viewport)) {
				t.Errorf("the top layer menu is in layer %d (scroller %v, clip %v), want a page layer clipped by the screen", i, l.scroller, l.Clip)
			}
		}
	}
}

func TestVisibilityHiddenKeepsBoundsPaintsNothing(t *testing.T) {
	r := layout.Rect{X: 2, Y: 1, W: 10, H: 3}
	b := &layout.Box{BorderBox: r, PaddingBox: r, ContentBox: r, Clip: viewport, Style: layout.Style{Border: layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}}}
	red := paint(255, 0, 0, 255)
	s := style.ComputedStyle{Visibility: style.Hidden, Opacity: 1, Background: red, BorderStyle: style.BorderSingle, BorderColor: red, Color: red, Shadows: []style.Shadow{{Y: 4, Blur: 6, Color: red}}}
	n := New(b, s, Sanitize("secret"))
	if n.Bounds != r || n.Visibility != style.Hidden {
		t.Errorf("invisible box: bounds %+v visibility %d, want %+v and hidden", n.Bounds, n.Visibility, r)
	}
	if n.Lines(text.Widths{}) != nil || shows(n.Background) || len(n.Shadows) > 0 || n.Border.Style != style.BorderNone {
		t.Errorf("invisible box paints: lines %q background %+v shadows %d border %d", n.Lines(text.Widths{}), n.Background, len(n.Shadows), n.Border.Style)
	}
	if boxes := record(page(n)).Layers[0].Boxes; len(boxes) != 1 {
		t.Errorf("recorded %d boxes, want the page alone", len(boxes))
	}
}

func TestPointerEventsCarried(t *testing.T) {
	r := layout.Rect{W: 4, H: 1}
	for _, want := range []style.PointerEvents{style.PointerAuto, style.PointerNone} {
		n := New(&layout.Box{BorderBox: r}, style.ComputedStyle{Opacity: 1, PointerEvents: want}, Text{})
		if n.PointerEvents != want {
			t.Errorf("pointer-events %d, want %d", n.PointerEvents, want)
		}
	}
}
