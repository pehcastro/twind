package scene

import (
	"image"
	"testing"

	"github.com/twind-dev/twind/twi/layout"
)

func scroller(offset int) Node {
	view := box(5, 1, 20, 6, paint(24, 24, 27, 255))
	view.Scroll, view.ScrollContent = true, layout.Rect{X: 5, Y: 1 - offset, W: 20, H: 30}
	for i := range 30 {
		row := box(5, 1+i-offset, 20, 1, paint(40, 40, uint8(40+i*6), 255))
		row.Clip = view.Padding
		view.Children = append(view.Children, row)
	}
	return page(view)
}

func TestScrollIsALayerScroll(t *testing.T) {
	view := image.Rect(50, 20, 250, 140)
	d := damage(scroller(3), scroller(4))
	if len(d.Scrolls) != 1 || d.Scrolls[0].By != image.Pt(0, -20) || len(d.Moves) != 0 {
		t.Fatalf("scrolling one row: %+v, want one scroll by 0,-20 and no move", d)
	}
	strip := image.Rect(250-cell.X, view.Min.Y, 250, view.Max.Y)
	for _, r := range d.Rects {
		if !r.In(strip) {
			t.Errorf("scrolling one row damaged %v, want only the scrollbar strip %v", r, strip)
		}
	}
	if len(d.Rects) == 0 {
		t.Errorf("scrolling one row damaged nothing, want the thumb")
	}
	f := record(scroller(4))
	l := f.Layers[d.Scrolls[0].Layer]
	if l.Clip != view || l.Visual != view || len(l.Boxes) != 30 {
		t.Errorf("scroll layer: clip %v, visual %v, %d boxes, want %v, %v and all 30 rows kept", l.Clip, l.Visual, len(l.Boxes), view, view)
	}
	if bg := f.Layers[0].Boxes[1].Visual; bg != view {
		t.Errorf("the scroller's own box is at %v in the page layer, want %v, not scrolled", bg, view)
	}
}

func TestScrollThumb(t *testing.T) {
	n := Node{Scroll: true, Padding: layout.Rect{Y: 2, H: 20}}
	for _, c := range []struct {
		offset, content, from, to int
		ok                        bool
	}{
		{0, 200, 0, 16, true},
		{180, 200, 144, 160, true},
		{90, 200, 72, 88, true},
		{0, 20, 0, 0, false},
		{0, 4000, 0, 8, true},
	} {
		n.ScrollContent = layout.Rect{Y: 2 - c.offset, H: c.content}
		from, to, ok := n.Thumb(8)
		if from != c.from || to != c.to || ok != c.ok {
			t.Errorf("offset %d of %d: thumb %d to %d (%v), want %d to %d (%v)", c.offset, c.content, from, to, ok, c.from, c.to, c.ok)
		}
	}
}
