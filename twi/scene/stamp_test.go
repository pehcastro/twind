package scene

import (
	"image"
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi/style"
)

func TestCellChangeDropsStamps(t *testing.T) {
	pill := func(w, h int) Node {
		p := box(4, 2, w, h, paint(250, 250, 250, 255))
		p.Border.Radius = style.RadiusLg
		return page(p)
	}
	wide, tall := pill(2, 1), pill(1, 2)
	var reused, fresh Frame
	reused.Record(&wide, image.Pt(10, 20))
	reused.Record(&tall, image.Pt(20, 10))
	fresh.Record(&tall, image.Pt(20, 10))
	var got, want strings.Builder
	dump(&got, &reused)
	dump(&want, &fresh)
	if got.String() != want.String() {
		t.Errorf("a 20x20 pill recorded after a cell change kept the old cell's radius:\n%s\nwant\n%s", got.String(), want.String())
	}
}

func TestStampHoldsOnlyItsOwnLook(t *testing.T) {
	base := card(paint(24, 24, 27, 255))
	base.Border = Border{Style: style.BorderSingle, Radius: style.RadiusLg, Top: true, Right: true, Bottom: true, Left: true, Color: paint(63, 63, 70, 255)}
	f := record(page(base))
	size := f.pixels(base.Bounds).Size()
	var kept *stamp
	for i := range f.stamps.slots {
		if s := &f.stamps.slots[i]; s.size == size {
			kept = s
		}
	}
	if kept == nil {
		t.Fatal("recording a card kept no stamp for its size")
	}
	if !kept.holds(&f.stamps, &base, size, kept.visual) {
		t.Fatal("the stamp does not hold the card it was kept from")
	}
	for name, change := range map[string]func(n *Node){
		"border colour": func(n *Node) { n.Border.Color = paint(1, 2, 3, 255) },
		"border style":  func(n *Node) { n.Border.Style = style.BorderDashed },
		"radius":        func(n *Node) { n.Border.Radius = style.RadiusSm },
		"one edge":      func(n *Node) { n.Border.Left = false },
		"background":    func(n *Node) { n.Background = paint(39, 39, 42, 255) },
		"shadow blur":   func(n *Node) { n.Shadows = []style.Shadow{{X: 8, Y: 16, Blur: 8, Color: paint(0, 0, 0, 64)}} },
		"inset shadow":  func(n *Node) { n.InsetShadows = []style.Shadow{{Y: 2, Color: paint(0, 0, 0, 40)}} },
	} {
		n := base
		change(&n)
		if kept.holds(&f.stamps, &n, size, kept.visual) {
			t.Errorf("a card that differs in %s is served the first card's stamp", name)
		}
	}
}
