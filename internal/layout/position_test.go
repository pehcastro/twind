package layout

import "testing"

func absolute(in Insets, style Style, children ...*Box) *Box {
	style.Position, style.Inset = PositionAbsolute, in
	return box(style, children...)
}

func TestPositionAbsoluteTopLeft(t *testing.T) {
	floating := absolute(Insets{Top: cells(2), Left: cells(3)}, Style{Width: cells(4), Height: cells(1)})
	sibling := text("ab")
	root := box(Style{Direction: Column, Height: cells(10)}, floating, sibling)
	Layout(root, 20, Length{})
	borders(t, []*Box{floating, sibling}, Rect{3, 2, 4, 1}, Rect{0, 0, 20, 1})
}

func TestPositionAbsoluteBottomRight(t *testing.T) {
	sized := absolute(Insets{Right: cells(1), Bottom: cells(2)}, Style{Width: cells(4), Height: cells(3)})
	fitted := &Box{Style: Style{Position: PositionAbsolute, Inset: Insets{Right: cells(0), Bottom: cells(0)}}, Measure: text("abc").Measure}
	root := box(Style{Direction: Column, Height: cells(10)}, sized, fitted)
	Layout(root, 20, Length{})
	borders(t, root.Children, Rect{15, 5, 4, 3}, Rect{17, 9, 3, 1})
}

func TestPositionAbsoluteStretch(t *testing.T) {
	bar := absolute(Insets{Top: cells(1), Left: cells(2), Right: cells(3)}, Style{}, text("a"))
	root := box(Style{Direction: Column, Height: cells(10)}, bar)
	Layout(root, 20, Length{})
	borders(t, []*Box{bar, bar.Children[0]}, Rect{2, 1, 15, 1}, Rect{2, 1, 1, 1})
}

func TestPositionRelativeContainer(t *testing.T) {
	topLeft := absolute(Insets{Top: cells(0), Left: cells(0)}, Style{Width: cells(3), Height: cells(1)})
	bottomRight := absolute(Insets{Right: cells(0), Bottom: cells(0)}, Style{Width: cells(2), Height: cells(1)})
	parent := box(Style{Position: PositionRelative, Height: cells(5), Padding: Edges{1, 1, 1, 1}}, topLeft, bottomRight)
	loose := absolute(Insets{Top: cells(0), Left: cells(0)}, Style{Width: cells(1), Height: cells(1)})
	root := box(Style{Direction: Column, Padding: Edges{2, 2, 2, 2}}, text("a"), parent, loose)
	Layout(root, 30, Length{})
	borders(t, []*Box{root, parent, topLeft, bottomRight, loose},
		Rect{0, 0, 30, 10}, Rect{2, 3, 26, 5}, Rect{2, 3, 3, 1}, Rect{26, 7, 2, 1}, Rect{0, 0, 1, 1})
}

func TestPositionFixed(t *testing.T) {
	pinned := box(Style{Position: PositionFixed, Inset: Insets{Right: cells(0), Bottom: cells(0)}, Width: cells(2), Height: cells(1)})
	parent := box(Style{Position: PositionRelative, Inset: Insets{Top: cells(3), Left: cells(4)}, Width: cells(10), Height: cells(5), Padding: Edges{1, 1, 1, 1}}, pinned)
	root := box(Style{Direction: Column, Height: cells(10)}, parent)
	Layout(root, 20, Length{})
	borders(t, []*Box{parent, pinned}, Rect{4, 3, 10, 5}, Rect{18, 9, 2, 1})
}

func TestPositionRelativeOffset(t *testing.T) {
	first := box(Style{Position: PositionRelative, Inset: Insets{Top: cells(1), Left: cells(2)}}, text("a"))
	second := text("b")
	third := &Box{Style: Style{Position: PositionRelative, Inset: Insets{Right: cells(3), Bottom: cells(1)}}, Measure: text("c").Measure}
	root := box(Style{Direction: Column}, first, second, third)
	Layout(root, 20, Length{})
	borders(t, []*Box{root, first, first.Children[0], second, third},
		Rect{0, 0, 20, 3}, Rect{2, 1, 20, 1}, Rect{2, 1, 1, 1}, Rect{0, 1, 20, 1}, Rect{-3, 1, 20, 1})
}

func TestPositionAutoHeight(t *testing.T) {
	tall := absolute(Insets{Top: cells(0), Left: cells(0)}, Style{Width: cells(30), Height: cells(10)})
	parent := box(Style{}, text("a"), tall)
	root := box(Style{Direction: Column}, parent)
	Layout(root, 20, Length{})
	borders(t, []*Box{root, parent, tall}, Rect{0, 0, 20, 1}, Rect{0, 0, 20, 1}, Rect{0, 0, 30, 10})

	label := &Box{Style: Style{Position: PositionAbsolute}, Measure: text("abcdef").Measure}
	holder := box(Style{}, label)
	row := box(Style{}, holder)
	Layout(row, 20, Length{})
	borders(t, []*Box{row, holder, label}, Rect{0, 0, 20, 0}, Rect{0, 0, 0, 0}, Rect{0, 0, 6, 1})
}

