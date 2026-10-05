package render_test

import (
	"image"
	"reflect"
	"strconv"
	"strings"
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/twi/buffer"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/layout"
	"github.com/pehcastro/twind/twi/paint"
	"github.com/pehcastro/twind/twi/scene"
	"github.com/pehcastro/twind/twi/style"
)

func retainedSheet(t *testing.T) style.Sheet {
	t.Helper()
	cells := func(n float64) style.Length { return style.Length{Unit: style.Cells, Value: n} }
	literal := func(r, g, b uint8) color.Color {
		return color.Color{Kind: color.Literal, RGBA: color.RGBA{R: r, G: g, B: b, A: 255}}
	}
	padding := func(n float64) []style.Declaration {
		var out []style.Declaration
		for _, p := range []style.Property{style.PropPaddingTop, style.PropPaddingRight, style.PropPaddingBottom, style.PropPaddingLeft} {
			out = append(out, style.Declaration{Property: p, Length: cells(n)})
		}
		return out
	}
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "page", Decls: []style.Declaration{{Property: style.PropBackground, Color: literal(9, 9, 11)}, {Property: style.PropColor, Color: literal(250, 250, 250)}}},
		{Class: "row", Decls: []style.Declaration{{Property: style.PropDisplay, Display: style.DisplayFlex}, {Property: style.PropDirection, Direction: style.Row}, {Property: style.PropColumnGap, Length: cells(1)}}},
		{Class: "text-red", Decls: []style.Declaration{{Property: style.PropColor, Color: literal(255, 0, 0)}}},
		{Class: "text-blue", Decls: []style.Declaration{{Property: style.PropColor, Color: literal(0, 0, 255)}}},
		{Class: "p-1", Decls: padding(1)},
		{Class: "p-2", Decls: padding(2)},
		{Class: "w-6", Decls: []style.Declaration{{Property: style.PropWidth, Length: cells(6)}}},
		{Class: "inline", Decls: []style.Declaration{{Property: style.PropDisplay, Display: style.DisplayInline}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}

type page struct {
	keys, narrow string
	box          []string
	extra        bool
	child        render.Node
}

func (p page) node() render.Node {
	n := render.Node{Classes: []string{"page"}, Children: []render.Node{
		{Classes: []string{"row"}, Children: []render.Node{{Text: p.keys}, {Text: "tail"}}},
		{Classes: p.box, Children: []render.Node{p.child}},
		{Classes: []string{"w-6"}, Children: []render.Node{{Text: p.narrow}}},
	}}
	if p.extra {
		n.Children = append(n.Children, render.Node{Text: "extra"})
	}
	return n
}

func TestRetainedMatchesFresh(t *testing.T) {
	sheet := retainedSheet(t)
	frame := func(width int) render.Frame {
		return render.Frame{Sheet: sheet, Width: width, Height: layout.Length{Unit: layout.Cells, Value: 16}}
	}
	base := page{keys: "keys 1", narrow: "abcd efgh", box: []string{"p-1", "text-red"}, child: render.Node{Text: "child"}}
	type step struct {
		name     string
		page     page
		width    int
		restyle  bool
		cascades int
	}
	steps := []step{{name: "first frame", page: base, width: 30, cascades: 8}}
	add := func(name string, change func(p *page), width int, restyle bool, cascades int) {
		p := steps[len(steps)-1].page
		p.box = append([]string(nil), p.box...)
		change(&p)
		steps = append(steps, step{name, p, width, restyle, cascades})
	}
	add("same size text", func(p *page) { p.keys = "keys 2" }, 30, false, 0)
	add("wider text moves its sibling", func(p *page) { p.keys = "keys 10" }, 30, false, 0)
	add("same natural size, other wrap", func(p *page) { p.narrow = "a bcdefgh" }, 30, false, 0)
	add("inherited colour", func(p *page) { p.box[1] = "text-blue" }, 30, false, 2)
	add("padding only", func(p *page) { p.box[0] = "p-2" }, 30, false, 2)
	add("same classes again", func(*page) {}, 30, false, 0)
	add("text becomes element", func(p *page) {
		p.child = render.Node{Classes: []string{"text-red"}, Children: []render.Node{{Text: "inner"}}}
	}, 30, false, 2)
	add("child added", func(p *page) { p.extra = true }, 30, false, 1)
	add("resize", func(*page) {}, 20, false, 0)
	add("restyle", func(*page) {}, 20, true, 10)

	var tree render.Tree
	var got []scene.Node
	var want []picture
	for _, s := range steps {
		before := tree.Cascades()
		if s.restyle {
			tree.Restyle()
		}
		g, err := tree.Scene(s.page.node(), frame(s.width))
		if err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		w, err := render.Scene(s.page.node(), frame(s.width))
		if err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		if !reflect.DeepEqual(draw(g), draw(w)) {
			t.Errorf("%s: retained scene differs from a fresh one", s.name)
		}
		if n := tree.Cascades() - before; n != s.cascades {
			t.Errorf("%s: %d cascades, want %d", s.name, n, s.cascades)
		}
		got, want = append(got, g), append(want, draw(w))
	}
	for i := range got {
		if !reflect.DeepEqual(draw(got[i]), want[i]) {
			t.Errorf("%s: its scene changed after later frames", steps[i].name)
		}
	}
}

type picture struct {
	cells  [][]buffer.Cell
	layers []scene.Layer
}

func draw(n scene.Node) picture {
	buf := buffer.New(n.Bounds.W, n.Bounds.H)
	paint.Paint(buf, n, paint.Composited)
	var p picture
	for y := range buf.Height() {
		p.cells = append(p.cells, buf.Row(y))
	}
	var f scene.Frame
	f.Record(&n, image.Pt(10, 20))
	p.layers = f.Layers
	return p
}

func TestRetainedStartsCleanAfterAnError(t *testing.T) {
	sheet := retainedSheet(t)
	f := render.Frame{Sheet: sheet, Width: 10}
	ok := render.Node{Classes: []string{"page"}, Children: []render.Node{{Text: "a"}, {Classes: []string{"p-1"}, Text: "b"}}}
	bad := render.Node{Classes: []string{"page"}, Children: []render.Node{{Text: "a"}, {Classes: []string{"inline"}, Text: "b"}}}
	var tree render.Tree
	for i, n := range []render.Node{ok, bad, ok} {
		g, err := tree.Scene(n, f)
		if i == 1 {
			if err == nil || !strings.Contains(err.Error(), "display") {
				t.Fatalf("display inline gave %v, want an unsupported error", err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if w, _ := render.Scene(n, f); !reflect.DeepEqual(draw(g), draw(w)) {
			t.Errorf("frame %d differs from a fresh one", i)
		}
	}
}

func thousand(keys int) render.Node {
	left := 1000 - 2
	levels := [][]string{{"flex", "flex-col", "border"}, {"row"}, {"flex", "flex-col", "grow"}}
	var grow func(depth int) render.Node
	grow = func(depth int) render.Node {
		left--
		if depth == len(levels) {
			return render.Node{Text: "item " + strconv.Itoa(left)}
		}
		n := render.Node{Classes: levels[depth]}
		for range 4 {
			if left == 0 {
				break
			}
			n.Children = append(n.Children, grow(depth+1))
		}
		return n
	}
	root := render.Node{Classes: []string{"page"}, Children: []render.Node{{Text: "keys " + strconv.Itoa(keys)}}}
	for left > 0 {
		root.Children = append(root.Children, grow(0))
	}
	return root
}

func TestOneTextReCascadesAtMostItself(t *testing.T) {
	f := render.Frame{Sheet: retainedSheet(t), Width: 80, Height: layout.Length{Unit: layout.Cells, Value: 24}}
	var tree render.Tree
	if _, err := tree.Scene(thousand(1), f); err != nil {
		t.Fatal(err)
	}
	if n := tree.Cascades(); n != 1000 {
		t.Fatalf("first frame cascaded %d nodes, want 1000", n)
	}
	for keys := 2; keys < 12; keys++ {
		before := tree.Cascades()
		if _, err := tree.Scene(thousand(keys), f); err != nil {
			t.Fatal(err)
		}
		if n := tree.Cascades() - before; n > 1 {
			t.Errorf("changing one text to keys %d re-cascaded %d nodes, want at most 1", keys, n)
		}
	}
}
