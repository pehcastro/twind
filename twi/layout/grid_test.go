package layout

import (
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/layout"
)

var (
	autoTrack  = Track{}
	flexTrack  = Track{Max: Breadth{SizeFr, konst.FrUnit}}
	shareTrack = Track{Min: Breadth{SizeCells, 0}, Max: Breadth{SizeFr, konst.FrUnit}}
	minTrack   = Track{Min: Breadth{Kind: SizeMinContent}, Max: Breadth{Kind: SizeMinContent}}
)

func repeat(n int, t Track) []Track {
	out := make([]Track, n)
	for i := range out {
		out[i] = t
	}
	return out
}

func grid(s Style, children ...*Box) *Box {
	s.Display = DisplayGrid
	return box(s, children...)
}

func texts(words ...string) []*Box {
	out := make([]*Box, len(words))
	for i, w := range words {
		out[i] = text(w)
	}
	return out
}

func TestGridThreeColumnsGap(t *testing.T) {
	items := append([]*Box{wrap(30)}, texts("b", "c", "d", "e", "f", "g")...)
	root := grid(Style{Columns: repeat(3, shareTrack), RowGap: 2, ColumnGap: 2}, items...)
	Layout(root, 40, Length{})
	borders(t, append([]*Box{root}, items...),
		Rect{0, 0, 40, 9},
		Rect{0, 0, 12, 3}, Rect{14, 0, 12, 3}, Rect{28, 0, 12, 3},
		Rect{0, 5, 12, 1}, Rect{14, 5, 12, 1}, Rect{28, 5, 12, 1},
		Rect{0, 8, 12, 1})
}

func TestGridFrAndAuto(t *testing.T) {
	long, short := wrap(50), text("Action")
	root := grid(Style{Columns: []Track{flexTrack, autoTrack}, RowGap: 2, ColumnGap: 2}, long, short)
	Layout(root, 30, Length{})
	borders(t, []*Box{root, long, short}, Rect{0, 0, 30, 3}, Rect{0, 0, 22, 3}, Rect{24, 0, 6, 3})
}

func TestGridSpanSparseAndDense(t *testing.T) {
	span2 := Placement{Start: Line{Span: 2}, End: Line{Span: 2}}
	items := texts("a", "b", "c", "d")
	items[0].Style.Column = span2
	root := grid(Style{Columns: repeat(3, shareTrack), RowGap: 2, ColumnGap: 2}, items...)
	Layout(root, 40, Length{})
	borders(t, items, Rect{0, 0, 26, 1}, Rect{28, 0, 12, 1}, Rect{0, 3, 12, 1}, Rect{14, 3, 12, 1})

	for _, c := range []struct {
		flow  Flow
		third Rect
	}{{FlowRow, Rect{28, 3, 12, 1}}, {FlowRowDense, Rect{28, 0, 12, 1}}} {
		items := texts("a", "b", "c")
		items[0].Style.Column, items[1].Style.Column = span2, span2
		root := grid(Style{Columns: repeat(3, shareTrack), RowGap: 2, ColumnGap: 2, Flow: c.flow}, items...)
		Layout(root, 40, Length{})
		borders(t, items, Rect{0, 0, 26, 1}, Rect{0, 3, 26, 1}, c.third)
	}
}

func TestGridSpanningAutoTracks(t *testing.T) {
	items := texts("abcdefghij", "x", "y")
	items[0].Style.Column = Placement{Start: Line{Span: 2}, End: Line{Span: 2}}
	root := grid(Style{Columns: []Track{autoTrack, autoTrack}, Justify: JustifyStart}, items...)
	Layout(root, 30, Length{})
	borders(t, append([]*Box{root}, items...), Rect{0, 0, 30, 2}, Rect{0, 0, 10, 1}, Rect{0, 1, 5, 1}, Rect{5, 1, 5, 1})
}

