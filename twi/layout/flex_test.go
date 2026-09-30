package layout

import (
	"testing"
	"unicode/utf8"
)

func cells(n int) Length { return Length{Unit: Cells, Value: n} }

func pct(n int) Length { return Length{Unit: Percent, Value: n} }

func text(s string) *Box {
	return &Box{Measure: func(int) (int, int) { return utf8.RuneCountInString(s), 1 }}
}

func wrap(n int) *Box {
	return &Box{Measure: func(avail int) (int, int) {
		avail = max(avail, 1)
		return min(n, avail), (n + avail - 1) / avail
	}}
}

func box(style Style, children ...*Box) *Box {
	return &Box{Style: style, Children: children}
}

func borders(t *testing.T, boxes []*Box, want ...Rect) {
	t.Helper()
	for i, b := range boxes {
		if b.BorderBox != want[i] {
			t.Errorf("box %d: border box %+v, want %+v", i, b.BorderBox, want[i])
		}
	}
}

func TestFlexSidebar(t *testing.T) {
	sidebar := box(Style{Width: cells(20)})
	content := box(Style{Grow: 1})
	root := box(Style{Width: cells(100)}, sidebar, content)
	Layout(root, 100, Length{})
	borders(t, []*Box{root, sidebar, content}, Rect{0, 0, 100, 0}, Rect{0, 0, 20, 0}, Rect{20, 0, 80, 0})
}

func TestFlexGrowLeftover(t *testing.T) {
	root := box(Style{}, box(Style{Grow: 1}), box(Style{Grow: 1}), box(Style{Grow: 1}))
	Layout(root, 10, Length{})
	borders(t, root.Children, Rect{0, 0, 4, 0}, Rect{4, 0, 3, 0}, Rect{7, 0, 3, 0})

	for width := range 41 {
		for n := 1; n <= 5; n++ {
			root := box(Style{})
			for i := range n {
				root.Children = append(root.Children, box(Style{Grow: 1 + i%3}))
			}
			Layout(root, width, Length{})
			sum := 0
			for _, c := range root.Children {
				sum += c.BorderBox.W
			}
			if sum != width {
				t.Errorf("width %d, %d children: sum %d", width, n, sum)
			}
		}
	}
}

func TestFlexGrow(t *testing.T) {
	root := box(Style{}, box(Style{Width: cells(2)}), box(Style{Grow: 1}), box(Style{Grow: 2}))
	Layout(root, 11, Length{})
	borders(t, root.Children, Rect{0, 0, 2, 0}, Rect{2, 0, 3, 0}, Rect{5, 0, 6, 0})

	column := box(Style{Direction: Column}, box(Style{Grow: 1}), text("a"))
	Layout(column, 5, Length{})
	borders(t, append([]*Box{column}, column.Children...), Rect{0, 0, 5, 1}, Rect{0, 0, 5, 0}, Rect{0, 0, 5, 1})

	column = box(Style{Direction: Column, Height: cells(10)}, text("a"), box(Style{Grow: 1}))
	Layout(column, 5, Length{})
	borders(t, column.Children, Rect{0, 0, 5, 1}, Rect{0, 1, 5, 9})
}

func TestFlexShrink(t *testing.T) {
	root := box(Style{}, box(Style{Width: cells(10), Shrink: 1}), box(Style{Width: cells(5), Shrink: 1}))
	Layout(root, 10, Length{})
	borders(t, root.Children, Rect{0, 0, 7, 0}, Rect{7, 0, 3, 0})

	root = box(Style{}, box(Style{Width: cells(6)}), box(Style{Width: cells(6), Shrink: 1}))
	Layout(root, 10, Length{})
	borders(t, root.Children, Rect{0, 0, 6, 0}, Rect{6, 0, 4, 0})

	root = box(Style{}, box(Style{Width: cells(10), Shrink: 1, MinWidth: cells(8)}), box(Style{Width: cells(10), Shrink: 1}))
	Layout(root, 10, Length{})
	borders(t, root.Children, Rect{0, 0, 8, 0}, Rect{8, 0, 2, 0})

	root = box(Style{}, box(Style{Width: cells(10)}), box(Style{Width: cells(2), Shrink: 1}))
	Layout(root, 5, Length{})
	borders(t, root.Children, Rect{0, 0, 10, 0}, Rect{10, 0, 0, 0})

	root = box(Style{}, box(Style{Width: cells(3), Shrink: 1}))
	Layout(root, 10, Length{})
	borders(t, root.Children, Rect{0, 0, 3, 0})
}

