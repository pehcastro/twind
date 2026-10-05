package render_test

import (
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/style"
)

func TestCousinsShareOnlyWhatTheyInherit(t *testing.T) {
	ink := func(r uint8) style.Declaration {
		return style.Declaration{Property: style.PropColor, Color: color.Color{Kind: color.Literal, RGBA: color.RGBA{R: r, A: 255}}}
	}
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "flex", Decls: []style.Declaration{{Property: style.PropDisplay, Display: style.DisplayFlex}}},
		{Class: "flex-col", Decls: []style.Declaration{{Property: style.PropDirection, Direction: style.Column}}},
		{Class: "red", Decls: []style.Declaration{ink(200)}},
		{Class: "dim", Decls: []style.Declaration{ink(100)}},
		{Class: "basis-2", Decls: []style.Declaration{{Property: style.PropBasis, Length: style.Length{Unit: style.Cells, Value: 2}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	pair := func(classes ...string) render.Node {
		return render.Node{Classes: classes, Children: []render.Node{
			{Classes: []string{"basis-2"}, Text: "a"},
			{Text: "b"},
		}}
	}
	root := render.Node{Classes: []string{"flex", "flex-col"}, Children: []render.Node{
		pair("flex", "red"),
		pair("flex", "flex-col", "red"),
		pair("flex", "dim"),
	}}
	got, err := render.Scene(root, render.Frame{Sheet: sheet, Width: 10, Height: layout.Length{Unit: layout.Cells, Value: 8}, Graphics: true})
	if err != nil {
		t.Fatal(err)
	}
	ink200, ink100 := color.RGBA{R: 200, A: 255}, color.RGBA{R: 100, A: 255}
	for i, want := range []struct {
		ink    color.RGBA
		bounds [2]layout.Rect
	}{
		{ink200, [2]layout.Rect{{X: 0, Y: 0, W: 2, H: 1}, {X: 2, Y: 0, W: 1, H: 1}}},
		{ink200, [2]layout.Rect{{X: 0, Y: 1, W: 10, H: 2}, {X: 0, Y: 3, W: 10, H: 1}}},
		{ink100, [2]layout.Rect{{X: 0, Y: 4, W: 2, H: 1}, {X: 2, Y: 4, W: 1, H: 1}}},
	} {
		for j, c := range got.Children[i].Children {
			if c.Foreground.RGBA != want.ink || c.Bounds != want.bounds[j] {
				t.Errorf("pair %d child %d: ink %v at %+v, want %v at %+v", i, j, c.Foreground.RGBA, c.Bounds, want.ink, want.bounds[j])
			}
		}
	}
}
