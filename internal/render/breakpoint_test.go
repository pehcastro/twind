package render_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/internal/render/testdata/sheet"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/paint"
	"github.com/twind-dev/twind/twi/scene"
)

func node(classes string, children ...render.Node) render.Node {
	return render.Node{Classes: strings.Fields(classes), Children: children}
}

func text(s string) render.Node { return render.Node{Text: s} }

func frameRows(n scene.Node) []string {
	buf := buffer.New(n.Bounds.W, n.Bounds.H)
	paint.Paint(buf, n, paint.Composited)
	return rowsOf(buf)
}

func TestBreakpointRetained(t *testing.T) {
	root := node(sheet.Page, node(sheet.Stack, text("one"), text("two")), node(sheet.Padded, text("pad")))
	for range 5 {
		root.Children = append(root.Children, node(sheet.Shrink0, text("row")))
	}
	var tree render.Tree
	for _, step := range []struct {
		width, cascades int
		stacked         bool
		pad             int
	}{
		{99, 16, true, 0},
		{90, 0, true, 0},
		{100, 5, false, 2},
		{139, 0, false, 2},
		{140, 3, false, 4},
		{99, 5, true, 0},
	} {
		before := tree.Cascades()
		f := cssFrame(t, step.width)
		got, err := tree.Scene(root, f)
		if err != nil {
			t.Fatalf("%d columns: %v", step.width, err)
		}
		fresh, err := render.Scene(root, f)
		if err != nil {
			t.Fatalf("%d columns: %v", step.width, err)
		}
		if !reflect.DeepEqual(draw(got), draw(fresh)) {
			t.Errorf("%d columns: retained scene differs from a fresh one", step.width)
		}
		if n := tree.Cascades() - before; n != step.cascades {
			t.Errorf("%d columns: %d cascades, want %d", step.width, n, step.cascades)
		}
		one, two := got.Children[0].Children[0].Bounds, got.Children[0].Children[1].Bounds
		if stacked := two.Y > one.Y; stacked != step.stacked {
			t.Errorf("%d columns: one at %+v, two at %+v, stacked %v, want %v", step.width, one, two, stacked, step.stacked)
		}
		padded := got.Children[1]
		if pad := padded.Content.X - padded.Bounds.X; pad != step.pad {
			t.Errorf("%d columns: padding %d, want %d", step.width, pad, step.pad)
		}
	}
}

func TestBreakpointDialogFooter(t *testing.T) {
	dialog := node(sheet.Dialog, node(sheet.Footer, node(sheet.Button, text("Cancel")), node(sheet.Button, text("Save"))))
	for _, tc := range []struct {
		width int
		want  []string
	}{
		{70, []string{"", "", "    Save", "", "", "    Cancel", "", ""}},
		{120, []string{"", "", strings.Repeat(" ", 100) + "Cancel      Save", "", ""}},
	} {
		n, err := render.Scene(dialog, cssFrame(t, tc.width))
		if err != nil {
			t.Fatalf("%d columns: %v", tc.width, err)
		}
		rows := frameRows(n)
		t.Logf("%d columns:\n%s", tc.width, strings.Join(rows, "\n"))
		inner := make([]string, len(rows))
		for i, r := range rows {
			inner[i] = strings.TrimRight(strings.Map(func(r rune) rune {
				if strings.ContainsRune("─│┌┐└┘╭╮╰╯", r) {
					return ' '
				}
				return r
			}, r), " ")
		}
		if !slices.Equal(inner, tc.want) {
			t.Errorf("%d columns: rows %q, want %q", tc.width, inner, tc.want)
		}
	}
}

func TestReverseDirection(t *testing.T) {
	for _, tc := range []struct {
		name string
		node render.Node
		want []string
	}{
		{"column-reverse packs at the bottom", node(sheet.Upward, text("a"), text("b")), []string{"", "", "b", "a"}},
		{"row-reverse packs at the right", node(sheet.Leftward, text("a"), text("b")), []string{"       b a"}},
	} {
		n, err := render.Scene(tc.node, cssFrame(t, 20))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got := frameRows(n); !slices.Equal(got, tc.want) {
			t.Errorf("%s: rows %q, want %q", tc.name, got, tc.want)
		}
	}
	if _, err := render.Scene(node(sheet.Wrapped, text("a")), cssFrame(t, 20)); err == nil || !strings.Contains(err.Error(), "flex-wrap") {
		t.Errorf("reverse with wrap: error %v, want one naming flex-wrap", err)
	}
	fit, err := render.Scene(node(sheet.Upward, node("w-fit", text("fit"))), cssFrame(t, 20))
	if err != nil {
		t.Fatal(err)
	}
	if w := fit.Children[0].Bounds.W; w != 3 {
		t.Errorf("fit-content child of a column-reverse parent is %d wide, want 3", w)
	}
	var tree render.Tree
	for _, classes := range []string{sheet.Upward, "flex flex-col h-4", sheet.Upward + " " + sheet.Middle, "flex flex-col h-4 " + sheet.Middle} {
		n := node(classes, text("a"), text("b"))
		got, err := tree.Scene(n, cssFrame(t, 20))
		if err != nil {
			t.Fatal(err)
		}
		fresh, err := render.Scene(n, cssFrame(t, 20))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(draw(got), draw(fresh)) {
			t.Errorf("%s after a toggle: retained %q, fresh %q", classes, frameRows(got), frameRows(fresh))
		}
	}
}
