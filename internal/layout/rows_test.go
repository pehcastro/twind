package layout

import "testing"

func halfText() *Box {
	return &Box{Style: Style{RowUnits: 2}, Measure: func(int) (int, int) { return 4, 2 }}
}

func halves(s Style, children ...*Box) *Box {
	s.RowUnits = 2
	return box(s, children...)
}

func TestHalfRowsNudgeContentOntoWholeRows(t *testing.T) {
	edge := Edges{Top: 1, Bottom: 1}
	for _, tc := range []struct {
		gap      int
		boxAt    int
		afterAt  int
		describe string
	}{
		{1, 3, 8, "a half gap puts the bordered box half a row in, as planned"},
		{2, 5, 12, "a whole gap would put its text on a half row: the box moves down a half"},
	} {
		field := halves(Style{Direction: Column, Border: edge}, halfText())
		after := halfText()
		root := halves(Style{Direction: Column, RowGap: tc.gap, Width: cells(20)}, halfText(), field, after)
		Layout(root, 20, Length{})
		if field.BorderBox.Y != tc.boxAt || after.BorderBox.Y != tc.afterAt {
			t.Errorf("gap %d: box at %d, text after it at %d, want %d and %d: %s", tc.gap, field.BorderBox.Y, after.BorderBox.Y, tc.boxAt, tc.afterAt, tc.describe)
		}
	}
}

func TestHalfRowsLeaveEmptyBoxesWhereTheyAre(t *testing.T) {
	line := halves(Style{Height: cells(1)})
	root := halves(Style{Direction: Column, Width: cells(20)}, halves(Style{Height: cells(1)}), line)
	Layout(root, 20, Length{})
	if line.BorderBox.Y != 1 {
		t.Errorf("an empty one-unit box after another at %d, want 1", line.BorderBox.Y)
	}
}

func TestHalfRowsRowMeasuresTheNudge(t *testing.T) {
	field := halves(Style{Direction: Column, Border: Edges{Top: 1, Bottom: 1}}, halfText())
	row := halves(Style{AlignItems: AlignStart}, halfText(), field)
	next := halfText()
	root := halves(Style{Direction: Column, Width: cells(20)}, row, next)
	Layout(root, 20, Length{})
	if field.BorderBox.Y != 1 || row.BorderBox.H != 5 || next.BorderBox.Y != 6 {
		t.Errorf("bordered box at %d in a row %d units tall, text below at %d, want 1, 5 and 6", field.BorderBox.Y, row.BorderBox.H, next.BorderBox.Y)
	}
}

func TestHalfRowsAspectIsWholeRows(t *testing.T) {
	wide := halves(Style{Width: cells(3), Aspect: Ratio{W: 2, H: 1}})
	tall := halves(Style{Height: cells(6), Aspect: Ratio{W: 2, H: 1}})
	root := halves(Style{Direction: Column, Width: cells(20), AlignItems: AlignStart}, wide, tall)
	Layout(root, 20, Length{})
	if wide.BorderBox.H != 4 || tall.BorderBox.W != 6 {
		t.Errorf("a 2:1 ratio in cell units: 3 columns wide is %d units tall, 6 units (3 rows) tall is %d columns wide, want 4 (the 2 rows it takes without halves) and 6", wide.BorderBox.H, tall.BorderBox.W)
	}
}

func TestHalfRowsJustifySpacesInWholeRows(t *testing.T) {
	texts := []*Box{halfText(), halfText(), halfText()}
	root := halves(Style{Direction: Column, Justify: JustifyBetween, Width: cells(20), Height: cells(11)}, texts...)
	Layout(root, 20, Length{})
	if got := [3]int{texts[0].BorderBox.Y, texts[1].BorderBox.Y, texts[2].BorderBox.Y}; got != [3]int{0, 4, 8} {
		t.Errorf("three texts spread over 11 units at %v, want [0 4 8]: 5 free units are 2 whole rows, one per gap", got)
	}
}

func TestHalfRowsPlacedBoxes(t *testing.T) {
	placed := halves(Style{Position: PositionAbsolute, Inset: Insets{Top: cells(1), Left: cells(0)}}, halfText())
	root := halves(Style{Direction: Column, Position: PositionRelative, Width: cells(20), Height: cells(10)}, placed)
	Layout(root, 20, Length{})
	if placed.BorderBox.Y != 2 {
		t.Errorf("an absolute box with text at top 1 lands at %d, want 2", placed.BorderBox.Y)
	}
}

func TestHalfRowsContentStartsOnWholeRows(t *testing.T) {
	edge := Edges{Top: 1, Bottom: 1}
	scroller := halves(Style{Direction: Column, Height: cells(6), Overflow: OverflowScroll, RowGap: 1},
		halfText(), halves(Style{Direction: Column, Border: edge}, halfText()), halfText(), halfText(), halfText(), halfText())
	scroller.ScrollY = 2
	grid := halves(Style{Display: DisplayGrid, Columns: []Track{{Min: Breadth{Kind: SizeFr, Value: 100}, Max: Breadth{Kind: SizeFr, Value: 100}}}, AlignItems: AlignCenter, RowGap: 1},
		halfText(), halves(Style{Direction: Column, Padding: Edges{Top: 1}}, halfText()), halves(Style{Height: cells(5)}, halfText()))
	root := halves(Style{Direction: Column, Width: cells(40), Height: cells(60), Position: PositionRelative},
		halves(Style{Direction: Column, Padding: Edges{Top: 1, Bottom: 1}}, halfText()),
		halves(Style{AlignItems: AlignCenter, Height: cells(5)}, halfText(), halves(Style{Direction: Column, Border: edge}, halfText()), halves(Style{Height: cells(3)}, halfText())),
		halves(Style{Direction: Column, Justify: JustifyBetween, Height: cells(11)}, halfText(), halfText(), halfText()),
		halves(Style{Direction: Column, Grow: 1, Height: cells(3)}, halfText()),
		halves(Style{Direction: Column, Grow: 1, Height: cells(3)}, halfText()),
		scroller,
		grid,
		halves(Style{Direction: Column, Height: cells(1)}),
		halves(Style{Position: PositionAbsolute, Inset: Insets{Top: cells(7), Left: cells(3)}, Padding: Edges{Top: 1}}, halfText()),
	)
	Layout(root, 40, Length{Unit: Cells, Value: 60})
	var walk func(b *Box, path []int)
	walk = func(b *Box, path []int) {
		if (len(b.Children) > 0 || b.Measure != nil) && b.ContentBox.Y%2 != 0 {
			t.Errorf("box %v holds something and its content starts at %d, a half row", path, b.ContentBox.Y)
		}
		for i, c := range b.Children {
			walk(c, append(path, i))
		}
	}
	walk(root, nil)
}
