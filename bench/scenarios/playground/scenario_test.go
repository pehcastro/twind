package playground

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/tailwind"
)

func TestCopyFresh(t *testing.T) {
	for _, name := range []string{"app.go", "pages.go"} {
		source, err := os.ReadFile(filepath.Join("..", "..", "..", "examples", "playground", name))
		if err != nil {
			t.Fatal(err)
		}
		copied, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Replace(string(source), "package main\n", "package playground\n", 1) != string(copied) {
			t.Errorf("%s differs from examples/playground/%s: copy it again with package playground", name, name)
		}
	}
	stale, err := tailwind.Stale(".", "twir_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("twir_gen.go is stale against the classes in this package: run go generate")
	}
}

func TestBenchSteps(t *testing.T) {
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	d := drive.New(App, drive.Size(Columns, Rows), drive.Styles(sheet))
	defer func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}()
	status := func() string {
		all := strings.Split(strings.TrimRight(d.Frame().Text(), "\n"), "\n")
		return strings.Join(strings.Fields(all[len(all)-1]), " ")
	}
	d.Press("a")
	if !strings.Contains(d.Frame().Text(), "▌ a") {
		t.Errorf("a did not reach the focused input:\n%s", d.Frame().Text())
	}
	t.Logf("after a:\n%s", d.Frame().Text())
	d.Press("backspace")
	for i, want := range []string{"slate-light", "zinc-dark", "slate-light"} {
		d.Press("t")
		d.Press("enter")
		d.Press(map[bool]string{true: "down", false: "up"}[i%2 == 0])
		d.Press("enter")
		if got := status(); got != "● fullscreen 120x40 truecolor "+want+" focus input surfaces" {
			t.Errorf("switch %d: status %q, want %s with focus back on the input", i, got, want)
		}
	}
	t.Logf("after three theme switches:\n%s", d.Frame().Text())
	if err := d.Err(); err != nil {
		t.Fatal(err)
	}
}
