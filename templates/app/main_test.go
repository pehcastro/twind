package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/tailwind"
)

func TestStylesFresh(t *testing.T) {
	stale, err := tailwind.Stale(".", "twir_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("twir_gen.go is stale against the classes in this package: run twind build")
	}
}

func TestDemoScript(t *testing.T) {
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile(filepath.Join("testdata", "demo.twd"))
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := drive.RunScript(strings.NewReader(string(script)), App, out, drive.Styles(sheet)); err != nil {
		t.Fatal(err)
	}
	for frame, want := range map[string]map[string]bool{
		"start":  {"Theme: Light": true, "New project": true, "Dark": false, "Create project": false},
		"picker": {"Theme: Light": true, "Dark": true},
		"dark":   {"Theme: Dark": true, "Light": false},
		"dialog": {"Theme: Dark": true, "Create project": true, "Cancel": true},
	} {
		text, err := os.ReadFile(filepath.Join(out, frame+".txt"))
		if err != nil {
			t.Fatal(err)
		}
		for s, shown := range want {
			if strings.Contains(string(text), s) != shown {
				t.Errorf("frame %s: %q shown is %t, want %t:\n%s", frame, s, !shown, shown, text)
			}
		}
	}
}

func TestQuitsOnQOutsideTheDialog(t *testing.T) {
	d := drive.New(App, drive.Size(120, 34))
	d.Press("tab")
	d.Press("tab")
	d.Press("enter")
	d.Press("q")
	d.Press("escape")
	if err := d.Err(); err != nil {
		t.Fatalf("q in the dialog quit the app: %v", err)
	}
	d.Press("q")
	d.Press("x")
	if d.Err() == nil {
		t.Error("the app still takes keys after q")
	}
}
