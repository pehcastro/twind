package tailwind

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckerListsOnlyWhenTheGraphChanges(t *testing.T) {
	const node = "\n\nimport \"github.com/pehcastro/twind/twi\"\n\nvar _ twi.Node\n"
	app, twind := t.TempDir(), t.TempDir()
	put := func(root, name, body string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write := func(name, body string) { put(app, name, body) }
	put(twind, "go.mod", "module github.com/pehcastro/twind\n\ngo 1.26\n")
	put(twind, "twi/twi.go", "package twi\n\ntype Node struct{}\n")
	write("go.mod", "module example.com/app\n\ngo 1.26\n\nrequire github.com/pehcastro/twind v0.0.0\n\nreplace github.com/pehcastro/twind => "+filepath.ToSlash(twind)+"\n")
	write("main.go", "package main\n\nimport _ \"example.com/app/lib\"\n\nconst own = \"p-4\"\n\nfunc main() {}\n")
	write("lib/lib.go", "package lib"+node+"\nconst Card = \"rounded-lg\"\n")
	write("plain/plain.go", "package plain\n\nconst Words = \"not classes\"\n")
	_, hash, err := Inputs(app, "twir_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	write("twir_gen.go", Header(hash)+"package main\n")
	var c Checker
	check := func(want bool, lists int, why string) {
		t.Helper()
		before, at := c.lists, time.Now()
		got, err := c.Stale(app, "twir_gen.go")
		took := time.Since(at)
		if err != nil || got != want {
			t.Errorf("%s: stale %v, error %v, want %v", why, got, err, want)
		}
		if c.lists-before != lists {
			t.Errorf("%s: %d go list runs, want %d", why, c.lists-before, lists)
		}
		t.Logf("%s: %v", why, took)
	}
	check(false, 1, "first look")
	check(false, 0, "nothing changed")
	write("lib/lib.go", "package lib"+node+"\nconst Card = \"rounded-md\"\n")
	check(true, 0, "a class edited in a library package")
	write("lib/lib.go", "package lib"+node+"\nconst Card = \"rounded-lg\"\n")
	check(false, 0, "the edit reverted")
	write("lib/more.go", "package lib\n\nconst More = \"gap-2\"\n")
	check(true, 1, "a file added to a library package")
	write("lib/more.go", "package lib\n")
	check(false, 0, "the new file emptied")
	write("plain/plain.go", "package plain"+node+"\nconst Words = \"text-red-500\"\n")
	check(false, 0, "a package the app does not import starts to use twi")
	write("main.go", "package main\n\nimport (\n\t_ \"example.com/app/lib\"\n\t_ \"example.com/app/plain\"\n)\n\nconst own = \"p-4\"\n\nfunc main() {}\n")
	check(true, 1, "the app imports it")
}
