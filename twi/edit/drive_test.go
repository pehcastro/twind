package edit_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
)

func TestUndoDrivenTypeOver(t *testing.T) {
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
	steps := []struct {
		do   func()
		name string
		want string
	}{
		{func() { d.Type("hello world") }, "type", "value hello world[]"},
		{func() { d.Press("ctrl+left") }, "ctrl+left", "value hello []world"},
		{func() { d.Press("shift+end") }, "shift+end", "value hello [world]"},
		{func() { d.Type("there") }, "type over", "value hello there[]"},
		{func() { d.Press("ctrl+z") }, "ctrl+z", "value hello [world]"},
		{func() { d.Press("ctrl+z") }, "ctrl+z", "value []"},
		{func() { d.Press("ctrl+shift+z") }, "ctrl+shift+z", "value hello [world]"},
	}
	for _, s := range steps {
		s.do()
		frame := d.Frame().Text()
		t.Logf("after %s:\n%s", s.name, frame)
		if !slices.Contains(strings.Split(frame, "\n"), s.want) {
			t.Fatalf("after %s: no %q", s.name, s.want)
		}
	}
	if err := d.Err(); err != nil {
		t.Fatal(err)
	}
}
