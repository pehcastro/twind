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
	write("main.go", "package main\n\nconst classes = \"flex p-4\"\n")
	sources, hash, err := Inputs(dir, "twir_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sources, ",") != filepath.Join(dir, "main.go") {
		t.Errorf("sources %q, want main.go only", sources)
	}
	write("twir_gen.go", Header(hash)+"package main\n\nconst generated = \"p-2 hover:flex\"\n")
	stale(false, "just generated")
	write("main.go", "package main\r\n\r\nconst classes = \"flex p-4\"\r\n")
	stale(false, "same source with CRLF line ends")
	write("main.go", "package main\n\nconst classes = \"flex p-2\"\n")
	stale(true, "a class edited")
	write("main.go", "package main\n\nconst classes = \"flex p-4\"\n")
	stale(false, "edit reverted")
	write("extra.go", "package main\n")
	stale(true, "a source file added")
	_, hash, _ = Inputs(dir, "twir_gen.go")
	write("twir_gen.go", strings.Replace(Header(hash), "version=1", "version=0", 1)+"package main\n")
	stale(true, "an older IR version")
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
	write("go.mod", "module example.com/app\n\ngo 1.26\n")
	write("main.go", "package main\n\nconst classes = \"flex p-4\"\n")
	write("main_test.go", "package main\n")
	generate("twir_gen.go")
	sources, _, err := Inputs(dir, "twir_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sources, ",") != filepath.Join(dir, "main.go") {
		t.Errorf("sources of a package IR %q, want main.go only", sources)
	}
	write("main_test.go", "package main\n\nconst fixture = \"bg-lime-500\"\n")
	stale("twir_gen.go", false, "a class edited in a test beside a package IR")
	generate("twir_gen_test.go")
	stale("twir_gen.go", false, "a second IR written beside it")
	sources, _, err = Inputs(dir, "twir_gen_test.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sources, ",") != filepath.Join(dir, "main.go")+","+filepath.Join(dir, "main_test.go") {
		t.Errorf("sources of a test IR %q, want main.go and main_test.go", sources)
	}
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
	module := func(button string) string {
		dir := t.TempDir()
		write(filepath.Join(dir, "go.mod"), "module github.com/twind-dev/twind\n\ngo 1.26\n")
		write(filepath.Join(dir, "twi", "ui", "button.go"), "package ui\n\nconst Destructive = \""+button+"\"\n")
		write(filepath.Join(dir, "twi", "ui", "button_plan9.go"), "package ui\n\nconst plan9 = \"bg-amber-500\"\n")
		write(filepath.Join(dir, "twi", "ui", "button_test.go"), "package ui\n\nconst testOnly = \"bg-lime-500\"\n")
		write(filepath.Join(dir, "twi", "uikit", "kit.go"), "package uikit\n\nconst Kit = \"bg-sky-500\"\n")
		return dir
	}
	app := t.TempDir()
	point := func(twind string) {
		write(filepath.Join(app, "go.mod"), "module example.com/app\n\ngo 1.26\n\nrequire github.com/twind-dev/twind v0.0.0\n\nreplace github.com/twind-dev/twind => "+filepath.ToSlash(twind)+"\n")
	}
	twind := module("bg-destructive text-white")
	point(twind)
	write(filepath.Join(app, "main.go"), "package main\n\nimport (\n\t\"github.com/twind-dev/twind/twi/ui\"\n\t\"github.com/twind-dev/twind/twi/uikit\"\n)\n\nfunc main() { println(ui.Destructive, uikit.Kit) }\n")
	sources, hash, err := Inputs(app, "twir_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, s := range sources {
		names = append(names, filepath.Base(filepath.Dir(s))+"/"+filepath.Base(s))
	}
	if got := strings.Join(names, ","); got != filepath.Base(app)+"/main.go,ui/button.go,ui/button_plan9.go" {
		t.Errorf("sources %s, want the app's main.go and twi/ui's two non-test files, never twi/uikit", got)
	}
	write(filepath.Join(app, "twir_gen.go"), Header(hash)+"package main\n")
	check := func(want bool, why string) {
		t.Helper()
		if got, err := Stale(app, "twir_gen.go"); err != nil || got != want {
			t.Errorf("%s: stale %v, error %v, want stale %v", why, got, err, want)
		}
	}
	check(false, "just generated")
	point(module("bg-destructive text-white"))
	check(false, "the same twi/ui at another path")
	crlf := module("bg-destructive text-white")
	src, _ := os.ReadFile(filepath.Join(crlf, "twi", "ui", "button.go"))
	write(filepath.Join(crlf, "twi", "ui", "button.go"), strings.ReplaceAll(string(src), "\n", "\r\n"))
	point(crlf)
	check(false, "twi/ui with CRLF line ends")
	point(module("bg-destructive text-black"))
	check(true, "a twi/ui class changed")
	write(filepath.Join(app, "go.mod"), "module example.com/app\n\ngo 1.26\n")
	if _, err := Stale(app, "twir_gen.go"); err == nil {
		t.Error("an import go list cannot resolve: no error")
	}
}

func TestStaleWhenCompilerOutputChanges(t *testing.T) {
	fixture := filepath.Join("..", "..", "internal", "twirgen", "testdata", "hello")
	lostRule := func(src string) ([]style.Rule, []Warning, error) {
		rules, warnings, err := Compile(src)
		return rules[:len(rules)-1], warnings, err
	}
	if got, err := stale(fixture, "twir_gen.go", Compile); err != nil || got {
		t.Errorf("committed fixture with today's compiler: stale %v, error %v, want fresh", got, err)
	}
	if got, err := stale(fixture, "twir_gen.go", lostRule); err != nil || !got {
		t.Errorf("committed fixture with a compiler whose output lost a rule: stale %v, error %v, want stale", got, err)
	}
}

func TestInput(t *testing.T) {
	css := Input([]string{`C:\src\app\main.go`})
	for _, want := range []string{`@import "tailwindcss" source(none);`, `@source "C:/src/app/main.go";`, "--spacing: 1px;"} {
		if !strings.Contains(css, want) {
			t.Errorf("input css lacks %q:\n%s", want, css)
		}
	}
}
