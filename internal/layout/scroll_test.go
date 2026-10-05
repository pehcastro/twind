package layout

import (
	"strconv"
	"testing"
)

func scrolled(style Style) (*Box, []*Box) {
	rows := make([]*Box, 40)
	for i := range rows {
		rows[i] = text("row " + strconv.Itoa(i))
		rows[i].Style.Shrink = 1
	}
	style.Direction, style.Overflow = Column, OverflowScroll
	return box(style, rows...), rows
}

func TestScrollChildAtOffset(t *testing.T) {
	view, rows := scrolled(Style{Height: cells(10)})
	root := box(Style{Direction: Column, Height: cells(20)}, view)
	view.ScrollY = 25
	Layout(root, 20, Length{})
	borders(t, []*Box{view, rows[30], rows[25], rows[36]}, Rect{0, 0, 20, 10}, Rect{0, 5, 20, 1}, Rect{0, 0, 20, 1}, Rect{0, 11, 20, 1})
	if rows[30].Clip != view.PaddingBox {
		t.Errorf("row 30 clip %+v, want the view's padding box %+v", rows[30].Clip, view.PaddingBox)
	}
	if seen := intersect(rows[36].BorderBox, rows[36].Clip); seen.W*seen.H != 0 {
		t.Errorf("row 36 scrolled out shows %+v, want nothing", seen)
	}
	if view.ScrollWidth != 20 || view.ScrollHeight != 40 {
		t.Errorf("content size %dx%d, want 20x40", view.ScrollWidth, view.ScrollHeight)
	}
}

func TestScrollClamps(t *testing.T) {
	for _, c := range []struct {
		name          string
		style         Style
		x, y          int
		wantY, height int
	}{
		{"past the end", Style{Height: cells(10)}, 0, 100, 30, 40},
		{"below zero", Style{Height: cells(10)}, -4, -3, 0, 40},
		{"no horizontal overflow", Style{Height: cells(10)}, 5, 2, 2, 40},
		{"end padding and margin", Style{Height: cells(10), Padding: Edges{1, 1, 1, 1}}, 0, 100, 33, 43},
		{"nothing overflows", Style{Height: cells(50)}, 0, 7, 0, 50},
	} {
		view, rows := scrolled(c.style)
		rows[39].Style.Margin.Bottom = c.style.Padding.Bottom
		view.ScrollX, view.ScrollY = c.x, c.y
		Layout(box(Style{Direction: Column, Height: cells(60)}, view), 20, Length{})
		if view.ScrollX != 0 || view.ScrollY != c.wantY || view.ScrollHeight != c.height {
			t.Errorf("%s: offset %d,%d height %d, want 0,%d and %d", c.name, view.ScrollX, view.ScrollY, view.ScrollHeight, c.wantY, c.height)
		}
		if first := rows[0].BorderBox.Y - view.ContentBox.Y; first != -c.wantY {
			t.Errorf("%s: first row at %d from the content box, want %d", c.name, first, -c.wantY)
		}
	}
}

func TestScrollKeptAcrossLayouts(t *testing.T) {
	view, rows := scrolled(Style{Height: cells(10)})
	root := box(Style{Direction: Column, Height: cells(20)}, view)
	view.ScrollY = 12
	Layout(root, 20, Length{})
	Layout(root, 24, Length{})
	if view.ScrollY != 12 || rows[12].BorderBox.Y != 0 {
		t.Errorf("after a second layout: offset %d, row 12 at %d, want 12 and 0", view.ScrollY, rows[12].BorderBox.Y)
	}
}

func TestScrollPositionedChildren(t *testing.T) {
	view, _ := scrolled(Style{Height: cells(10), Position: PositionRelative})
	floating := absolute(Insets{Top: cells(30), Left: cells(2)}, Style{Width: cells(4), Height: cells(1)})
	pinned := box(Style{Position: PositionFixed, Inset: Insets{Top: cells(0), Left: cells(0)}, Width: cells(3), Height: cells(1)})
	view.Children = append(view.Children, floating, pinned)
	view.ScrollY = 25
	Layout(box(Style{Direction: Column, Height: cells(20)}, view), 20, Length{})
	borders(t, []*Box{floating, pinned}, Rect{2, 5, 4, 1}, Rect{0, 0, 3, 1})
	if floating.Clip != view.PaddingBox {
		t.Errorf("absolute child clip %+v, want the view %+v", floating.Clip, view.PaddingBox)
	}
}

func TestScrollContainerShrinks(t *testing.T) {
	view, _ := scrolled(Style{Grow: 1, Shrink: 1})
	root := box(Style{Direction: Column, Height: cells(10)}, text("title"), view)
	Layout(root, 20, cells(10))
	if view.BorderBox != (Rect{0, 1, 20, 9}) || view.ScrollHeight != 40 {
		t.Errorf("scroller as a flex item: %+v with content height %d, want {0 1 20 9} and 40", view.BorderBox, view.ScrollHeight)
	}
}
