package render_test

import (
	"strconv"
	"strings"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
)

func scrollSheet(t *testing.T) style.Sheet {
	t.Helper()
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "h-5", Decls: []style.Declaration{{Property: style.PropHeight, Length: style.Length{Unit: style.Cells, Value: 5}}}},
		{Class: "shrink-0", Decls: []style.Declaration{{Property: style.PropShrink}}},
		{Class: "w-12", Decls: []style.Declaration{{Property: style.PropWidth, Length: style.Length{Unit: style.Cells, Value: 12}}}},
		{Class: "col", Decls: []style.Declaration{{Property: style.PropDisplay, Display: style.DisplayFlex}, {Property: style.PropDirection, Direction: style.Column}}},
		{Class: "overflow-y-auto", Decls: []style.Declaration{{Property: style.PropOverflowY, Overflow: style.OverflowAuto}}},
		{Class: "overflow-hidden", Decls: []style.Declaration{{Property: style.PropOverflowX, Overflow: style.OverflowHidden}, {Property: style.PropOverflowY, Overflow: style.OverflowHidden}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}

func listNode(view string, rows int) render.Node {
	items := make([]render.Node, rows)
	for i := range items {
		items[i] = render.Node{Text: "row " + strconv.Itoa(i)}
	}
	return render.Node{Classes: []string{"col"}, Children: []render.Node{{Classes: strings.Fields(view), Children: items}}}
}

func firstRow(root scene.Node) int {
	view := root.Children[0]
	for i, c := range view.Children {
		if c.Bounds.Y >= view.Padding.Y {
			return i
		}
	}
	return -1
}

func TestScrollAPI(t *testing.T) {
	frame := render.Frame{Sheet: scrollSheet(t), Width: 20, Height: layout.Length{Unit: layout.Cells, Value: 10}}
	view := []int{0}
	var tree render.Tree
	scene := func(n render.Node) scene.Node {
		t.Helper()
		root, err := tree.Scene(n, frame)
		if err != nil {
			t.Fatal(err)
		}
		return root
	}
	list := listNode("col h-5 w-12 overflow-y-auto", 30)
	scene(list)
	for _, c := range []struct {
		name  string
		call  func() bool
		moved bool
		first int
	}{
		{"by three", func() bool { return tree.ScrollBy(view, 0, 3) }, true, 3},
		{"to the end and past it", func() bool { return tree.ScrollTo(view, 0, 99) }, true, 25},
		{"again at the end", func() bool { return tree.ScrollBy(view, 0, 1) }, false, 25},
		{"a row that is not a scroller", func() bool { return tree.ScrollBy([]int{0, 2}, 0, 1) }, false, 25},
		{"a path that does not exist", func() bool { return tree.ScrollBy([]int{4}, 0, 1) }, false, 25},
		{"into view above", func() bool { return tree.ScrollIntoView([]int{0, 10}) }, true, 10},
		{"into view below", func() bool { return tree.ScrollIntoView([]int{0, 20}) }, true, 16},
		{"already in view", func() bool { return tree.ScrollIntoView([]int{0, 18}) }, false, 16},
	} {
		if moved := c.call(); moved != c.moved {
			t.Errorf("%s: moved %v, want %v", c.name, moved, c.moved)
		}
		if first := firstRow(scene(list)); first != c.first {
			t.Errorf("%s: first row in view %d, want %d", c.name, first, c.first)
		}
	}
	list.Children[0].Children[3].Text = "row X"
	if first := firstRow(scene(list)); first != 16 {
		t.Errorf("a text change moved the view to row %d, want it kept at 16", first)
	}
}

func BenchmarkScrollScene(b *testing.B) {
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "h-20", Decls: []style.Declaration{{Property: style.PropHeight, Length: style.Length{Unit: style.Cells, Value: 20}}}},
		{Class: "col", Decls: []style.Declaration{{Property: style.PropDisplay, Display: style.DisplayFlex}, {Property: style.PropDirection, Direction: style.Column}}},
		{Class: "overflow-y-auto", Decls: []style.Declaration{{Property: style.PropOverflowY, Overflow: style.OverflowAuto}}},
	})
	if err != nil {
		b.Fatal(err)
	}
	frame := render.Frame{Sheet: sheet, Width: 90, Height: layout.Length{Unit: layout.Cells, Value: 28}}
	list := listNode("col h-20 overflow-y-auto", 200)
	var tree render.Tree
	if _, err := tree.Scene(list, frame); err != nil {
		b.Fatal(err)
	}
	down := 1
	for b.Loop() {
		if !tree.ScrollBy([]int{0}, 0, down) {
			down = -down
		}
		if _, err := tree.Scene(list, frame); err != nil {
			b.Fatal(err)
		}
	}
}

func TestScrollIntoViewNested(t *testing.T) {
	frame := render.Frame{Sheet: scrollSheet(t), Width: 20, Height: layout.Length{Unit: layout.Cells, Value: 10}}
	inner := listNode("col h-5 w-12 shrink-0 overflow-y-auto", 10).Children[0]
	outer := listNode("col h-5 w-12 overflow-y-auto", 8)
	outer.Children[0].Children = append(outer.Children[0].Children, inner)
	var tree render.Tree
	if _, err := tree.Scene(outer, frame); err != nil {
		t.Fatal(err)
	}
	if !tree.ScrollIntoView([]int{0, 8, 9}) {
		t.Fatal("scrolling the last inner row into view moved nothing")
	}
	root, err := tree.Scene(outer, frame)
	if err != nil {
		t.Fatal(err)
	}
	view, row := root.Children[0], root.Children[0].Children[8].Children[9]
	if row.Bounds.Y < view.Padding.Y || row.Bounds.Y >= view.Padding.Y+view.Padding.H {
		t.Errorf("inner row 9 at y %d, outside the outer view %+v", row.Bounds.Y, view.Padding)
	}
}

func TestScrollbarDriven(t *testing.T) {
	sheet := scrollSheet(t)
	for _, c := range []struct {
		classes string
		want    []string
	}{
		{"col h-5 w-12 overflow-y-auto", []string{"row 0      █", "row 1      █", "row 2", "row 3", "row 4", "", ""}},
		{"col h-5 w-12 overflow-hidden", []string{"row 0", "row 1", "row 2", "row 3", "row 4", "", ""}},
	} {
		app := func(*twi.Runtime) func() twi.Node {
			return func() twi.Node {
				rows := make([]twi.NodeOption, 0, 13)
				rows = append(rows, twi.Class(c.classes))
				for i := range 12 {
					rows = append(rows, twi.Text("row "+strconv.Itoa(i)))
				}
				return twi.Element(twi.Class("col"), twi.Element(rows...))
			}
		}
		d := drive.New(app, drive.Size(16, 7), drive.Styles(sheet))
		got, want := d.Frame().Text(), strings.Join(c.want, "\n")+"\n"
		t.Logf("%s:\n%s", c.classes, got)
		if got != want {
			t.Errorf("%s: frame\n%s\nwant\n%s", c.classes, got, want)
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}
}
