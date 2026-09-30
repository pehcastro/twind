package runtime_test

import (
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
)

func TestKeyClickActivatesTheFocusedElement(t *testing.T) {
	var got []string
	record := func(name string) twi.NodeOption { return twi.OnClick(func(*twi.Event) { got = append(got, name) }) }
	disabled := false
	app := func(rt *twi.Runtime) func() twi.Node {
		return func() twi.Node {
			button := []twi.NodeOption{twi.Key("button"), twi.Focusable(), twi.AutoFocus(), record("button"), twi.Text("button"),
				twi.OnKeyDown(func(e *twi.Event) {
					if e.Key.Rune == 'd' {
						disabled = true
						e.PreventDefault()
						rt.Invalidate()
					}
				})}
			if disabled {
				button = append(button, twi.Disabled())
			}
			return twi.Element(record("root"),
				twi.Element(button...),
				twi.Element(twi.Key("field"), twi.Focusable(), record("field"), twi.Text("field"),
					twi.OnKeyDown(func(e *twi.Event) {
						if e.Key.Key == input.KeyEnter {
							got = append(got, "field key")
							e.PreventDefault()
						}
					})),
				twi.Element(twi.Key("bare"), twi.Focusable(), twi.Text("bare")),
			)
		}
	}
	d := drive.New(app, drive.Size(20, 4))
	defer func() {
		if err := d.Err(); err != nil {
			t.Error(err)
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}()
	for _, step := range []struct{ keys, want string }{
		{"enter", "button, root"},
		{"space", "button, root"},
		{"shift+enter alt+space ctrl+space", ""},
		{"tab enter", "field key"},
		{"space", "field, root"},
		{"tab enter space", ""},
		{"shift+tab shift+tab d enter space", ""},
	} {
		got = nil
		for k := range strings.FieldsSeq(step.keys) {
			d.Press(k)
		}
		if strings.Join(got, ", ") != step.want {
			t.Errorf("%s: clicks %q, want %q", step.keys, strings.Join(got, ", "), step.want)
		}
	}
}