func TestFlexColumnAutoMin(t *testing.T) {
	shrink := func(s string, style Style) *Box {
		style.Shrink = 1
		return &Box{Style: style, Measure: text(s).Measure}
	}
	rows := []*Box{shrink("title", Style{}), shrink("row 1", Style{}), shrink("row 2", Style{}), shrink("row 3", Style{}), shrink("row 4", Style{})}
	column := box(Style{Direction: Column, Height: cells(4), Overflow: OverflowHidden}, rows...)
	Layout(column, 10, Length{})
	borders(t, rows, Rect{0, 0, 10, 1}, Rect{0, 1, 10, 1}, Rect{0, 2, 10, 1}, Rect{0, 3, 10, 1}, Rect{0, 4, 10, 1})

	column.Style.Border, column.Style.Padding = Edges{1, 1, 1, 1}, Edges{Left: 1, Right: 1}
	Layout(column, 10, Length{})
	borders(t, rows, Rect{2, 1, 6, 1}, Rect{2, 2, 6, 1}, Rect{2, 3, 6, 1}, Rect{2, 4, 6, 1}, Rect{2, 5, 6, 1})

	hidden := box(Style{Direction: Column, Shrink: 1, Overflow: OverflowHidden}, text("a"), text("b"), text("c"))
	column = box(Style{Direction: Column, Height: cells(3)}, shrink("x", Style{}), hidden)
	Layout(column, 5, Length{})
	borders(t, column.Children, Rect{0, 0, 5, 1}, Rect{0, 1, 5, 2})

	column = box(Style{Direction: Column, Height: cells(1)}, shrink("b", Style{}), shrink("a", Style{MinHeight: cells(0)}))
	Layout(column, 5, Length{})
	borders(t, column.Children, Rect{0, 0, 5, 1}, Rect{0, 1, 5, 0})

	sized := box(Style{Direction: Column, Height: cells(1), Shrink: 1}, text("a"), text("b"))
	column = box(Style{Direction: Column, Height: cells(1)}, sized, shrink("z", Style{}))
	Layout(column, 5, Length{})
	borders(t, column.Children, Rect{0, 0, 5, 1}, Rect{0, 1, 5, 1})

	row := box(Style{Width: cells(4)}, shrink("abcd", Style{}), shrink("efgh", Style{}))
	Layout(row, 4, Length{})
	borders(t, row.Children, Rect{0, 0, 2, 1}, Rect{2, 0, 2, 1})
}

func TestFlexBasis(t *testing.T) {
	root := box(Style{},
		box(Style{Basis: cells(6), Width: cells(2)}),
		box(Style{Width: cells(5)}),
		text("abc"),
		box(Style{Basis: pct(25)}),
	)
	Layout(root, 20, Length{})
	borders(t, root.Children, Rect{0, 0, 6, 1}, Rect{6, 0, 5, 1}, Rect{11, 0, 3, 1}, Rect{14, 0, 5, 1})

	flex1 := Style{Basis: cells(0), Grow: 1, Shrink: 1}
	root = box(Style{}, &Box{Style: flex1, Measure: text("a").Measure}, &Box{Style: flex1, Measure: text("abcdef").Measure})
	Layout(root, 10, Length{})
	borders(t, root.Children, Rect{0, 0, 5, 1}, Rect{5, 0, 5, 1})

	column := box(Style{Direction: Column}, &Box{Style: Style{Basis: pct(50)}, Measure: text("ab").Measure})
	Layout(column, 10, Length{})
	borders(t, column.Children, Rect{0, 0, 10, 1})
}

