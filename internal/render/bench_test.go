package render_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/internal/render/testdata/sheet"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/style"
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

func BenchmarkRetainedStructural(b *testing.B) {
	styles, err := sheet.Styles()
	if err != nil {
		b.Fatal(err)
	}
	open := &style.NodeState{Attrs: []style.Attr{{Name: "data-state", Value: "open"}}}
	closed := &style.NodeState{Attrs: []style.Attr{{Name: "data-state", Value: "closed"}}}
	page := func(pageClasses string, opened int) render.Node {
		root := render.Node{Classes: strings.Fields(pageClasses)}
		for r := range 100 {
			row := render.Node{Classes: strings.Fields(sheet.Row + " " + sheet.Group), State: closed}
			if r == opened {
				row.State = open
			}
			for c := range 10 {
				row.Children = append(row.Children, render.Node{Classes: strings.Fields(sheet.Focus + " " + sheet.Opened), Text: "cell " + strconv.Itoa(r*10+c)})
			}
			root.Children = append(root.Children, row)
		}
		return root
	}
	for _, bc := range []struct{ name, page string }{
		{"group", sheet.Page},
		{"group under has", sheet.Page + " has-data-[slot=card-action]:grid-cols-[1fr_auto]"},
	} {
		b.Run(bc.name, func(b *testing.B) {
			pages := make([]render.Node, 100)
			for i := range pages {
				pages[i] = page(bc.page, i)
			}
			var tree render.Tree
			frame := render.Frame{Sheet: styles, Width: 200, Height: layout.Length{Unit: layout.Cells, Value: 60}}
			if _, err := tree.Scene(page(bc.page, -1), frame); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				if _, err := tree.Scene(pages[i%len(pages)], frame); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
