package layout

import "testing"

func TestFlexShrinkZeroOverflowHidden(t *testing.T) {
	wide := text("abcdefghijkl")
	kept := box(Style{Overflow: OverflowHidden, Padding: Edges{Left: 1, Right: 1}}, wide)
	squeezed := box(Style{Shrink: 1, Overflow: OverflowHidden}, text("xyz"))
	root := box(Style{}, kept, squeezed)
	Layout(root, 10, Length{})
	borders(t, []*Box{kept, wide, squeezed}, Rect{0, 0, 14, 1}, Rect{1, 0, 12, 1}, Rect{14, 0, 0, 1})
}

func TestFlexShrinkZeroKeepsGap(t *testing.T) {
	badge := func(s string) *Box {
		return box(Style{Overflow: OverflowHidden, Padding: Edges{Left: 1, Right: 1}, ColumnGap: 1}, &Box{Style: Style{Shrink: 1}, Measure: wrap(len(s)).Measure})
	}
	badges := box(Style{ColumnGap: 2, AlignItems: AlignCenter, Shrink: 1}, badge("Default"), badge("Secondary"), badge("Destructive"))
	column := box(Style{Direction: Column, Width: cells(15), Shrink: 1}, badges)
	root := box(Style{}, column, box(Style{Width: cells(50), Shrink: 1}))
	Layout(root, 40, Length{})
	borders(t, append(root.Children, badges.Children...), Rect{0, 0, 15, 1}, Rect{15, 0, 25, 1}, Rect{0, 0, 9, 1}, Rect{11, 0, 11, 1}, Rect{24, 0, 13, 1})
	borders(t, []*Box{badges.Children[1].Children[0]}, Rect{12, 0, 9, 1})

	column.Style.Width = Length{}
	Layout(root, 40, Length{})
	borders(t, root.Children, Rect{0, 0, 37, 1}, Rect{37, 0, 3, 1})
}

func TestFlexShrinkAutoMinimum(t *testing.T) {
	shrink := func(s string, style Style) *Box {
		style.Shrink = 1
		return &Box{Style: style, Measure: text(s).Measure}
	}
	root := box(Style{}, shrink("abcd", Style{}), shrink("efgh", Style{}))
	Layout(root, 6, Length{})
	borders(t, root.Children, Rect{0, 0, 4, 1}, Rect{4, 0, 4, 1})

	root = box(Style{}, shrink("abcd", Style{MinWidth: cells(0)}), shrink("efgh", Style{}))
	Layout(root, 6, Length{})
	borders(t, root.Children, Rect{0, 0, 2, 1}, Rect{2, 0, 4, 1})

	root = box(Style{}, shrink("abcd", Style{Overflow: OverflowHidden}), box(Style{Shrink: 1, Width: pct(50)}, text("efgh")))
	Layout(root, 6, Length{})
	borders(t, root.Children, Rect{0, 0, 3, 1}, Rect{3, 0, 3, 1})

	flex1 := Style{Basis: cells(0), Grow: 1, Shrink: 1}
	root = box(Style{}, &Box{Style: flex1, Measure: text("a").Measure}, &Box{Style: flex1, Measure: text("abcdef").Measure})
	Layout(root, 10, Length{})
	borders(t, root.Children, Rect{0, 0, 4, 1}, Rect{4, 0, 6, 1})
}

func TestAspectRatio(t *testing.T) {
	video := box(Style{Aspect: Ratio{32, 9}})
	column := box(Style{Direction: Column}, video, box(Style{Width: cells(10), Aspect: Ratio{16, 9}}))
	Layout(column, 32, Length{})
	borders(t, append([]*Box{column}, column.Children...), Rect{0, 0, 32, 15}, Rect{0, 0, 32, 9}, Rect{0, 9, 10, 6})

	square := box(Style{Height: cells(4), Aspect: Ratio{2, 1}})
	both := box(Style{Width: cells(5), Height: cells(5), Aspect: Ratio{2, 1}})
	row := box(Style{AlignItems: AlignStart}, square, both)
	Layout(row, 30, Length{})
	borders(t, row.Children, Rect{0, 0, 8, 4}, Rect{8, 0, 5, 5})

	floating := box(Style{Position: PositionAbsolute, Width: cells(6), Aspect: Ratio{3, 1}})
	root := box(Style{Height: cells(10)}, floating)
	Layout(root, 20, Length{})
	borders(t, []*Box{floating}, Rect{0, 0, 6, 2})
}

func TestFlexWrap(t *testing.T) {
	item := func(w int, style Style) *Box {
		style.Width, style.Height = cells(w), cells(1)
		return box(style)
	}
	root := box(Style{Wrap: Wrap, ColumnGap: 1, RowGap: 1}, item(4, Style{}), item(4, Style{}), item(4, Style{}))
	Layout(root, 10, Length{})
	borders(t, append([]*Box{root}, root.Children...), Rect{0, 0, 10, 3}, Rect{0, 0, 4, 1}, Rect{5, 0, 4, 1}, Rect{0, 2, 4, 1})

	root.Style.Wrap = WrapReverse
	Layout(root, 10, Length{})
	borders(t, root.Children, Rect{0, 2, 4, 1}, Rect{5, 2, 4, 1}, Rect{0, 0, 4, 1})

	root = box(Style{Wrap: Wrap, Justify: JustifyEnd}, item(12, Style{}), item(3, Style{}), item(3, Style{}))
	Layout(root, 10, Length{})
	borders(t, append([]*Box{root}, root.Children...), Rect{0, 0, 10, 2}, Rect{-2, 0, 12, 1}, Rect{4, 1, 3, 1}, Rect{7, 1, 3, 1})

	root = box(Style{Wrap: Wrap}, item(4, Style{}), item(4, Style{Grow: 1}), item(4, Style{}))
	Layout(root, 10, Length{})
	borders(t, root.Children, Rect{0, 0, 4, 1}, Rect{4, 0, 6, 1}, Rect{0, 1, 4, 1})

	tall := box(Style{Width: cells(3), Height: cells(2)})
	root = box(Style{Wrap: Wrap}, tall, box(Style{Width: cells(3)}), item(8, Style{}))
	Layout(root, 10, Length{})
	borders(t, append([]*Box{root}, root.Children...), Rect{0, 0, 10, 3}, Rect{0, 0, 3, 2}, Rect{3, 0, 3, 2}, Rect{0, 2, 8, 1})
}
