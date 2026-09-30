package edit_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
)

type driveStep struct {
	typed string
	key   string
	want  string
}

func driveInput(t *testing.T, steps []driveStep) {
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		in := twi.NewInput(rt)
		return func() twi.Node {
			start, end := in.Selection()
			v := in.Value()
			return twi.Element(in.Node(twi.AutoFocus()), twi.Text("value "+v[:start]+"["+v[start:end]+"]"+v[end:]))
		}
	}, drive.Size(24, 4))
	defer func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}()
	for _, s := range steps {
		d.Type(s.typed)
		if s.key != "" {
			d.Press(s.key)
		}
		frame := d.Frame().Text()
		t.Logf("after %q %s:\n%s", s.typed, s.key, frame)
		if !slices.Contains(strings.Split(frame, "\n"), s.want) {
			t.Fatalf("after %q %s: no %q", s.typed, s.key, s.want)
		}
	}
	if err := d.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestUndoDrivenTypeOver(t *testing.T) {
	driveInput(t, []driveStep{
		{"hello world", "", "value hello world[]"},
		{"", "ctrl+left", "value hello []world"},
		{"", "shift+end", "value hello [world]"},
		{"there", "", "value hello there[]"},
		{"", "ctrl+z", "value hello [world]"},
		{"", "ctrl+z", "value []"},
		{"", "ctrl+shift+z", "value hello [world]"},
	})
}

func TestWordDriven(t *testing.T) {
	driveInput(t, []driveStep{
		{"don't stop 3.14", "", "value don't stop 3.14[]"},
		{"", "ctrl+left", "value don't stop []3.14"},
		{"", "ctrl+left", "value don't []stop 3.14"},
		{"", "ctrl+left", "value []don't stop 3.14"},
		{"", "ctrl+delete", "value [] stop 3.14"},
		{"", "end", "value stop 3.14[]"},
		{"", "ctrl+backspace", "value stop []"},
	})
}