func TestFlexGap(t *testing.T) {
	root := box(Style{ColumnGap: 2, RowGap: 5}, box(Style{Width: cells(3)}), box(Style{Width: cells(3)}), box(Style{Width: cells(3), Grow: 1}))
	Layout(root, 20, Length{})
	borders(t, root.Children, Rect{0, 0, 3, 0}, Rect{5, 0, 3, 0}, Rect{10, 0, 10, 0})

	hidden := box(Style{Display: DisplayNone, Width: cells(4), Height: cells(4)})
	column := box(Style{Direction: Column, RowGap: 1, ColumnGap: 7}, text("a"), hidden, text("b"), text("c"))
	Layout(column, 10, Length{})
	borders(t, append([]*Box{column}, column.Children...), Rect{0, 0, 10, 5}, Rect{0, 0, 10, 1}, Rect{}, Rect{0, 2, 10, 1}, Rect{0, 4, 10, 1})
}

func TestFlexPadding(t *testing.T) {
	root := box(Style{Direction: Column, Padding: Edges{1, 2, 3, 4}, Border: Edges{1, 1, 1, 1}}, text("ab"))
	Layout(root, 20, Length{})
	if root.BorderBox != (Rect{0, 0, 20, 7}) || root.PaddingBox != (Rect{1, 1, 18, 5}) || root.ContentBox != (Rect{5, 2, 12, 1}) {
		t.Errorf("boxes %+v %+v %+v", root.BorderBox, root.PaddingBox, root.ContentBox)
	}
	borders(t, root.Children, Rect{5, 2, 12, 1})
}

func TestFlexAlign(t *testing.T) {
	root := box(Style{Height: cells(10), AlignItems: AlignCenter},
		text("ab"),
		&Box{Style: Style{AlignSelf: AlignEnd}, Measure: text("ab").Measure},
		box(Style{AlignSelf: AlignStretch, Width: cells(1)}),
		box(Style{AlignSelf: AlignStretch, Width: cells(1), Height: cells(3)}),
		&Box{Style: Style{AlignSelf: AlignStart}, Measure: text("ab").Measure},
		box(Style{Width: cells(1), MaxHeight: pct(50), AlignSelf: AlignStretch}),
	)
	Layout(root, 20, Length{})
	borders(t, root.Children, Rect{0, 4, 2, 1}, Rect{2, 9, 2, 1}, Rect{4, 0, 1, 10}, Rect{5, 0, 1, 3}, Rect{6, 0, 2, 1}, Rect{8, 0, 1, 5})

	auto := box(Style{}, box(Style{Width: cells(4)}, wrap(12)), box(Style{Width: cells(2)}))
	Layout(auto, 20, Length{})
	borders(t, append([]*Box{auto}, auto.Children...), Rect{0, 0, 20, 3}, Rect{0, 0, 4, 3}, Rect{4, 0, 2, 3})

	cases := []struct {
		justify Justify
		width   int
		xs      [2]int
	}{
		{JustifyStart, 10, [2]int{0, 2}},
		{JustifyEnd, 10, [2]int{6, 8}},
		{JustifyCenter, 10, [2]int{3, 5}},
		{JustifyBetween, 10, [2]int{0, 8}},
		{JustifyAround, 10, [2]int{2, 7}},
		{JustifyEvenly, 10, [2]int{2, 6}},
		{JustifyCenter, 11, [2]int{3, 5}},
		{JustifyBetween, 11, [2]int{0, 9}},
		{JustifyAround, 11, [2]int{2, 8}},
		{JustifyEvenly, 11, [2]int{3, 7}},
		{JustifyBetween, 3, [2]int{0, 2}},
		{JustifyAround, 3, [2]int{0, 2}},
		{JustifyEvenly, 3, [2]int{0, 2}},
	}
	for _, c := range cases {
		root := box(Style{Justify: c.justify}, box(Style{Width: cells(2)}), box(Style{Width: cells(2)}))
		Layout(root, c.width, Length{})
		if xs := [2]int{root.Children[0].BorderBox.X, root.Children[1].BorderBox.X}; xs != c.xs {
			t.Errorf("justify %d width %d: x %v, want %v", c.justify, c.width, xs, c.xs)
		}
	}
}

