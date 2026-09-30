package render_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/internal/render/testdata/sheet"
	"github.com/twind-dev/twind/twi/layout"
)

func BenchmarkRetained(b *testing.B) {
	styles, err := sheet.Styles()
	if err != nil {
		b.Fatal(err)
	}
	page := func(changed int) render.Node {
		root := render.Node{Classes: strings.Fields(sheet.Page)}
		for r := range 100 {
			row := render.Node{Classes: strings.Fields(sheet.Row)}
			for c := range 10 {
				text := "cell " + strconv.Itoa(r*10+c)
				if r*10+c == changed {
					text = "changed"
				}
				row.Children = append(row.Children, render.Node{Classes: strings.Fields(sheet.Focus), Text: text})
			}
			root.Children = append(root.Children, row)
		}
		return root
	}
	pages := make([]render.Node, 1000)
	for i := range pages {
		pages[i] = page(i)
	}
	var tree render.Tree
	frame := render.Frame{Sheet: styles, Width: 200, Height: layout.Length{Unit: layout.Cells, Value: 60}}
	if _, err := tree.Scene(page(-1), frame); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		if _, err := tree.Scene(pages[i%len(pages)], frame); err != nil {
			b.Fatal(err)
		}
	}
}
