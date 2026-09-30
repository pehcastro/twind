package layout

import "testing"

func TestMemoKeepsEachWidth(t *testing.T) {
	square := box(Style{Aspect: Ratio{2, 1}, Height: pct(100)})
	paragraph := wrap(30)
	paragraph.Style.Grow = 1
	row := box(Style{Height: cells(4), AlignItems: AlignStart}, square, paragraph)
	root := box(Style{Direction: Column}, row)
	Layout(root, 20, Length{})
	borders(t, []*Box{row, square, paragraph}, Rect{0, 0, 20, 4}, Rect{0, 0, 8, 4}, Rect{8, 0, 12, 3})
}

func TestMemoForgetsAnInvalidatedBox(t *testing.T) {
	label := box(Style{}, text("abc"))
	root := box(Style{Direction: Column, AlignItems: AlignStart}, label)
	Layout(root, 20, Length{})
	label.Style.Padding.Left = 2
	label.Invalidate()
	Layout(root, 20, Length{})
	borders(t, []*Box{label}, Rect{0, 0, 5, 1})
}

func TestMemoKeepsEachAvailableWidth(t *testing.T) {
	paragraph := wrap(30)
	popover := box(Style{Direction: Column, AlignItems: AlignStart, Position: PositionAbsolute, MaxWidth: cells(10)}, paragraph)
	root := box(Style{Width: cells(40), Height: cells(10)}, popover)
	Layout(root, 40, Length{})
	borders(t, []*Box{popover, paragraph}, Rect{0, 0, 10, 3}, Rect{0, 0, 10, 3})
}