func TestPositionPopover(t *testing.T) {
	menu := absolute(Insets{Bottom: pct(100), Left: cells(0)}, Style{Direction: Column, Width: cells(30)}, text("one"), text("two"), text("three"))
	bar := box(Style{Position: PositionRelative, Height: cells(3), Padding: Edges{1, 1, 1, 1}}, text("> "), menu)
	root := box(Style{Direction: Column, Width: cells(100), Height: cells(30)}, box(Style{Grow: 1}), bar)
	Layout(root, 100, Length{Unit: Cells, Value: 30})
	t.Logf("bar %+v menu %+v", bar.BorderBox, menu.BorderBox)
	if menu.BorderBox.Y+menu.BorderBox.H != bar.BorderBox.Y || menu.BorderBox.X != bar.BorderBox.X {
		t.Errorf("menu %+v not directly above bar %+v", menu.BorderBox, bar.BorderBox)
	}
	borders(t, []*Box{bar, menu}, Rect{0, 27, 100, 3}, Rect{0, 24, 30, 3})
}

func TestPositionClip(t *testing.T) {
	escaped := absolute(Insets{Top: cells(0), Left: cells(0)}, Style{Width: cells(1), Height: cells(1)})
	pinned := box(Style{Position: PositionFixed, Width: cells(1), Height: cells(1)})
	grandchild := box(Style{Height: cells(1)}, escaped, pinned)
	child := box(Style{Overflow: OverflowHidden, Width: cells(20), Height: cells(6), Margin: Edges{Left: 3}}, grandchild)
	hidden := box(Style{Overflow: OverflowHidden, Width: cells(10), Height: cells(3)}, child)
	root := box(Style{Direction: Column, Height: cells(20), Padding: Edges{Top: 2, Left: 5}}, hidden)
	Layout(root, 40, Length{})
	t.Logf("child rect %+v clip %+v", child.BorderBox, child.Clip)
	borders(t, []*Box{hidden, child}, Rect{5, 2, 10, 3}, Rect{8, 2, 20, 6})
	viewport := Rect{0, 0, 40, 20}
	for name, c := range map[string]struct{ got, want Rect }{
		"root":       {root.Clip, viewport},
		"hidden":     {hidden.Clip, viewport},
		"child":      {child.Clip, Rect{5, 2, 10, 3}},
		"grandchild": {grandchild.Clip, Rect{8, 2, 7, 3}},
		"escaped":    {escaped.Clip, viewport},
		"pinned":     {pinned.Clip, viewport},
	} {
		if c.got != c.want {
			t.Errorf("%s clip %+v, want %+v", name, c.got, c.want)
		}
	}
}

func TestShrinkToFitAnchoredMenu(t *testing.T) {
	for _, position := range []Position{PositionAbsolute, PositionFixed} {
		label := text("abcdefghijklmnopqrst")
		menu := box(Style{Position: position, Direction: Column, Inset: Insets{Top: pct(100), Right: cells(0)}}, label)
		anchor := box(Style{Position: PositionRelative, Width: cells(6), Height: cells(1)}, menu)
		root := box(Style{Width: cells(40), Height: cells(10)}, anchor)
		Layout(root, 6, Length{Unit: Cells, Value: 10})
		if menu.BorderBox.W != 20 || label.BorderBox.W != 20 || menu.BorderBox.X != -14 {
			t.Errorf("position %d: menu %+v label %+v, want 20 wide ending at 6", position, menu.BorderBox, label.BorderBox)
		}
	}
}

func TestShrinkToFitWrapsAtAvailable(t *testing.T) {
	for _, position := range []Position{PositionAbsolute, PositionFixed} {
		paragraph := wrap(30)
		popover := box(Style{Position: position, Direction: Column, AlignItems: AlignStart}, paragraph)
		root := box(Style{Position: PositionRelative, Width: cells(12), Height: cells(10)}, popover)
		Layout(root, 12, Length{Unit: Cells, Value: 10})
		borders(t, []*Box{popover, paragraph}, Rect{0, 0, 12, 3}, Rect{0, 0, 12, 3})
	}
}

func TestShrinkToFitMinContentFloor(t *testing.T) {
	for _, position := range []Position{PositionAbsolute, PositionFixed} {
		paragraph, word := wrap(30), text("abcdefghij")
		popover := box(Style{Position: position, Direction: Column, AlignItems: AlignStart, Padding: Edges{Left: 1, Right: 1}}, paragraph, word)
		root := box(Style{Position: PositionRelative, Width: cells(4), Height: cells(10)}, popover)
		Layout(root, 4, Length{Unit: Cells, Value: 10})
		borders(t, []*Box{popover, paragraph, word}, Rect{0, 0, 12, 4}, Rect{1, 0, 10, 3}, Rect{1, 3, 10, 1})
	}
}

func TestShrinkToFitMemoAtMinAndAvailable(t *testing.T) {
	paragraph, word := wrap(30), text("abcdefgh")
	line := box(Style{AlignItems: AlignStart}, word, paragraph)
	popover := box(Style{Position: PositionAbsolute, Direction: Column}, line)
	root := box(Style{Position: PositionRelative, Width: cells(20), Height: cells(10)}, popover)
	Layout(root, 20, Length{Unit: Cells, Value: 10})
	borders(t, []*Box{popover, line, word, paragraph}, Rect{0, 0, 20, 3}, Rect{0, 0, 20, 3}, Rect{0, 0, 8, 1}, Rect{8, 0, 12, 3})
}