func TestGridCardHeaderAction(t *testing.T) {
	title, description := text("Card Title"), wrap(50)
	label := text("Button")
	action := box(Style{
		Column:      Placement{Start: Line{Index: 2}},
		Row:         Placement{Start: Line{Index: 1}, End: Line{Span: 2}},
		AlignSelf:   AlignStart,
		JustifySelf: AlignEnd,
	}, label)
	header := grid(Style{
		Columns:    []Track{flexTrack, autoTrack},
		Rows:       []Track{autoTrack, autoTrack},
		AutoRows:   []Track{minTrack},
		AlignItems: AlignStart,
		RowGap:     2, ColumnGap: 2,
	}, title, description, action)
	Layout(header, 40, Length{})
	borders(t, []*Box{header, title, description, action, label},
		Rect{0, 0, 40, 5}, Rect{0, 0, 32, 1}, Rect{0, 3, 32, 2}, Rect{34, 0, 6, 1}, Rect{34, 0, 6, 1})
}

func TestGridAutoRowsMin(t *testing.T) {
	for _, c := range []struct {
		name     string
		autoRows []Track
		a, b     Rect
	}{
		{"auto-rows-min", []Track{minTrack}, Rect{0, 0, 20, 1}, Rect{0, 1, 20, 2}},
		{"auto rows stretch", nil, Rect{0, 0, 20, 5}, Rect{0, 5, 20, 6}},
	} {
		a, b := text("abc"), wrap(30)
		root := grid(Style{Height: cells(11), AutoRows: c.autoRows}, a, b)
		Layout(root, 20, Length{})
		if a.BorderBox != c.a || b.BorderBox != c.b {
			t.Errorf("%s: %+v %+v, want %+v %+v", c.name, a.BorderBox, b.BorderBox, c.a, c.b)
		}
	}
}

func TestGridFrAgainstMinContent(t *testing.T) {
	for _, c := range []struct {
		name  string
		track Track
		a, b  Rect
	}{
		{"1fr 1fr", flexTrack, Rect{0, 0, 14, 1}, Rect{14, 0, 6, 1}},
		{"minmax(0,1fr) twice", shareTrack, Rect{0, 0, 10, 1}, Rect{10, 0, 10, 1}},
	} {
		a, b := text("abcdefghijklmn"), text("x")
		root := grid(Style{Columns: repeat(2, c.track)}, a, b)
		Layout(root, 20, Length{})
		if a.BorderBox != c.a || b.BorderBox != c.b {
			t.Errorf("%s: %+v %+v, want %+v %+v", c.name, a.BorderBox, b.BorderBox, c.a, c.b)
		}
	}
}

func TestGridNested(t *testing.T) {
	ab, grow := text("ab"), box(Style{Grow: 1})
	flex := box(Style{}, ab, grow)
	left, right := text("l"), text("r")
	inner := grid(Style{Columns: repeat(2, shareTrack)}, left, right)
	root := grid(Style{Columns: repeat(2, shareTrack), ColumnGap: 2}, inner, flex)
	Layout(root, 22, Length{})
	borders(t, []*Box{inner, left, right, flex, ab, grow},
		Rect{0, 0, 10, 1}, Rect{0, 0, 5, 1}, Rect{5, 0, 5, 1},
		Rect{12, 0, 10, 1}, Rect{12, 0, 2, 1}, Rect{14, 0, 8, 1})
}

func TestGridFlowColumn(t *testing.T) {
	items := texts("aa", "bbbb", "c")
	root := grid(Style{Rows: repeat(2, shareTrack), Flow: FlowColumn}, items...)
	Layout(root, 31, Length{})
	borders(t, append([]*Box{root}, items...), Rect{0, 0, 31, 2}, Rect{0, 0, 17, 1}, Rect{0, 1, 17, 1}, Rect{17, 0, 14, 1})
}

