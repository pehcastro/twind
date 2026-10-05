package app

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
)

func parsed(t *testing.T, dir string) []*ast.File {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, e.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	return files
}

func uiComponents(t *testing.T) []string {
	var components []string
	for _, f := range parsed(t, filepath.Join("..", "..", "..", "twi", "ui")) {
		var roots, parts []string
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !fn.Name.IsExported() {
				continue
			}
			if name, ok := strings.CutPrefix(fn.Name.Name, "New"); ok {
				roots = append(roots, name)
				continue
			}
			if r := fn.Type.Results; r != nil && len(r.List) == 1 {
				if sel, ok := r.List[0].Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Node" {
					parts = append(parts, fn.Name.Name)
				}
			}
		}
		slices.SortFunc(parts, func(a, b string) int { return len(a) - len(b) })
		for _, p := range parts {
			if !slices.ContainsFunc(roots, func(r string) bool { return strings.HasPrefix(p, r) }) {
				roots = append(roots, p)
			}
		}
		components = append(components, roots...)
	}
	slices.Sort(components)
	return components
}

func spaced(name string) string {
	var b strings.Builder
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) && unicode.IsLower(rune(name[i-1])) {
			b.WriteByte(' ')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func TestEveryComponentHasAPage(t *testing.T) {
	used := map[string]bool{}
	for _, f := range parsed(t, ".") {
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == "ui" {
					used[strings.TrimPrefix(sel.Sel.Name, "New")] = true
				}
			}
			return true
		})
	}
	components := uiComponents(t)
	if len(components) < 40 {
		t.Fatalf("found %d twi/ui components, want the whole kit: %v", len(components), components)
	}
	for _, c := range components {
		if !slices.ContainsFunc(pages(), func(p page) bool { return p.name == spaced(c) }) {
			t.Errorf("ui.%s has no page named %q", c, spaced(c))
		}
		if !used[c] {
			t.Errorf("ui.%s is never used in the playground", c)
		}
	}
	t.Logf("%d components: %s", len(components), strings.Join(components, ", "))
}

func TestOverlayKeepsGlobalKeys(t *testing.T) {
	d := open(t)
	d.Press("ctrl+k")
	d.Type("alert dialog")
	d.Press("enter")
	d.Advance(time.Second)
	d.Click(spot(t, d, "Show dialog"))
	d.Advance(time.Second)
	press(d, "4", "t", "q")
	if text := d.Frame().Text(); d.Err() != nil || !strings.HasSuffix(status(d), " alert dialog") || !strings.Contains(text, "Are you absolutely sure?") || strings.Contains(text, "Enter keeps") {
		t.Errorf("4, t and q under the alert dialog reached the page, err %v, status %q:\n%s", d.Err(), status(d), text)
	}
}

func TestTourOfComponents(t *testing.T) {
	var renders atomic.Int64
	counted := func(rt *twi.Runtime) func() twi.Node {
		view := App(rt)
		return func() twi.Node {
			renders.Add(1)
			return view()
		}
	}
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	d := drive.New(counted, drive.Size(100, 30), drive.Styles(sheet))
	defer func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}()
	for _, p := range pages() {
		d.Press("ctrl+k")
		d.Type(p.name)
		d.Press("enter")
		d.Advance(time.Second)
		if !strings.HasSuffix(status(d), " "+p.name) {
			t.Errorf("%s: the palette opened another page, status %q:\n%s", p.name, status(d), d.Frame().Text())
		}
		renders.Store(0)
		d.Advance(10 * time.Second)
		if n := renders.Load(); n != 0 {
			t.Errorf("%s: %d frames in 10 s with nothing moving, want 0", p.name, n)
		}
		t.Logf("%s:\n%s", p.name, d.Frame().Text())
	}
	if err := d.Err(); err != nil {
		t.Fatal(err)
	}
}
