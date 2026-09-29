package render_test

import (
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/internal/render"
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
