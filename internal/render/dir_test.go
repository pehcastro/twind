package render_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/render/testdata/sheet"
	"github.com/pehcastro/twind/twi/buffer"
	"github.com/pehcastro/twind/twi/paint"
	"github.com/pehcastro/twind/twi/style"
)

func directed(dir string, n render.Node) render.Node {
	n.State = &style.NodeState{Attrs: []style.Attr{{Name: "dir", Value: dir}}}
	return n
}

func drawn(t *testing.T, tree *render.Tree, n render.Node) []string {
	t.Helper()
	f := cssFrame(t, 16)
	root, err := tree.Scene(n, f)
	if err != nil {
		t.Fatal(err)
	}
	buf := buffer.New(f.Width, root.Bounds.H)
	paint.Paint(buf, root, paint.Composited)
	return rowsOf(buf)
}

func TestDirection(t *testing.T) {
	const mixed = "שלום world"
	rtl, ltr := "      world םולש", "םולש world"
	for _, c := range []struct {
		name string
		node render.Node
		want []string
	}{
		{"no dir", text(mixed), []string{ltr}},
		{"rtl paragraph", directed("rtl", render.Node{Text: mixed}), []string{rtl}},
		{"rtl inherited by an element and its text", directed("rtl", node("", node("", text(mixed)))), []string{rtl}},
		{"ltr inside rtl", directed("rtl", node("", directed("ltr", node("", text(mixed))))), []string{ltr}},
		{"unknown value inherits", directed("rtl", node("", directed("up", node("", text(mixed))))), []string{rtl}},
		{"explicit text-center stays centred", directed("rtl", node(sheet.PanelHead, text("ab"))), []string{"       ab"}},
		{"flex row starts at the right", directed("rtl", node(sheet.Row, text("a"), text("b"))), []string{"              ba", "", ""}},
		{"justify-end packs to the left", directed("rtl", node(sheet.End, text("a"), text("b"))), []string{"ba"}},
		{"row-reverse reads left to right", directed("rtl", node(sheet.Leftward, text("a"), text("b"))), []string{"a b"}},
	} {
		got := drawn(t, new(render.Tree), c.node)
		t.Logf("%s:\n%s", c.name, strings.Join(got, "\n"))
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: rows %q, want %q", c.name, got, c.want)
		}
	}
}

func TestDirectionRetained(t *testing.T) {
	page := func(dir string) render.Node {
		return directed(dir, node("", node("", text("שלום world"))))
	}
	var tree render.Tree
	for _, step := range []struct {
		dir  string
		want string
	}{
		{"ltr", "םולש world"},
		{"rtl", "      world םולש"},
		{"ltr", "םולש world"},
	} {
		if got := drawn(t, &tree, page(step.dir)); !slices.Equal(got, []string{step.want}) {
			t.Errorf("parent turned %s: rows %q, want %q", step.dir, got, step.want)
		}
	}
}

func TestDirectionKeepsCachedCascadesApart(t *testing.T) {
	got := drawn(t, new(render.Tree), node(sheet.Page, directed("rtl", node(sheet.Row, text("a"), text("b"))), directed("ltr", node(sheet.Row, text("a"), text("b")))))
	want := []string{"              ba", "", "", "ab", "", ""}
	if !slices.Equal(got, want) {
		t.Errorf("rtl and ltr rows with the same classes: %q, want %q", got, want)
	}
}
