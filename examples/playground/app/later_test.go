package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
)

func TestLaterScript(t *testing.T) {
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join("testdata", "later.twd"))
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := drive.RunScript(bytes.NewReader(src), App, out, drive.With(twi.Styles(sheet))); err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		name  string
		shown bool
	}{{"1-at-699ms", false}, {"2-at-700ms", true}, {"3-at-3699ms", true}, {"4-at-3700ms", false}} {
		frame, err := os.ReadFile(filepath.Join(out, step.name+".txt"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(frame), "shown 700 ms after Enter") != step.shown {
			t.Errorf("%s: card shown %v, want %v:\n%s", step.name, !step.shown, step.shown, frame)
		}
		t.Logf("%s:\n%s", step.name, frame)
	}
}