func TestGridAlignment(t *testing.T) {
	percent := texts("a", "b")
	root := grid(Style{Columns: []Track{{Breadth{SizePercent, 25}, Breadth{SizePercent, 25}}, flexTrack}}, percent...)
	Layout(root, 40, Length{})
	borders(t, percent, Rect{0, 0, 10, 1}, Rect{10, 0, 30, 1})

	for _, c := range []struct {
		name    string
		justify Justify
		a, b    Rect
	}{
		{"place-content-center", JustifyCenter, Rect{6, 0, 3, 1}, Rect{9, 0, 5, 1}},
		{"justify-start", JustifyStart, Rect{0, 0, 3, 1}, Rect{3, 0, 5, 1}},
		{"justify-normal stretches auto tracks", JustifyStretch, Rect{0, 0, 9, 1}, Rect{9, 0, 11, 1}},
		{"justify-between", JustifyBetween, Rect{0, 0, 3, 1}, Rect{15, 0, 5, 1}},
	} {
		items := texts("abc", "abcde")
		root := grid(Style{Columns: []Track{autoTrack, autoTrack}, Justify: c.justify}, items...)
		Layout(root, 20, Length{})
		if items[0].BorderBox != c.a || items[1].BorderBox != c.b {
			t.Errorf("%s: %+v %+v, want %+v %+v", c.name, items[0].BorderBox, items[1].BorderBox, c.a, c.b)
		}
	}

	centred := texts("abc", "abcde")
	root = grid(Style{Columns: repeat(2, shareTrack), JustifyItems: AlignCenter}, centred...)
	Layout(root, 22, Length{})
	borders(t, centred, Rect{4, 0, 3, 1}, Rect{14, 0, 5, 1})

	end := text("abc")
	end.Style.JustifySelf = AlignEnd
	tall := wrap(30)
	root = grid(Style{Columns: repeat(2, shareTrack), AlignItems: AlignCenter}, end, tall)
	Layout(root, 20, Length{})
	borders(t, []*Box{end, tall}, Rect{7, 1, 3, 1}, Rect{10, 0, 10, 3})
}

func TestGridIntrinsicInFlex(t *testing.T) {
	a, b := text("abc"), text("abcde")
	inner := grid(Style{Columns: []Track{autoTrack, autoTrack}, ColumnGap: 1}, a, b)
	grow := box(Style{Grow: 1})
	root := box(Style{}, inner, grow)
	Layout(root, 30, Length{})
	borders(t, []*Box{inner, grow, a, b}, Rect{0, 0, 9, 1}, Rect{9, 0, 21, 1}, Rect{0, 0, 3, 1}, Rect{4, 0, 5, 1})
}

func TestGridSkipsOutOfFlowAndImplicitLines(t *testing.T) {
	floating := box(Style{Position: PositionAbsolute, Width: cells(3), Height: cells(1), Inset: Insets{Top: cells(0), Left: cells(0)}})
	hidden := box(Style{Display: DisplayNone}, text("x"))
	first, second := text("a"), text("b")
	root := grid(Style{Columns: repeat(2, shareTrack), Position: PositionRelative}, floating, hidden, first, second)
	Layout(root, 20, Length{})
	borders(t, []*Box{floating, hidden, first, second}, Rect{0, 0, 3, 1}, Rect{}, Rect{0, 0, 10, 1}, Rect{10, 0, 10, 1})

	far, near := text("ab"), text("c")
	far.Style.Column = Placement{Start: Line{Index: 4}}
	root = grid(Style{Columns: repeat(2, shareTrack)}, far, near)
	Layout(root, 20, Length{})
	borders(t, []*Box{far, near}, Rect{18, 0, 2, 1}, Rect{0, 1, 9, 1})

	full, next := text("a"), text("b")
	full.Style.Column = Placement{Start: Line{Index: 1}, End: Line{Index: -1}}
	root = grid(Style{}, full, next)
	Layout(root, 20, Length{})
	borders(t, []*Box{root, full, next}, Rect{0, 0, 20, 2}, Rect{0, 0, 20, 1}, Rect{0, 1, 20, 1})
}
