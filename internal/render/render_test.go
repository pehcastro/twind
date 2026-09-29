package render_test

import (
	"slices"
	"strings"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/testdata/hello"
)

func TestSanitiseOncePerText(t *testing.T) {
	sheet, err := hello.Styles()
	if err != nil {
		t.Fatal(err)
	}
	tree := render.Node{
		Classes: []string{"flex", "flex-col", "gap-2", "p-4", "bg-zinc-950", "text-zinc-100"},
		Children: []render.Node{
			{Text: "Hello Twind"},
			{Classes: []string{"border", "rounded-lg", "p-2"}, Children: []render.Node{{Text: "Terminal DOM"}}},
		},
	}
	calls := 0
	sanitize := func(raw string) scene.Text {
		calls++
		return scene.Sanitize(raw)
	}
	buf, err := render.Render(tree, render.Frame{Sheet: sheet, Width: 80, Sanitize: sanitize})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("sanitiser ran %d times for 2 text nodes, want 2", calls)
	}
	if buf.Width() != 80 || buf.Height() != 18 {
		t.Errorf("buffer %dx%d, want 80x18", buf.Width(), buf.Height())
	}
}

func positionSheet(t *testing.T) style.Sheet {
	t.Helper()
	cells := func(n float64) style.Length { return style.Length{Unit: style.Cells, Value: n} }
	fill := func(c color.RGBA) style.Declaration {
		return style.Declaration{Property: style.PropBackground, Color: color.Color{Kind: color.Literal, RGBA: c}}
	}
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "page", Decls: []style.Declaration{
			fill(color.RGBA{R: 9, G: 9, B: 11, A: 255}),
			{Property: style.PropColor, Color: color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 255, G: 255, B: 255, A: 255}}},
			{Property: style.PropHeight, Length: style.Length{Unit: style.Percent, Value: 100}},
		}},
		{Class: "fixed", Decls: []style.Declaration{{Property: style.PropPosition, Position: style.PositionFixed}}},
		{Class: "sticky", Decls: []style.Declaration{{Property: style.PropPosition, Position: style.PositionSticky}}},
		{Class: "inset-0", Decls: []style.Declaration{
			{Property: style.PropTop, Length: cells(0)}, {Property: style.PropRight, Length: cells(0)},
			{Property: style.PropBottom, Length: cells(0)}, {Property: style.PropLeft, Length: cells(0)},
		}},
		{Class: "top-2", Decls: []style.Declaration{{Property: style.PropTop, Length: cells(2)}}},
		{Class: "z-10", Decls: []style.Declaration{{Property: style.PropZIndex, Number: 10}}},
		{Class: "overflow-x-hidden", Decls: []style.Declaration{{Property: style.PropOverflowX, Overflow: style.OverflowHidden}}},
		{Class: "w-2", Decls: []style.Declaration{{Property: style.PropWidth, Length: cells(2)}}},
		{Class: "w-3", Decls: []style.Declaration{{Property: style.PropWidth, Length: cells(3)}}},
		{Class: "w-6", Decls: []style.Declaration{{Property: style.PropWidth, Length: cells(6)}}},
		{Class: "bg-black/50", Decls: []style.Declaration{fill(color.RGBA{A: 128})}},
		{Class: "bg-blue", Decls: []style.Declaration{fill(color.RGBA{R: 43, G: 127, B: 255, A: 255})}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}

func TestPositionOverflowAndZ(t *testing.T) {
	tree := render.Node{Classes: []string{"page"}, Children: []render.Node{
		{Text: "abcdef"},
		{Classes: []string{"w-3", "overflow-x-hidden"}, Children: []render.Node{{Classes: []string{"w-6"}, Text: "uvwxyz"}}},
		{Classes: []string{"fixed", "inset-0", "z-10", "bg-black/50"}},
		{Classes: []string{"fixed", "top-2", "w-2", "bg-blue"}, Text: "ok"},
	}}
	buf, err := render.Render(tree, render.Frame{Sheet: positionSheet(t), Width: 8, Height: layout.Length{Unit: layout.Cells, Value: 3}})
	if err != nil {
		t.Fatal(err)
	}
	var rows []string
	for y := range buf.Height() {
		var row string
		for _, c := range buf.Row(y) {
			row += c.Grapheme
		}
		rows = append(rows, row)
	}
	if want := []string{"abcdef  ", "uvw     ", "ok      "}; !slices.Equal(rows, want) {
		t.Errorf("rows %q, want %q", rows, want)
	}
	for _, at := range []struct {
		x, y int
		want color.RGBA
	}{{0, 0, color.RGBA{R: 4, G: 4, B: 5, A: 255}}, {1, 2, color.RGBA{R: 21, G: 63, B: 127, A: 255}}} {
		if got := buf.At(at.x, at.y).Bg.RGBA; got != at.want {
			t.Errorf("cell %d,%d under the z-10 bg-black/50 backdrop: bg %+v, want %+v", at.x, at.y, got, at.want)
		}
	}
}

func TestPositionSticky(t *testing.T) {
	_, err := render.Render(render.Node{Classes: []string{"sticky"}}, render.Frame{Sheet: positionSheet(t), Width: 4})
	if err == nil || !strings.Contains(err.Error(), "position sticky") {
		t.Errorf("sticky gave %v, want an unsupported error", err)
	}
}

func TestFixedHeight(t *testing.T) {
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{{Class: "h-full", Decls: []style.Declaration{
		{Property: style.PropHeight, Length: style.Length{Unit: style.Percent, Value: 100}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	tree := render.Node{Classes: []string{"h-full"}, Children: []render.Node{{Text: "x"}}}
	buf, err := render.Render(tree, render.Frame{Sheet: sheet, Width: 10, Height: layout.Length{Unit: layout.Cells, Value: 5}})
	if err != nil {
		t.Fatal(err)
	}
	if buf.Height() != 5 || buf.At(0, 0).Grapheme != "x" {
		t.Errorf("fixed height 5 gave %d rows, first cell %q", buf.Height(), buf.At(0, 0).Grapheme)
	}
}
