package docsapp

import (
	"cmp"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const module = "github.com/pehcastro/twind/"

func TestReference(t *testing.T) {
	write := os.Getenv("TWIND_WRITE_REFERENCE") == "1"
	for _, pkg := range []string{"twi", "twi/ui", "twi/drive", "twi/theme", "twi/markdown", "twi/chart", "twi/fix"} {
		want := reference(t, pkg)
		file := filepath.Join("..", "..", "docs", "api-"+path.Base(pkg)+".md")
		if write {
			if err := os.WriteFile(file, []byte(want), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		got := strings.ReplaceAll(string(src), "\r\n", "\n")
		if got == want {
			continue
		}
		wantLines, gotLines := strings.Split(want, "\n"), strings.Split(got, "\n")
		for _, line := range wantLines {
			if !slices.Contains(gotLines, line) {
				t.Errorf("%s: no line %s", file, line)
			}
		}
		for _, line := range gotLines {
			if !slices.Contains(wantLines, line) {
				t.Errorf("%s: %s is not in the source of %s", file, line, pkg)
			}
		}
		t.Errorf("%s differs from the source of %s: run TWIND_WRITE_REFERENCE=1 go test ./apps/documentation -run TestReference", file, pkg)
	}
}

func TestReferenceOutlineScrolls(t *testing.T) {
	d := open(t)
	jump(t, d, "twi/ui")
	x, y := spot(t, d, "On this page")
	for range 10 {
		d.Wheel(x, y+2, 1)
	}
	d.Click(spot(t, d, "Toggle"))
	d.Advance(settle)
	t.Logf("twi/ui, the outline wheeled to its end and Toggle clicked:\n%s", d.Frame().Text())
	if _, row := spot(t, d, "type Toggle struct"); row > 6 {
		t.Errorf("type Toggle struct on row %d, want it at the top of the page", row)
	}
}

func reference(t *testing.T, root string) string {
	fset := token.NewFileSet()
	imports := map[string]string{}
	load := func(pkg string) *doc.Package {
		names, err := filepath.Glob(filepath.Join("..", "..", filepath.FromSlash(pkg), "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		var files []*ast.File
		for _, name := range names {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			f, err := parser.ParseFile(fset, name, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, imp := range f.Imports {
				local, _ := strconv.Unquote(imp.Path.Value)
				if rest, ok := strings.CutPrefix(local, module); ok && imp.Name == nil {
					imports[path.Base(rest)] = rest
				}
			}
			files = append(files, f)
		}
		p, err := doc.NewFromFiles(fset, files, module+pkg, doc.AllDecls)
		if err != nil {
			t.Fatal(err)
		}
		if pkg != root {
			for _, f := range files {
				ast.Inspect(f, qualify(path.Base(pkg)))
			}
		}
		imports[path.Base(pkg)] = pkg
		return p
	}
	show := func(n any) string {
		var b strings.Builder
		if err := printer.Fprint(&b, fset, n); err != nil {
			t.Fatal(err)
		}
		return strings.Join(strings.Fields(b.String()), " ")
	}
	values := func(tok string, vs []*doc.Value) []string {
		var lines []string
		for _, v := range vs {
			var typ ast.Expr
			for _, spec := range v.Decl.Specs {
				s := spec.(*ast.ValueSpec)
				if s.Type != nil || len(s.Values) > 0 {
					typ = s.Type
				}
				for i, n := range s.Names {
					switch {
					case !n.IsExported():
					case typ != nil:
						lines = append(lines, tok+" "+n.Name+" "+show(typ))
					case i < len(s.Values):
						lines = append(lines, tok+" "+n.Name+" = "+show(s.Values[i]))
					default:
						lines = append(lines, tok+" "+n.Name)
					}
				}
			}
		}
		return lines
	}
	funcs := func(fs []*doc.Func, recv string) []string {
		var lines []string
		for _, f := range fs {
			if !ast.IsExported(f.Name) {
				continue
			}
			d := &ast.FuncDecl{Name: f.Decl.Name, Type: f.Decl.Type}
			if f.Recv != "" {
				d.Recv = &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent(cmp.Or(recv, f.Recv))}}}
			}
			lines = append(lines, show(d))
		}
		return lines
	}
	var members func(p *doc.Package, ty *doc.Type, owner string) []string
	members = func(p *doc.Package, ty *doc.Type, owner string) []string {
		var fields []*ast.Field
		switch kind := ty.Decl.Specs[0].(*ast.TypeSpec).Type.(type) {
		case *ast.StructType:
			fields = kind.Fields.List
		case *ast.InterfaceType:
			fields = kind.Methods.List
		}
		var lines []string
		var promoted [][]string
		for _, m := range fields {
			for _, n := range m.Names {
				if n.IsExported() {
					lines = append(lines, owner+"."+n.Name+" "+show(m.Type))
				}
			}
			if len(m.Names) > 0 {
				continue
			}
			name := strings.TrimPrefix(show(m.Type), "*")
			from, typ, foreign := strings.Cut(name, ".")
			if ast.IsExported(typ) || !foreign && ast.IsExported(name) {
				lines = append(lines, owner+" embeds "+show(m.Type))
			}
			inner := p
			switch {
			case foreign && imports[from] == "":
				continue
			case foreign:
				inner = load(imports[from])
			default:
				typ = name
			}
			if i := slices.IndexFunc(inner.Types, func(ty *doc.Type) bool { return ty.Name == typ }); i >= 0 {
				promoted = append(promoted, members(inner, inner.Types[i], owner))
			}
		}
		recv := "*" + owner
		if ty.Name == owner {
			recv = ""
		}
		lines = append(lines, funcs(ty.Methods, recv)...)
		for _, line := range slices.Concat(promoted...) {
			if !slices.ContainsFunc(lines, func(l string) bool { return member(l) == member(line) }) {
				lines = append(lines, line)
			}
		}
		return lines
	}
	type part struct {
		at    token.Position
		name  string
		lines []string
	}
	p := load(root)
	var parts []part
	for _, v := range slices.Concat(p.Consts, p.Vars) {
		parts = append(parts, part{fset.Position(v.Decl.Pos()), "", values(v.Decl.Tok.String(), []*doc.Value{v})})
	}
	for _, f := range p.Funcs {
		parts = append(parts, part{fset.Position(f.Decl.Pos()), f.Name, funcs([]*doc.Func{f}, "")})
	}
	for _, ty := range p.Types {
		spec := ty.Decl.Specs[0].(*ast.TypeSpec)
		head := &ast.TypeSpec{Name: spec.Name, TypeParams: spec.TypeParams, Assign: spec.Assign, Type: spec.Type}
		switch spec.Type.(type) {
		case *ast.StructType:
			head.Type = ast.NewIdent("struct")
		case *ast.InterfaceType:
			head.Type = ast.NewIdent("interface")
		}
		lines := slices.Concat(values("const", ty.Consts), values("var", ty.Vars), funcs(ty.Funcs, ""))
		if ast.IsExported(ty.Name) {
			lines = slices.Concat([]string{"type " + show(head)}, lines, members(p, ty, ty.Name))
		}
		parts = append(parts, part{fset.Position(spec.Pos()), ty.Name, lines})
	}
	parts = slices.DeleteFunc(parts, func(p part) bool { return len(p.lines) == 0 })
	slices.SortFunc(parts, func(a, b part) int {
		return cmp.Or(strings.Compare(filepath.Base(a.at.Filename), filepath.Base(b.at.Filename)), cmp.Compare(a.at.Offset, b.at.Offset))
	})
	var b strings.Builder
	b.WriteString("# " + root + "\n\nEvery exported name in `" + root + "`, one line each, grouped by the file that declares it. A test reads them from the Go source and fails when this page drifts from it.")
	escape := strings.NewReplacer(`\`, `\\`, "`", "\\`", "*", `\*`, "_", `\_`, "[", `\[`, "]", `\]`, "<", `\<`)
	for i, pt := range parts {
		file := filepath.Base(pt.at.Filename)
		next := "\\\n"
		if i == 0 || file != filepath.Base(parts[i-1].at.Filename) {
			heading := file
			for _, other := range parts {
				if filepath.Base(other.at.Filename) == file && strings.EqualFold(other.name, strings.TrimSuffix(file, ".go")) {
					heading = other.name
				}
			}
			next = "\n\n## " + heading + "\n\n"
		}
		for _, line := range pt.lines {
			b.WriteString(next + escape.Replace(line))
			next = "\\\n"
		}
	}
	return b.String() + "\n"
}

func member(line string) string {
	if rest, ok := strings.CutPrefix(line, "func ("); ok {
		_, line, _ = strings.Cut(rest, ") ")
	} else {
		_, line, _ = strings.Cut(line, ".")
	}
	name, _, _ := strings.Cut(line, "(")
	name, _, _ = strings.Cut(name, " ")
	return name
}

func qualify(pkg string) func(ast.Node) bool {
	var types func(ast.Node) bool
	types = func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			return false
		case *ast.Field:
			ast.Inspect(n.Type, types)
			return false
		case *ast.Ident:
			if n.IsExported() {
				n.Name = pkg + "." + n.Name
			}
		}
		return true
	}
	return func(n ast.Node) bool {
		if f, ok := n.(*ast.Field); ok {
			ast.Inspect(f.Type, types)
			return false
		}
		return true
	}
}
