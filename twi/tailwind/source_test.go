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
	write("main.go", "package main\n\nconst classes = \"flex p-4\"\n")
	names, hash, err := Inputs(dir, "twir_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(names, ",") != "main.go" {
		t.Errorf("sources %q, want main.go only", names)
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
