package render_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/render/testdata/sheet"
	"github.com/pehcastro/twind/twi/style"
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
	before := tree.Cascades()
	b.ResetTimer()
	for i := range b.N {
		if _, err := tree.Scene(pages[i%len(pages)], frame); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(tree.Cascades()-before)/float64(b.N), "cascades/op")
}

func BenchmarkRetainedAnimating(b *testing.B) {
	styles, err := sheet.Styles()
	if err != nil {
		b.Fatal(err)
	}
	root := render.Node{Classes: strings.Fields(sheet.Page)}
	for r := range 100 {
		row := render.Node{Classes: strings.Fields(sheet.Row)}
		for c := range 10 {
			cell := render.Node{Classes: strings.Fields(sheet.Focus), Text: "cell " + strconv.Itoa(r*10+c)}
			if c == 0 {
				cell.Classes = strings.Fields(sheet.Pulsing)
			}
			row.Children = append(row.Children, cell)
		}
		root.Children = append(root.Children, row)
	}
	var tree render.Tree
	frame := render.Frame{Sheet: styles, Width: 200, Height: layout.Length{Unit: layout.Cells, Value: 60}}
	if _, err := tree.Scene(root, frame); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		frame.Now = time.Duration(i+1) * 16 * time.Millisecond
		if _, err := tree.Scene(root, frame); err != nil {
			b.Fatal(err)
		}
	}
	if _, moving := tree.Wake(); !moving {
		b.Fatal("100 pulsing nodes report no wake")
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
