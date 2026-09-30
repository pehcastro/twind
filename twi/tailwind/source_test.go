package tailwind

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/style"
)

func TestStale(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	stale := func(want bool, why string) {
		t.Helper()
		got, err := Stale(dir, "twir_gen.go")
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s: stale %v, want %v", why, got, want)
		}
	}
	if _, err := Stale(dir, "twir_gen.go"); err == nil {
		t.Error("no generated file: no error")
	}
	write("go.mod", "module example.com/app\n\ngo 1.26\n")
	app := "package main\n\nimport \"fmt\"\n\nconst classes = \"flex p-4\"\n\nvar raw = `grow\n\tclass=\"hidden\"`\n\nfunc main() { fmt.Println(classes, raw) }\n"
	write("main.go", app)
	candidates, hash, err := Inputs(dir, "twir_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(candidates, " "); got != `class="hidden" flex grow p-4` {
		t.Errorf("candidates %q, want the literals' words sorted, no import path", got)
	}
	write("twir_gen.go", Header(hash)+"package main\n\nconst generated = \"p-2 hover:flex\"\n")
	stale(false, "just generated")
	write("main.go", strings.ReplaceAll(app, "\n", "\r\n"))
	stale(false, "same source with CRLF line ends, a raw string across them")
	write("main.go", strings.Replace(app, "func main() {", "func twice(s string) string { return s + s }\n\nfunc main() {\n\tfmt.Println(twice(raw))\n\t", 1))
	stale(false, "a function added in the package")
	write("main.go", strings.ReplaceAll(strings.Replace(app, "import \"fmt\"", "import (\n\t\"fmt\"\n\t\"os\"\n)", 1), "classes", "names"))
	stale(false, "a constant renamed and an import added")
	write("extra.go", "package main\n\nfunc extra() int { return 1 }\n")
	stale(false, "a source file added without a class")
	write("extra.go", "package main\n\nconst again = \"p-4 flex\"\n")
	stale(false, "classes the package already has written again in another file")
	write("main.go", strings.Replace(app, "flex p-4", "flex p-2", 1))
	stale(true, "a class edited")
	write("main.go", strings.Replace(app, "flex p-4", "flex p-4 gap-1", 1))
	stale(true, "a class added")
	write("extra.go", "package main\n")
	write("main.go", strings.Replace(app, "flex p-4", "flex", 1))
	stale(true, "a class removed")
	write("main.go", app)
	stale(false, "edits reverted")
	write("twir_gen.go", strings.Replace(Header(hash), "version=1", "version=0", 1)+"package main\n")
	stale(true, "an older IR version")
	write("extra.go", "package main\n\nconst broken = \"flex\n")
	if _, err := Stale(dir, "twir_gen.go"); err == nil || !strings.Contains(err.Error(), "extra.go") {
		t.Errorf("an unterminated string: want an error naming extra.go, got %v", err)
	}
}

