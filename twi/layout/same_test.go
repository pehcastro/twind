package layout

import (
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
)

func cardsTree() *Box {
	root := box(Style{Direction: Column, RowGap: 1, Padding: Edges{1, 2, 1, 2}})
	for range 50 {
		header := box(Style{ColumnGap: 2, AlignItems: AlignCenter})
		footer := box(Style{ColumnGap: 1})
		for i := range 5 {
			label := strings.Repeat("badge", i+1)
			header.Children = append(header.Children, box(Style{Overflow: OverflowHidden, Padding: Edges{Left: 1, Right: 1}}, &Box{Style: Style{Shrink: 1}, Measure: text(label).Measure}))
			footer.Children = append(footer.Children, box(Style{Grow: 1, Shrink: 1, Basis: cells(0)}, text("ok")))
		}
		card := box(Style{Direction: Column, Border: Edges{1, 1, 1, 1}, Padding: Edges{Left: 1, Right: 1}}, header, wrap(300), footer)
		root.Children = append(root.Children, card)
	}
	return root
}

func gridTree() *Box {
	root := box(Style{Direction: Column, RowGap: 1, Padding: Edges{1, 2, 1, 2}})
	for range 50 {
		action := box(Style{Column: Placement{Start: Line{Index: 2}}, Row: Placement{Start: Line{Index: 1}, End: Line{Span: 2}}, AlignSelf: AlignStart, JustifySelf: AlignEnd}, text("Button"))
		header := grid(Style{Columns: []Track{flexTrack, autoTrack}, Rows: []Track{autoTrack, autoTrack}, AutoRows: []Track{minTrack}, AlignItems: AlignStart, RowGap: 2, ColumnGap: 2}, text("Card Title"), wrap(120), action)
		body := grid(Style{Columns: repeat(3, shareTrack), RowGap: 1, ColumnGap: 2})
		for i := range 13 {
			cell := wrap(10 + 7*i)
			if i%4 == 0 {
				cell.Style.Column = Placement{Start: Line{Span: 2}, End: Line{Span: 2}}
			}
			body.Children = append(body.Children, cell)
		}
		root.Children = append(root.Children, box(Style{Direction: Column, Border: Edges{1, 1, 1, 1}, Padding: Edges{Left: 1, Right: 1}}, header, body))
	}
	return root
}

func randomLength(r *rand.Rand, auto int) Length {
	switch n := r.IntN(10); {
	case n < auto:
		return Length{}
	case n < 8:
		return cells(r.IntN(30))
	}
	return pct(r.IntN(101))
}

func randomBreadth(r *rand.Rand) Breadth {
	switch k := TrackSize(r.IntN(6)); k {
	case SizeCells:
		return Breadth{k, r.IntN(16)}
	case SizePercent:
		return Breadth{k, r.IntN(101)}
	case SizeFr:
		return Breadth{k, 25 * (1 + r.IntN(8))}
	default:
		return Breadth{Kind: k}
	}
}

func randomTracks(r *rand.Rand, most int) []Track {
	ts := make([]Track, r.IntN(most+1))
	for i := range ts {
		ts[i] = Track{randomBreadth(r), randomBreadth(r)}
	}
	return ts
}

func randomLine(r *rand.Rand) Line {
	if r.IntN(2) == 0 {
		return Line{Span: r.IntN(3)}
	}
	return Line{Index: r.IntN(8) - 3}
}

func randomEdges(r *rand.Rand, low, high int) Edges {
	if r.IntN(2) == 0 {
		return Edges{}
	}
	n := func() int { return low + r.IntN(high-low+1) }
	return Edges{n(), n(), n(), n()}
}

