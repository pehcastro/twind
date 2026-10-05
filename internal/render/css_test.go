package render_test

import (
	"image"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/buffer"
	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/paint"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/render/testdata/sheet"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/style"
)

func cssFrame(t *testing.T, width int) render.Frame {
	t.Helper()
	styles, err := sheet.Styles()
	if err != nil {
		t.Fatal(err)
	}
	return render.Frame{Sheet: styles, Width: width}
}

func rowsOf(buf *buffer.Buffer) []string {
	var rows []string
	for y := range buf.Height() {
		var row strings.Builder
		for _, c := range buf.Row(y) {
			row.WriteString(c.Grapheme)
		}
		rows = append(rows, strings.TrimRight(row.String(), " "))
	}
	return rows
}

func TestTruncateAndNowrap(t *testing.T) {
	const long = "abcdefghijklmnopqrstuvwxyz0123"
	classes := strings.Fields
	for _, c := range []struct {
		name string
		node render.Node
		want []string
	}{
		{"truncate, own text", render.Node{Classes: classes(sheet.Truncate), Text: long}, []string{"abcdefghi…"}},
		{"truncate, text child", render.Node{Classes: classes(sheet.Truncate), Children: []render.Node{{Text: long}}}, []string{"abcdefghi…"}},
		{"truncate, classed child clips", render.Node{Classes: classes(sheet.Truncate), Children: []render.Node{{Classes: classes(sheet.TextClip), Text: long}}}, []string{"abcdefghij"}},
		{"truncate text-clip", render.Node{Classes: classes(sheet.Truncate + " " + sheet.TextClip), Text: long}, []string{"abcdefghij"}},
		{"text-ellipsis on wrapping text", render.Node{Classes: classes(sheet.Ellipsis), Text: "hello world again"}, []string{"hello", "world", "again"}},
		{"nowrap in an overflow-hidden row", render.Node{Classes: classes(sheet.Clip), Children: []render.Node{{Classes: classes(sheet.Nowrap), Text: "hello world again"}}}, []string{"hello worl"}},
	} {
		buf, err := render.Render(c.node, cssFrame(t, 20))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := rowsOf(buf); !slices.Equal(got, c.want) {
			t.Errorf("%s: rows %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFocusVisibleRing(t *testing.T) {
	f := cssFrame(t, 20)
	page := func(states style.State, attrs ...style.Attr) render.Node {
		return render.Node{Classes: strings.Fields(sheet.Page), Children: []render.Node{
			{Text: "before"},
			{Classes: strings.Fields(sheet.Focus), State: &style.NodeState{States: states, Attrs: attrs}, Children: []render.Node{{Text: "ok"}}},
			{Text: "after"},
		}}
	}
	visible := style.StateFocusVisible
	var tree render.Tree
	steps := []struct {
		name     string
		states   style.State
		attrs    []style.Attr
		rings    int
		cascades int
	}{
		{"first frame", 0, nil, 0, 5},
		{"focus only", style.StateFocus, nil, 0, 1},
		{"focus-visible", visible, nil, 1, 2},
		{"new attrs", visible, []style.Attr{{Name: "data-state", Value: "open"}}, 1, 1},
		{"same attrs, rebuilt slice", visible, []style.Attr{{Name: "data-state", Value: "open"}}, 1, 0},
		{"focus lost", 0, nil, 0, 2},
	}
	for _, s := range steps {
		before := tree.Cascades()
		got, err := tree.Scene(page(s.states, s.attrs...), f)
		if err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		fresh, err := render.Scene(page(s.states, s.attrs...), f)
		if err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		buf := buffer.New(f.Width, got.Bounds.H)
		paint.Paint(buf, got, paint.Composited)
		t.Logf("%s:\n%s", s.name, strings.Join(rowsOf(buf), "\n"))
		if n := len(got.Children[1].Shadows); n != s.rings {
			t.Errorf("%s: %d ring shadows, want %d", s.name, n, s.rings)
		}
		if !reflect.DeepEqual(draw(got), draw(fresh)) {
			t.Errorf("%s: retained scene differs from a fresh one", s.name)
		}
		if n := tree.Cascades() - before; n != s.cascades {
			t.Errorf("%s: %d cascades, want %d", s.name, n, s.cascades)
		}
	}
}

func TestRetainedWhiteSpace(t *testing.T) {
	f := cssFrame(t, 20)
	page := func(nowrap bool) render.Node {
		text := render.Node{Text: "hello world again"}
		if nowrap {
			text.Classes = strings.Fields(sheet.Nowrap)
		}
		return render.Node{Classes: strings.Fields(sheet.Clip), Children: []render.Node{text}}
	}
	var tree render.Tree
	for _, nowrap := range []bool{false, true, false} {
		got, err := tree.Scene(page(nowrap), f)
		if err != nil {
			t.Fatal(err)
		}
		fresh, err := render.Scene(page(nowrap), f)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(draw(got), draw(fresh)) {
			t.Errorf("nowrap %v: retained scene differs from a fresh one", nowrap)
		}
	}
}

func TestFitContent(t *testing.T) {
	f := cssFrame(t, 30)
	fit := render.Node{Classes: strings.Fields(sheet.Fit), Text: "fit"}
	for _, c := range []struct {
		name   string
		parent string
		want   layout.Rect
	}{
		{"column", sheet.Column, layout.Rect{X: 0, Y: 0, W: 3, H: 1}},
		{"centred column", sheet.Column + " " + sheet.Centred, layout.Rect{X: 8, Y: 0, W: 3, H: 1}},
		{"row", sheet.Row, layout.Rect{X: 0, Y: 0, W: 3, H: 1}},
	} {
		root, err := render.Scene(render.Node{Classes: strings.Fields(c.parent), Children: []render.Node{fit}}, f)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := root.Children[0].Bounds; got != c.want {
			t.Errorf("%s: w-fit h-fit box %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestFlexWrap(t *testing.T) {
	f := cssFrame(t, 20)
	item := func(text string) render.Node { return render.Node{Classes: strings.Fields(sheet.Shrink0), Text: text} }
	for _, c := range []struct {
		row  string
		want []image.Point
	}{
		{sheet.Clip, []image.Point{{0, 0}, {4, 0}, {8, 0}}},
		{sheet.Wrap, []image.Point{{0, 0}, {5, 0}, {0, 1}}},
		{sheet.Reverse, []image.Point{{0, 1}, {5, 1}, {0, 0}}},
	} {
		root, err := render.Scene(render.Node{Classes: strings.Fields(c.row), Children: []render.Node{item("abcd"), item("efgh"), item("ij")}}, f)
		if err != nil {
			t.Fatalf("%s: %v", c.row, err)
		}
		var got []image.Point
		for _, child := range root.Children {
			got = append(got, image.Pt(child.Bounds.X, child.Bounds.Y))
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: items at %v, want %v", c.row, got, c.want)
		}
	}
}

func TestDrivenTruncateFitAspect(t *testing.T) {
	styles, err := sheet.Styles()
	if err != nil {
		t.Fatal(err)
	}
	app := func(*twi.Runtime) func() twi.Node {
		return func() twi.Node {
			return twi.Element(twi.Class(sheet.Page),
				twi.Element(twi.Class(sheet.Truncate), twi.Text("abcdefghijklmnopqrstuvwxyz0123")),
				twi.Element(twi.Class(sheet.Clip), twi.Element(twi.Class(sheet.Nowrap), twi.Text("hello world again"))),
				twi.Element(twi.Class(sheet.Column+" "+sheet.Centred), twi.Element(twi.Class(sheet.Fit), twi.Text("fit"))),
				twi.Element(twi.Class(sheet.Video), twi.Text("video")),
			)
		}
	}
	d := drive.New(app, drive.Size(24, 10), drive.With(twi.Styles(styles)))
	got := d.Frame().Text()
	t.Logf("frame 24x10:\n%s", got)
	want := strings.Join([]string{"abcdefghi…", "hello worl", "        fit", "video", "", "", "", "", "", ""}, "\n") + "\n"
	if got != want {
		t.Errorf("frame\n%s\nwant\n%s", got, want)
	}
	cells := d.Frame().Cells()
	video, fill := cells.At(0, 3).Bg, 0
	var shade strings.Builder
	for y := range cells.Height() {
		for x := range cells.Width() {
			shade.WriteString(map[bool]string{true: "#", false: "."}[cells.At(x, y).Bg == video])
		}
		shade.WriteString("\n")
		if cells.At(19, y).Bg == video {
			fill++
		}
	}
	t.Logf("cells with the aspect-video background:\n%s", shade.String())
	if fill != 6 {
		t.Errorf("aspect-video on w-20 fills %d rows, want 6 (a nominal 8x16 cell)", fill)
	}
	if err := d.Close(); err != nil {
		t.Error(err)
	}
}

func TestAspectFromCellPixels(t *testing.T) {
	f := cssFrame(t, 30)
	video := render.Node{Classes: strings.Fields(sheet.Video)}
	var tree render.Tree
	for _, c := range []struct {
		cell image.Point
		want int
	}{
		{image.Point{}, 6},
		{image.Pt(10, 24), 5},
		{image.Pt(12, 16), 8},
		{image.Point{}, 6},
	} {
		f.Cell = c.cell
		fresh, err := render.Scene(video, f)
		if err != nil {
			t.Fatalf("cell %v: %v", c.cell, err)
		}
		retained, err := tree.Scene(video, f)
		if err != nil {
			t.Fatalf("cell %v: %v", c.cell, err)
		}
		for name, got := range map[string]layout.Rect{"fresh": fresh.Bounds, "retained": retained.Bounds} {
			if got.W != 20 || got.H != c.want {
				t.Errorf("%s, cell %v: aspect-video on w-20 is %dx%d, want 20x%d", name, c.cell, got.W, got.H, c.want)
			}
		}
	}
}
