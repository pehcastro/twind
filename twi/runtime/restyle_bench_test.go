package runtime

import (
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/style"
)

func BenchmarkRestylesWalk(b *testing.B) {
	lit := []style.Declaration{{Property: style.PropBackground}}
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "peer-hover", Decls: lit, Near: style.Match{Relation: style.RelationPrevious, Class: "peer", States: style.StateHover}},
		{Class: "item-hover", Decls: lit, Near: style.Match{Relation: style.RelationAncestor, Class: "group/item", States: style.StateHover}},
	})
	if err != nil {
		b.Fatal(err)
	}
	rows := make([]render.Node, 200)
	for i := range rows {
		rows[i] = render.Node{Classes: []string{"flex", "group"}}
		for range 10 {
			rows[i].Children = append(rows[i].Children, render.Node{Classes: []string{"px-1", "bg", "peer-hover", "item-hover"}, Children: []render.Node{{Text: "cell"}}})
		}
	}
	root := render.Node{Classes: []string{"flex"}, Children: []render.Node{
		{Classes: []string{"px-1"}, Children: []render.Node{{Text: "side"}}},
		{Classes: []string{"flex", "flex-col"}, Children: rows},
	}}
	for b.Loop() {
		if stateful(sheet, &root, []int{1, 0, 0, 0}, 1, style.StateHover) {
			b.Fatal("a move into the main column found a rule that changes")
		}
	}
}