func TestFlexMinMax(t *testing.T) {
	root := box(Style{}, box(Style{Grow: 1, MaxWidth: cells(2)}), box(Style{Grow: 1}), box(Style{Grow: 1}))
	Layout(root, 12, Length{})
	borders(t, root.Children, Rect{0, 0, 2, 0}, Rect{2, 0, 5, 0}, Rect{7, 0, 5, 0})

	root = box(Style{}, box(Style{Grow: 1, MinWidth: cells(8)}), box(Style{Grow: 1}), box(Style{Grow: 1}))
	Layout(root, 12, Length{})
	borders(t, root.Children, Rect{0, 0, 8, 0}, Rect{8, 0, 2, 0}, Rect{10, 0, 2, 0})

	root = box(Style{}, box(Style{MinWidth: cells(6), MaxWidth: cells(4)}), box(Style{Grow: 1, MaxWidth: pct(25)}))
	Layout(root, 20, Length{})
	borders(t, root.Children, Rect{0, 0, 6, 0}, Rect{6, 0, 5, 0})

	column := box(Style{Direction: Column},
		box(Style{MaxWidth: cells(10)}),
		&Box{Style: Style{MinHeight: cells(3)}, Measure: text("ab").Measure},
	)
	Layout(column, 20, Length{})
	borders(t, append([]*Box{column}, column.Children...), Rect{0, 0, 20, 3}, Rect{0, 0, 10, 0}, Rect{0, 0, 20, 3})
}

func TestFlexNested(t *testing.T) {
	inner := box(Style{}, box(Style{Grow: 1}, text("x")), box(Style{Grow: 1}, text("y")))
	para := wrap(30)
	column := box(Style{Direction: Column, Grow: 1, Padding: Edges{1, 1, 1, 1}}, para, inner)
	root := box(Style{Padding: Edges{1, 1, 1, 1}, Border: Edges{1, 1, 1, 1}}, column)
	Layout(root, 30, Length{})
	borders(t, []*Box{root, column, para, inner, inner.Children[0], inner.Children[1], inner.Children[1].Children[0]},
		Rect{0, 0, 30, 9}, Rect{2, 2, 26, 5}, Rect{3, 3, 24, 2}, Rect{3, 5, 24, 1}, Rect{3, 5, 12, 1}, Rect{15, 5, 12, 1}, Rect{15, 5, 1, 1})

	para = wrap(18)
	root = box(Style{}, box(Style{Width: cells(4)}), box(Style{Direction: Column, Grow: 1, Basis: cells(0)}, para))
	Layout(root, 10, Length{})
	borders(t, []*Box{root, root.Children[1], para}, Rect{0, 0, 10, 3}, Rect{4, 0, 6, 3}, Rect{4, 0, 6, 3})
}

func TestFlexHelloTwind(t *testing.T) {
	hello := text("Hello Twind")
	dom := text("Terminal DOM")
	card := box(Style{Direction: Column, Border: Edges{1, 1, 1, 1}, Padding: Edges{2, 2, 2, 2}}, dom)
	root := box(Style{Direction: Column, RowGap: 2, ColumnGap: 2, Padding: Edges{4, 4, 4, 4}}, hello, card)
	Layout(root, 80, Length{})
	for _, b := range []*Box{root, hello, card, dom} {
		t.Logf("border box (%d,%d,%d,%d)", b.BorderBox.X, b.BorderBox.Y, b.BorderBox.W, b.BorderBox.H)
	}
	borders(t, []*Box{root, hello, card, dom}, Rect{0, 0, 80, 18}, Rect{4, 4, 72, 1}, Rect{4, 7, 72, 7}, Rect{7, 10, 66, 1})
}
