package render_test

import (
	"slices"
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/twi/buffer"
	"github.com/pehcastro/twind/twi/layout"
	"github.com/pehcastro/twind/twi/paint"
	"github.com/pehcastro/twind/twi/style"
)

func menuRows(t *testing.T, tree *render.Tree, root render.Node) []string {
	t.Helper()
	cells := func(n float64) style.Length { return style.Length{Unit: style.Cells, Value: n} }
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "end", Decls: []style.Declaration{
			{Property: style.PropDisplay, Display: style.DisplayFlex}, {Property: style.PropJustify, Justify: style.JustifyEnd},
		}},
		{Class: "relative", Decls: []style.Declaration{{Property: style.PropPosition, Position: style.PositionRelative}}},
		{Class: "absolute", Decls: []style.Declaration{{Property: style.PropPosition, Position: style.PositionAbsolute}}},
		{Class: "top-full", Decls: []style.Declaration{{Property: style.PropTop, Length: style.Length{Unit: style.Percent, Value: 100}}}},
		{Class: "right-0", Decls: []style.Declaration{{Property: style.PropRight, Length: cells(0)}}},
		{Class: "w-3", Decls: []style.Declaration{{Property: style.PropWidth, Length: cells(3)}}},
		{Class: "w-6", Decls: []style.Declaration{{Property: style.PropWidth, Length: cells(6)}}},
		{Class: "nowrap", Decls: []style.Declaration{{Property: style.PropWhiteSpace, WhiteSpace: style.WhiteSpaceNowrap}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	f := render.Frame{Sheet: sheet, Width: 30, Height: layout.Length{Unit: layout.Cells, Value: 6}}
	n, err := tree.Scene(root, f)
	if err != nil {
		t.Fatal(err)
	}
	buf := buffer.New(f.Width, f.Height.Value)
	paint.Paint(buf, n, paint.Composited)
	return rowsOf(buf)
}

func menu(labels ...string) render.Node {
	items := node("absolute top-full right-0")
	for _, l := range labels {
		items.Children = append(items.Children, text(l))
	}
	return node("end", node("relative w-6", text("Open"), items))
}

func TestMinContentMenu(t *testing.T) {
	var tree render.Tree
	for _, step := range []struct {
		labels []string
		want   []string
	}{
		{[]string{"Profile", "Log out"}, []string{
			"                        Open",
			"                       Profile",
			"                       Log out",
			"", "", "",
		}},
		{[]string{"Profile", "Keyboard shortcuts", "Log out"}, []string{
			"                        Open",
			"                     Profile",
			"                     Keyboard",
			"                     shortcuts",
			"                     Log out",
			"",
		}},
	} {
		if got := menuRows(t, &tree, menu(step.labels...)); !slices.Equal(got, step.want) {
			t.Errorf("menu %q:\n%q\nwant\n%q", step.labels, got, step.want)
		}
	}
}

func TestMinContentNarrowerBoxBreaksWord(t *testing.T) {
	got := menuRows(t, new(render.Tree), node("", node("w-3", text("Profile")), node("w-3 nowrap", text("a b c"))))
	want := []string{"Pro", "fil", "e", "a b", "", ""}
	if !slices.Equal(got, want) {
		t.Errorf("narrow boxes:\n%q\nwant\n%q", got, want)
	}
}
