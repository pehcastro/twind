package layout

import (
	"strings"
	"testing"
)

func BenchmarkLayout1000(b *testing.B) {
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
	b.ReportAllocs()
	for b.Loop() {
		Layout(root, 80, Length{})
	}
}

func BenchmarkGrid1000(b *testing.B) {
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
	b.ReportAllocs()
	for b.Loop() {
		Layout(root, 80, Length{})
	}
}
