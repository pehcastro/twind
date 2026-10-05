package render_test

import (
	"testing"
	"time"

	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/motion"
	"github.com/pehcastro/twind/twi/style"
)

func TestTextLeavesTakeTheirOwnParentsStyle(t *testing.T) {
	ink := func(r uint8) style.Declaration {
		return style.Declaration{Property: style.PropColor, Color: color.Color{Kind: color.Literal, RGBA: color.RGBA{R: r, A: 255}}}
	}
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "flex", Decls: []style.Declaration{{Property: style.PropDisplay, Display: style.DisplayFlex}}},
		{Class: "flex-col", Decls: []style.Declaration{{Property: style.PropDirection, Direction: style.Column}}},
		{Class: "red", Decls: []style.Declaration{ink(200)}},
		{Class: "dim", Decls: []style.Declaration{ink(100)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	rtl := &style.NodeState{Attrs: []style.Attr{{Name: "dir", Value: "rtl"}}}
	enter := &motion.Presence{Duration: time.Second}
	page := func(outer string) render.Node {
		return render.Node{Classes: []string{"flex", "flex-col"}, Children: []render.Node{
			{Classes: []string{"flex", outer}, Children: []render.Node{
				{Text: "a1"},
				{Classes: []string{"flex", "dim"}, Children: []render.Node{{Text: "b1"}, {Text: "b2"}}},
				{Text: "a2"},
				{Text: "a3", Enter: enter},
			}},
			{Classes: []string{"flex", outer}, State: rtl, Children: []render.Node{{Text: "c1"}, {Text: "c2"}}},
		}}
	}
	frame := render.Frame{Sheet: sheet, Width: 20, Height: layout.Length{Unit: layout.Cells, Value: 4}, Graphics: true}
	var tree render.Tree
	for _, outer := range []struct {
		class string
		ink   uint8
	}{{"red", 200}, {"dim", 100}, {"red", 200}} {
		got, err := tree.Scene(page(outer.class), frame)
		if err != nil {
			t.Fatal(err)
		}
		first, second := got.Children[0].Children, got.Children[1].Children
		for _, leaf := range []struct {
			name string
			node scene.Node
			ink  uint8
		}{
			{"a1", first[0], outer.ink}, {"b1", first[1].Children[0], 100}, {"b2", first[1].Children[1], 100},
			{"a2", first[2], outer.ink}, {"a3", first[3], outer.ink}, {"c1", second[0], outer.ink}, {"c2", second[1], outer.ink},
		} {
			if leaf.node.Foreground.RGBA != (color.RGBA{R: leaf.ink, A: 255}) {
				t.Errorf("%s under %s: ink %v, want red %d", leaf.name, outer.class, leaf.node.Foreground.RGBA, leaf.ink)
			}
		}
		if c1, c2 := second[0], second[1]; c1.Bounds.X <= c2.Bounds.X {
			t.Errorf("under %s: rtl leaves at %d and %d, want the first on the right", outer.class, c1.Bounds.X, c2.Bounds.X)
		}
	}
}
