package render_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/internal/render/testdata/sheet"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/style"
)

func seams(labels ...string) []render.Node {
	out := make([]render.Node, len(labels))
	for i, l := range labels {
		out[i] = node(sheet.Seam+" "+sheet.Striped, text(l))
	}
	return out
}

func TestPlaceButtonGroup(t *testing.T) {
	f := cssFrame(t, 40)
	none, md := style.RadiusNone, style.RadiusMd
	first, middle, last := [4]style.Radius{md, none, none, md}, [4]style.Radius{}, [4]style.Radius{none, md, md, none}
	var tree render.Tree
	for _, step := range []struct {
		name     string
		group    render.Node
		want     [][4]style.Radius
		cascades int
	}{
		{"three buttons", node(sheet.Row, seams("a", "b", "c")...), [][4]style.Radius{first, middle, last}, 7},
		{"the same frame again", node(sheet.Row, seams("a", "b", "c")...), [][4]style.Radius{first, middle, last}, 0},
		{"the first removed", node(sheet.Row, seams("b", "c")...), [][4]style.Radius{first, last}, 2},
		{"a text run before them", node(sheet.Row, append([]render.Node{text("x")}, seams("b", "c")...)...), [][4]style.Radius{{}, first, last}, 5},
		{"reversed", node(sheet.Leftward, seams("a", "b", "c")...), [][4]style.Radius{first, middle, last}, 7},
	} {
		before := tree.Cascades()
		root, err := tree.Scene(step.group, f)
		if err != nil {
			t.Fatal(err)
		}
		var got [][4]style.Radius
		for _, c := range root.Children {
			r := c.Border.Radius
			got = append(got, [4]style.Radius{r.At(style.CornerTopLeft), r.At(style.CornerTopRight), r.At(style.CornerBottomRight), r.At(style.CornerBottomLeft)})
		}
		if !slices.Equal(got, step.want) {
			t.Errorf("%s: radii %v, want %v (top left, top right, bottom right, bottom left)", step.name, got, step.want)
		}
		if n := tree.Cascades() - before; n != step.cascades {
			t.Errorf("%s: %d cascades, want %d", step.name, n, step.cascades)
		}
	}
}

func TestPlaceAppendRestylesOnlyTheEnds(t *testing.T) {
	f := cssFrame(t, 40)
	list := func(n int) render.Node {
		rows := make([]render.Node, n)
		for i := range rows {
			rows[i] = render.Node{Classes: strings.Fields(sheet.Seam + " " + sheet.Striped), Text: "row"}
		}
		return node(sheet.Column, rows...)
	}
	var tree render.Tree
	if _, err := tree.Scene(list(4), f); err != nil {
		t.Fatal(err)
	}
	before := tree.Cascades()
	root, err := tree.Scene(list(5), f)
	if err != nil {
		t.Fatal(err)
	}
	if n := tree.Cascades() - before; n != 2 {
		t.Errorf("appending a fifth row: %d cascades, want 2 (the old last and the new last)", n)
	}
	odd, even := f.Sheet.Compute(style.ComputedStyle{}, []string{"bg-zinc-800"}).Background, f.Sheet.Compute(style.ComputedStyle{}, []string{"bg-zinc-900"}).Background
	for i, c := range root.Children {
		if want := [2]color.Color{odd, even}[i%2]; c.Background != want {
			t.Errorf("row %d (nth-child %d): background %v, want %v", i, i+1, c.Background, want)
		}
	}
}