func TestStaleTakesTestFilesOnlyForATestIR(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	generate := func(ir string) {
		t.Helper()
		_, hash, err := Inputs(dir, ir)
		if err != nil {
			t.Fatal(err)
		}
		write(ir, Header(hash)+"package main\n")
	}
	stale := func(ir string, want bool, why string) {
		t.Helper()
		if got, err := Stale(dir, ir); err != nil || got != want {
			t.Errorf("%s: %s stale %v, error %v, want stale %v", why, ir, got, err, want)
		}
	}
	candidates := func(ir, want string) {
		t.Helper()
		got, _, err := Inputs(dir, ir)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(got, " ") != want {
			t.Errorf("candidates of %s %q, want %q", ir, got, want)
		}
	}
	write("go.mod", "module example.com/app\n\ngo 1.26\n")
	write("main.go", "package main\n\nconst classes = \"flex p-4\"\n")
	write("main_test.go", "package main\n")
	generate("twir_gen.go")
	write("main_test.go", "package main\n\nconst fixture = \"bg-lime-500\"\n")
	candidates("twir_gen.go", "flex p-4")
	stale("twir_gen.go", false, "a class edited in a test beside a package IR")
	generate("twir_gen_test.go")
	stale("twir_gen.go", false, "a second IR written beside it")
	candidates("twir_gen_test.go", "bg-lime-500 flex p-4")
	write("main_test.go", "package main\n\nconst fixture = \"bg-sky-500\"\n")
	stale("twir_gen_test.go", true, "a class edited in a test beside a test IR")
	generate("twir_gen_test.go")
	src, err := os.ReadFile(filepath.Join(dir, "twir_gen_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	write("twir_gen_test.go", strings.ReplaceAll(string(src), "\n", "\r\n"))
	stale("twir_gen_test.go", false, "a CRLF checkout of the IR")
	stale("twir_gen.go", false, "the other IR regenerated")
}

func TestInputsFollowClassPackages(t *testing.T) {
	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	module := func(button, code string) string {
		dir := t.TempDir()
		write(filepath.Join(dir, "go.mod"), "module github.com/twind-dev/twind\n\ngo 1.26\n")
		write(filepath.Join(dir, "twi", "ui", "button.go"), "package ui\n\nconst Destructive = \""+button+"\"\n"+code)
		write(filepath.Join(dir, "twi", "ui", "button_plan9.go"), "package ui\n\nconst plan9 = \"bg-amber-500\"\n")
		write(filepath.Join(dir, "twi", "ui", "button_test.go"), "package ui\n\nconst testOnly = \"bg-lime-500\"\n")
		write(filepath.Join(dir, "twi", "uikit", "kit.go"), "package uikit\n\nconst Kit = \"bg-sky-500\"\n")
		return dir
	}
	app := t.TempDir()
	point := func(twind string) {
		write(filepath.Join(app, "go.mod"), "module example.com/app\n\ngo 1.26\n\nrequire github.com/twind-dev/twind v0.0.0\n\nreplace github.com/twind-dev/twind => "+filepath.ToSlash(twind)+"\n")
	}
	point(module("bg-destructive text-white", ""))
	write(filepath.Join(app, "main.go"), "package main\n\nimport (\n\t\"github.com/twind-dev/twind/twi/ui\"\n\t\"github.com/twind-dev/twind/twi/uikit\"\n)\n\nfunc main() { println(ui.Destructive, uikit.Kit) }\n")
	candidates, hash, err := Inputs(app, "twir_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(candidates, " "); got != "bg-amber-500 bg-destructive text-white" {
		t.Errorf("candidates %q, want twi/ui's two non-test files only, never twi/uikit", got)
	}
	write(filepath.Join(app, "twir_gen.go"), Header(hash)+"package main\n")
	check := func(want bool, why string) {
		t.Helper()
		if got, err := Stale(app, "twir_gen.go"); err != nil || got != want {
			t.Errorf("%s: stale %v, error %v, want stale %v", why, got, err, want)
		}
	}
	check(false, "just generated")
	point(module("bg-destructive text-white", ""))
	check(false, "the same twi/ui at another path")
	crlf := module("bg-destructive text-white", "")
	src, _ := os.ReadFile(filepath.Join(crlf, "twi", "ui", "button.go"))
	write(filepath.Join(crlf, "twi", "ui", "button.go"), strings.ReplaceAll(string(src), "\n", "\r\n"))
	point(crlf)
	check(false, "twi/ui with CRLF line ends")
	point(module("bg-destructive text-white", "\nfunc Pressed(n int) int { return n + 1 }\n"))
	check(false, "a function added in twi/ui")
	point(module("bg-destructive text-black", ""))
	check(true, "a twi/ui class changed")
	point(module("bg-destructive text-white ring-2", ""))
	check(true, "a twi/ui class added")
	point(module("text-white", ""))
	check(true, "a twi/ui class removed")
	write(filepath.Join(app, "go.mod"), "module example.com/app\n\ngo 1.26\n")
	if _, err := Stale(app, "twir_gen.go"); err == nil {
		t.Error("an import go list cannot resolve: no error")
	}
}

func TestStaleWhenCompilerOutputChanges(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{"go.mod": "module example.com/app\n\ngo 1.26\n", "main.go": "package main\n\nconst classes = \"flex p-4\"\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	_, hash, err := Inputs(dir, "twir_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "twir_gen.go"), []byte(Header(hash)+"package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	lostRule := func(src string) ([]style.Rule, []Warning, error) {
		rules, warnings, err := Compile(src)
		return rules[:len(rules)-1], warnings, err
	}
	if got, err := stale(dir, "twir_gen.go", Compile); err != nil || got {
		t.Errorf("an IR from today's compiler: stale %v, error %v, want fresh", got, err)
	}
	if got, err := stale(dir, "twir_gen.go", lostRule); err != nil || !got {
		t.Errorf("a compiler whose output lost a rule: stale %v, error %v, want stale", got, err)
	}
}

func TestInput(t *testing.T) {
	css := Input([]string{`C:\tmp\twirgen\manifest.txt`})
	for _, want := range []string{`@import "tailwindcss" source(none);`, `@source "C:/tmp/twirgen/manifest.txt";`, "--spacing: 1px;"} {
		if !strings.Contains(css, want) {
			t.Errorf("input css lacks %q:\n%s", want, css)
		}
	}
}
