package layout

import (
	"strconv"
	"testing"
)

func stuck(scrollY int, children ...*Box) *Box {
	view := box(Style{Direction: Column, Overflow: OverflowScroll, Height: cells(10)}, children...)
	view.ScrollY = scrollY
	Layout(box(Style{Direction: Column, Height: cells(20)}, view), 20, Length{})
	return view
}

func numbered(n int) []*Box {
	out := make([]*Box, n)
	for i := range out {
		out[i] = text("row " + strconv.Itoa(i))
	}
	return out
}

func TestStickyTopStaysOnTheView(t *testing.T) {
	for _, c := range []struct {
		scroll, want int
	}{{0, 1}, {5, 0}, {12, 0}} {
		header := box(Style{Position: PositionSticky, Inset: Insets{Top: cells(0)}, Height: cells(1)})
		view := stuck(c.scroll, append([]*Box{text("above"), header}, numbered(20)...)...)
		if header.BorderBox.Y != c.want {
			t.Errorf("scrolled %d: header on row %d, want %d", c.scroll, header.BorderBox.Y, c.want)
		}
		if view.ScrollHeight != 22 {
			t.Errorf("scrolled %d: scroll height %d, want 22 as if the header did not stick", c.scroll, view.ScrollHeight)
		}
	}
}

func TestStickyLeavesWithItsSection(t *testing.T) {
	for _, c := range []struct {
		scroll, want int
	}{{3, 0}, {5, 0}, {7, -2}} {
		header := box(Style{Position: PositionSticky, Inset: Insets{Top: cells(0)}, Height: cells(1)})
		section := box(Style{Direction: Column}, append([]*Box{header}, numbered(5)...)...)
		stuck(c.scroll, append([]*Box{section}, numbered(20)...)...)
		if header.BorderBox.Y != c.want {
			t.Errorf("scrolled %d: header on row %d, want %d, bounded by its six-row section", c.scroll, header.BorderBox.Y, c.want)
		}
	}
}

func TestStickyBottomStaysOnTheView(t *testing.T) {
	for _, c := range []struct {
		scroll, want int
	}{{0, 9}, {10, 5}} {
		footer := box(Style{Position: PositionSticky, Inset: Insets{Bottom: cells(0)}, Height: cells(1)})
		rows := numbered(20)
		stuck(c.scroll, append(append(rows[:15:15], footer), rows[15:]...)...)
		if footer.BorderBox.Y != c.want {
			t.Errorf("scrolled %d: footer on row %d, want %d", c.scroll, footer.BorderBox.Y, c.want)
		}
	}
}
