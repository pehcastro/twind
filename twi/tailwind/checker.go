package tailwind

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/style"
)

type Checker struct {
	compile compileFunc
	prefix  []byte
	files   map[string]sourceFile
	graphs  map[string]graph
	lists   int
}

type sourceFile struct {
	sum   [sha256.Size]byte
	head  string
	words []string
}

type graph struct {
	pkgs []listed
	mods map[string][sha256.Size]byte
}

type listed struct {
	dir   string
	twi   bool
	live  bool
	files []string
	heads map[string]string
}

func (c *Checker) Stale(dir, generated string) (bool, error) {
	src, err := os.ReadFile(filepath.Join(dir, generated))
	if err != nil {
		return false, err
	}
	_, hash, err := c.Inputs(dir, generated)
	if err != nil {
		return false, err
	}
	recorded, _, _ := strings.Cut(string(src), "\n")
	current, _, _ := strings.Cut(Header(hash), "\n")
	return strings.TrimSuffix(recorded, "\r") != current, nil
}

func (c *Checker) Inputs(dir, generated string) ([]string, string, error) {
	if c.prefix == nil {
		compile := c.compile
		if compile == nil {
			compile = Compile
		}
		rules, _, err := compile(strings.ReplaceAll(compilerCorpus, "\r\n", "\n"))
		if err != nil {
			return nil, "", err
		}
		c.prefix = fmt.Appendf(nil, "%s%s%s%#v", Header(""), konst.PresetTheme, twAnimate, rules)
		c.files, c.graphs = map[string]sourceFile{}, map[string]graph{}
	}
	g, err := c.graph(dir)
	if err != nil {
		return nil, "", err
	}
	words := map[string]bool{}
	add := func(path string) error {
		f, err := c.file(path)
		for _, w := range f.words {
			words[w] = true
		}
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", err
	}
	testIR := strings.HasSuffix(generated, "_test.go")
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == generated || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") && !testIR {
			continue
		}
		if err := add(filepath.Join(dir, name)); err != nil {
			return nil, "", err
		}
	}
	for _, p := range g.pkgs {
		if !p.twi {
			continue
		}
		for _, name := range p.files {
			if err := add(filepath.Join(p.dir, name)); err != nil {
				return nil, "", err
			}
		}
	}
	candidates := slices.Sorted(maps.Keys(words))
	h := sha256.New()
	h.Write(c.prefix)
	h.Write([]byte(strings.Join(candidates, "\n")))
	return candidates, hex.EncodeToString(h.Sum(nil)), nil
}

func (c *Checker) graph(dir string) (graph, error) {
	if g, ok := c.graphs[dir]; ok && c.current(g) {
		return g, nil
	}
	c.lists++
	out, err := exec.Command("go", "list", "-C", dir, "-deps", "-f", `{{if not .Standard}}{{.Dir}}{{"\t"}}{{if .DepOnly}}{{range .Imports}}{{if eq . "github.com/twind-dev/twind/twi"}}twi{{end}}{{end}}{{end}}{{"\t"}}{{with .Module}}{{if or .Main .Replace}}live{{end}}{{"\t"}}{{.GoMod}}{{else}}live{{"\t"}}{{end}}{{range .GoFiles}}{{"\t"}}{{.}}{{end}}{{range .IgnoredGoFiles}}{{"\t"}}{{.}}{{end}}{{"\n"}}{{end}}`, ".").CombinedOutput()
	if err != nil {
		return graph{}, fmt.Errorf("go list: %w\n%s", err, out)
	}
	g := graph{mods: map[string][sha256.Size]byte{}}
	for line := range strings.Lines(string(out)) {
		fields := strings.Split(strings.TrimRight(line, "\r\n"), "\t")
		p := listed{dir: fields[0], twi: fields[1] == "twi", live: fields[2] == "live"}
		for _, name := range fields[4:] {
			if !strings.HasSuffix(name, "_test.go") {
				p.files = append(p.files, name)
			}
		}
		if p.live {
			if fields[3] != "" {
				src, err := os.ReadFile(fields[3])
				if err != nil {
					return graph{}, err
				}
				g.mods[fields[3]] = sha256.Sum256(src)
			}
			if p.heads, err = c.heads(p.dir); err != nil {
				return graph{}, err
			}
		}
		g.pkgs = append(g.pkgs, p)
	}
	c.graphs[dir] = g
	return g, nil
}

func (c *Checker) current(g graph) bool {
	for path, sum := range g.mods {
		if src, err := os.ReadFile(path); err != nil || sha256.Sum256(src) != sum {
			return false
		}
	}
	for _, p := range g.pkgs {
		if !p.live {
			continue
		}
		if heads, err := c.heads(p.dir); err != nil || !maps.Equal(heads, p.heads) {
			return false
		}
	}
	return true
}

func (c *Checker) heads(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	heads := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := c.file(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		heads[name] = f.head
	}
	return heads, nil
}

func (c *Checker) file(path string) (sourceFile, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return sourceFile{}, err
	}
	sum := sha256.Sum256(src)
	if f, ok := c.files[path]; ok && f.sum == sum {
		return f, nil
	}
	f := sourceFile{sum: sum}
	if bytes.HasPrefix(src, []byte("// "+konst.IRMagic+" ")) {
		c.files[path] = f
		return f, nil
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, src, parser.SkipObjectResolution)
	if err != nil {
		return sourceFile{}, err
	}
	var head strings.Builder
	head.Write(src[:file.Package-1])
	words := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ImportSpec:
			head.WriteString("\n" + n.Path.Value)
			return false
		case *ast.BasicLit:
			if n.Kind != token.STRING {
				return true
			}
			text, _ := strconv.Unquote(n.Value)
			for _, word := range strings.Fields(text) {
				words[word] = true
			}
		}
		return true
	})
	f.head, f.words = head.String(), slices.Sorted(maps.Keys(words))
	c.files[path] = f
	return f, nil
}