func randomTree(r *rand.Rand, depth int) *Box {
	s := Style{
		Grow:      r.IntN(4) * r.IntN(2),
		Shrink:    r.IntN(4) - r.IntN(2),
		Basis:     randomLength(r, 8),
		Width:     randomLength(r, 7),
		Height:    randomLength(r, 8),
		MinWidth:  randomLength(r, 8),
		MinHeight: randomLength(r, 9),
		MaxWidth:  randomLength(r, 9),
		MaxHeight: randomLength(r, 9),
		AlignSelf: Align(r.IntN(5) * r.IntN(2)),
		Margin:    randomEdges(r, -1, 2),
		Padding:   randomEdges(r, 0, 2),
		Border:    randomEdges(r, 0, 1),
		Overflow:  Overflow(r.IntN(3) * r.IntN(2)),
		Column:    Placement{randomLine(r), randomLine(r)},
		Row:       Placement{randomLine(r), randomLine(r)},
	}
	if r.IntN(8) == 0 {
		s.Aspect = Ratio{1 + r.IntN(4), 1 + r.IntN(3)}
	}
	switch n := r.IntN(20); {
	case n < 2:
		s.Position = PositionRelative
	case n < 4:
		s.Position = PositionAbsolute
	case n < 5:
		s.Position = PositionFixed
	}
	if s.Position != PositionStatic {
		s.Inset = Insets{randomLength(r, 6), randomLength(r, 6), randomLength(r, 6), randomLength(r, 6)}
	}
	if r.IntN(25) == 0 {
		s.Display = DisplayNone
	}
	b := &Box{Style: s, ScrollX: r.IntN(20), ScrollY: r.IntN(20)}
	if depth == 0 || r.IntN(4) == 0 {
		switch r.IntN(3) {
		case 0:
			b.Measure = text(strings.Repeat("w", r.IntN(14))).Measure
		case 1:
			b.Measure = wrap(r.IntN(80)).Measure
		}
		return b
	}
	b.Style.Direction = Direction(r.IntN(2))
	b.Style.Wrap = Wrapping(r.IntN(3) * r.IntN(2))
	b.Style.Justify = Justify(r.IntN(7))
	b.Style.AlignItems = Align(r.IntN(5))
	b.Style.RowGap, b.Style.ColumnGap = r.IntN(3), r.IntN(3)
	if r.IntN(3) == 0 {
		b.Style.Display = DisplayGrid
		b.Style.Columns, b.Style.Rows = randomTracks(r, 4), randomTracks(r, 3)
		b.Style.AutoColumns, b.Style.AutoRows = randomTracks(r, 2), randomTracks(r, 2)
		b.Style.Flow = Flow(r.IntN(4))
		b.Style.JustifyItems, b.Style.JustifySelf = Align(r.IntN(5)), Align(r.IntN(5))
		b.Style.AlignContent = Justify(r.IntN(7))
	}
	for range r.IntN(6) {
		b.Children = append(b.Children, randomTree(r, depth-1))
	}
	return b
}

func boxesHash(root *Box) uint64 {
	var boxes []byte
	var walk func(b *Box)
	walk = func(b *Box) {
		boxes = fmt.Append(boxes, b.BorderBox, b.PaddingBox, b.ContentBox, b.Clip, b.ScrollX, b.ScrollY, b.ScrollWidth, b.ScrollHeight)
		for _, c := range b.Children {
			walk(c)
		}
	}
	walk(root)
	h := fnv.New64a()
	h.Write(boxes)
	return h.Sum64()
}

func invalidate(b *Box) {
	b.current, b.prepared = false, false
	for _, c := range b.Children {
		invalidate(c)
	}
}

func full(root *Box, width int, height Length) {
	invalidate(root)
	Layout(root, width, height)
}

func laidOut(lay func(*Box, int, Length), root *Box, width int, height Length) (hash string) {
	defer func() {
		if recover() != nil {
			hash = "panic"
		}
	}()
	lay(root, width, height)
	return fmt.Sprintf("%016x", boxesHash(root))
}

func treeHashes(lay func(*Box, int, Length)) []string {
	var out []string
	sizes := []struct {
		width  int
		height Length
	}{{80, Length{}}, {37, cells(20)}, {120, cells(9)}, {3, Length{}}}
	for _, root := range []*Box{cardsTree(), gridTree()} {
		for _, size := range sizes {
			out = append(out, laidOut(lay, root, size.width, size.height))
		}
	}
	for i := range 1000 {
		r := rand.New(rand.NewPCG(93, uint64(i)))
		root := randomTree(r, 1+r.IntN(4))
		line := ""
		for range 3 {
			height := Length{}
			if r.IntN(2) == 0 {
				height = cells(r.IntN(50))
			}
			line += laidOut(lay, root, r.IntN(100), height) + " "
		}
		out = append(out, strings.TrimSpace(line))
	}
	return out
}

func TestSameBoxesColumnShrunkOverMax(t *testing.T) {
	capped := box(Style{Basis: cells(3), MaxHeight: cells(1), Shrink: 1})
	negative := box(Style{Basis: cells(3), Shrink: -1})
	root := box(Style{Direction: Column}, capped, negative)
	Layout(root, 10, Length{})
	borders(t, []*Box{root, capped, negative}, Rect{0, 0, 10, 4}, Rect{0, 0, 10, 3}, Rect{0, 3, 10, 3})
}

func TestSameBoxesAsBefore(t *testing.T) {
	data, err := os.ReadFile("testdata/boxes.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Split(strings.TrimSpace(string(data)), "\n")
	for name, lay := range map[string]func(*Box, int, Length){"full": full, "incremental": Layout} {
		got := treeHashes(lay)
		if len(got) != len(want) {
			t.Fatalf("%s: %d trees, fixture has %d", name, len(got), len(want))
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%s: tree %d: %s, want %s", name, i, got[i], want[i])
			}
		}
	}
}
