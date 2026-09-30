package render_test

import (
	"image"
	"slices"
	"strings"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/paint"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
)

func topSheet(t *testing.T) style.Sheet {
	t.Helper()
	cells := func(n float64) style.Length { return style.Length{Unit: style.Cells, Value: n} }
	fill := func(c color.RGBA) style.Declaration {
		return style.Declaration{Property: style.PropBackground, Color: color.Color{Kind: color.Literal, RGBA: c}}
	}
	ink := func(c color.RGBA) style.Declaration {
		return style.Declaration{Property: style.PropColor, Color: color.Color{Kind: color.Literal, RGBA: c}}
	}
	at := func(p style.Position) style.Declaration {
		return style.Declaration{Property: style.PropPosition, Position: p}
	}
	white, red := color.RGBA{R: 255, G: 255, B: 255, A: 255}, color.RGBA{R: 255, A: 255}
	hidden := []style.Declaration{{Property: style.PropOverflowX, Overflow: style.OverflowHidden}, {Property: style.PropOverflowY, Overflow: style.OverflowHidden}}
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "page", Decls: []style.Declaration{fill(color.RGBA{R: 9, G: 9, B: 11, A: 255}), ink(white), {Property: style.PropHeight, Length: style.Length{Unit: style.Percent, Value: 100}}}},
		{Class: "card", Decls: append([]style.Declaration{at(style.PositionRelative), {Property: style.PropHeight, Length: cells(4)}, fill(color.RGBA{R: 24, G: 24, B: 27, A: 255})}, hidden...)},
		{Class: "scroller", Decls: []style.Declaration{{Property: style.PropOverflowY, Overflow: style.OverflowAuto}, {Property: style.PropHeight, Length: cells(4)}, {Property: style.PropShrink}}},
		{Class: "none", Decls: []style.Declaration{{Property: style.PropDisplay, Display: style.DisplayNone}}},
		{Class: "relative", Decls: []style.Declaration{at(style.PositionRelative)}},
		{Class: "menu", Decls: []style.Declaration{at(style.PositionAbsolute), {Property: style.PropTop, Length: cells(2)}, {Property: style.PropLeft, Length: cells(2)}, {Property: style.PropWidth, Length: cells(8)}, fill(white), ink(color.RGBA{A: 255})}},
		{Class: "later", Decls: []style.Declaration{at(style.PositionRelative), {Property: style.PropZIndex, Number: 10}, {Property: style.PropHeight, Length: cells(3)}, {Property: style.PropShrink}, fill(red)}},
		{Class: "backdrop", Decls: []style.Declaration{
			at(style.PositionFixed), fill(color.RGBA{A: 128}),
			{Property: style.PropTop, Length: cells(0)}, {Property: style.PropRight, Length: cells(0)},
			{Property: style.PropBottom, Length: cells(0)}, {Property: style.PropLeft, Length: cells(0)},
		}},
		{Class: "invisible", Decls: []style.Declaration{{Property: style.PropVisibility, Visibility: style.Hidden}, {Property: style.PropHeight, Length: cells(2)}, fill(red)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}

func menuNode(top int) render.Node {
	m := node("menu", text("one"), text("two"), text("three"), text("four"), node("none", text("gone")))
	m.TopLayer = top
	return m
}

func paintTree(t *testing.T, tree *render.Tree, root render.Node, scroll []int, by int) (*buffer.Buffer, scene.Node) {
	t.Helper()
	f := render.Frame{Sheet: topSheet(t), Width: 16, Height: layout.Length{Unit: layout.Cells, Value: 8}}
	if _, err := tree.Scene(root, f); err != nil {
		t.Fatal(err)
	}
	if scroll != nil && !tree.ScrollBy(scroll, 0, by) {
		t.Fatalf("scroll %v by %d did not move", scroll, by)
	}
	sc, err := tree.Scene(root, f)
	if err != nil {
		t.Fatal(err)
	}
	buf := buffer.New(16, 8)
	paint.Paint(buf, sc, f.Look)
	return buf, sc
}

func frameOf(buf *buffer.Buffer) string {
	marks := map[color.RGBA]string{{R: 255, G: 255, B: 255, A: 255}: "#", {R: 255, A: 255}: "r", {R: 4, G: 4, B: 5, A: 255}: "."}
	var out strings.Builder
	for y := range buf.Height() {
		for x := range buf.Width() {
			c := buf.At(x, y)
			if mark, ok := marks[c.Bg.RGBA]; ok && c.Grapheme == " " {
				out.WriteString(mark)
				continue
			}
			out.WriteString(c.Grapheme)
		}
		out.WriteString("|\n")
	}
	return out.String()
}

func TestTopLayerMenuEscapesCardAndScroller(t *testing.T) {
	for _, c := range []struct {
		name   string
		root   func(top int) render.Node
		scroll []int
		want   []string
	}{
		{"overflow-hidden card", func(top int) render.Node {
			return node("page", node("card", text("card"), menuNode(top)), node("later", text("later")))
		}, nil, []string{
			"card            |",
			"                |",
			"  one#####      |",
			"  two#####      |",
			"lathree###rrrrrr|",
			"rrfour####rrrrrr|",
			"rrrrrrrrrrrrrrrr|",
			"                |",
		}},
		{"scrolled container", func(top int) render.Node {
			return node("page", node("scroller", text("a"), node("relative", text("b"), menuNode(top)), text("c"), text("d"), text("e"), text("f")), node("later", text("later")))
		}, []int{0}, []string{
			"b               |",
			"c               |",
			"d one#####      |",
			"e two#####      |",
			"lathree###rrrrrr|",
			"rrfour####rrrrrr|",
			"rrrrrrrrrrrrrrrr|",
			"                |",
		}},
	} {
		var tree render.Tree
		buf, _ := paintTree(t, &tree, c.root(1), c.scroll, 1)
		got := frameOf(buf)
		t.Logf("%s, top layer:\n%s", c.name, got)
		if want := strings.Join(c.want, "\n") + "\n"; got != want {
			t.Errorf("%s: frame\n%swant\n%s", c.name, got, want)
		}
		var page render.Tree
		buf, _ = paintTree(t, &page, c.root(0), c.scroll, 1)
		t.Logf("%s, in the page:\n%s", c.name, frameOf(buf))
	}
}

func TestTopLayerFlagAloneReclips(t *testing.T) {
	var tree render.Tree
	root := node("page", node("card", text("card"), menuNode(0)), node("later", text("later")))
	paintTree(t, &tree, root, nil, 0)
	root.Children[0].Children[1].TopLayer = 1
	_, sc := paintTree(t, &tree, root, nil, 0)
	menu := sc.Children[0].Children[1]
	if menu.TopLayer != 1 || menu.Clip != (layout.Rect{W: 16, H: 8}) || menu.Children[3].Clip != menu.Clip {
		t.Errorf("menu top layer %d clip %+v, last row clip %+v; want 1 and the viewport for both", menu.TopLayer, menu.Clip, menu.Children[3].Clip)
	}
	if hidden := menu.Children[4]; hidden.Clip != (layout.Rect{}) || hidden.Children[0].Clip != (layout.Rect{}) {
		t.Errorf("a display-none row in the menu got clip %+v and its text %+v, want none", hidden.Clip, hidden.Children[0].Clip)
	}
}

func TestFixedCoversTheViewportFromAScroller(t *testing.T) {
	rows := []render.Node{node("backdrop")}
	for _, s := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		rows = append(rows, text(s))
	}
	var tree render.Tree
	buf, sc := paintTree(t, &tree, node("page", node("card", text("above")), node("scroller", rows...)), []int{1}, 2)
	t.Logf("fixed inset-0 inside a scroller scrolled by 2:\n%s", frameOf(buf))
	for y := range buf.Height() {
		for x := range buf.Width() {
			if bg := buf.At(x, y).Bg.RGBA; bg != (color.RGBA{R: 4, G: 4, B: 5, A: 255}) && bg != (color.RGBA{R: 12, G: 12, B: 13, A: 255}) {
				t.Fatalf("cell %d,%d bg %+v, want the backdrop over the page or the card", x, y, bg)
			}
		}
	}
	backdrop := sc.Children[1].Children[0]
	if screen := (layout.Rect{W: 16, H: 8}); backdrop.Bounds != screen || backdrop.Clip != screen {
		t.Errorf("backdrop bounds %+v clip %+v, want the viewport %+v", backdrop.Bounds, backdrop.Clip, screen)
	}
	var f scene.Frame
	f.Record(&sc, image.Pt(10, 20))
	screen := image.Rect(0, 0, 160, 160)
	if !slices.ContainsFunc(f.Layers[1:], func(l scene.Layer) bool { return l.Clip == screen && l.Visual == screen }) {
		t.Errorf("no pixel layer covers the screen %v unclipped", screen)
	}
}

func TestVisibilityHiddenKeepsItsSpace(t *testing.T) {
	buf, err := render.Render(node("page", node("invisible", text("gone")), text("after")), render.Frame{Sheet: topSheet(t), Width: 8, Height: layout.Length{Unit: layout.Cells, Value: 3}})
	if err != nil {
		t.Fatal(err)
	}
	got := frameOf(buf)
	t.Logf("invisible h-2 then text:\n%s", got)
	if want := "        |\n        |\nafter   |\n"; got != want {
		t.Errorf("frame\n%swant\n%s", got, want)
	}
}
