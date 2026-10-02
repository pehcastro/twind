package runtime_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/ui"
)

func (r run) read(t *testing.T, f func() string) string {
	t.Helper()
	got := make(chan string, 1)
	r.rt.Dispatch(func() { got <- f() })
	select {
	case s := <-got:
		return s
	case <-time.After(2 * time.Second):
		t.Fatal("the runtime did not run the read within 2s")
		return ""
	}
}

func TestPasteReachesTheFocusedElementFirst(t *testing.T) {
	var heard []string
	listen := func(name string) twi.NodeOption {
		return twi.OnPaste(func(s string) { heard = append(heard, name+" "+s) })
	}
	r := start(static(func() twi.Node {
		return twi.Element(listen("root"),
			twi.Element(twi.Focusable(), twi.AutoFocus(), listen("field"), twi.Text("field")),
			twi.Element(listen("other"), twi.Text("other")))
	}))
	t.Cleanup(func() {
		if err := r.stop(t); err != nil {
			t.Error(err)
		}
	})
	r.next(t)
	r.b.events <- input.PasteEvent{Text: "clip"}
	if got := r.read(t, func() string { return strings.Join(heard, ", ") }); got != "field clip" {
		t.Fatalf("the paste reached %q, want only the focused field", got)
	}
}

func TestPasteIntoATextareaBreaksLinesAndChipsLongText(t *testing.T) {
	var area *ui.Textarea
	r := launch(newBackend(40, 8), func(rt *twi.Runtime) func() twi.Node {
		area = ui.NewTextarea(rt)
		return func() twi.Node { return area.Node(twi.AutoFocus()) }
	})
	t.Cleanup(func() {
		if err := r.stop(t); err != nil {
			t.Error(err)
		}
	})
	r.next(t)
	r.b.events <- input.PasteEvent{Text: "a\r\nb\tc"}
	r.b.events <- input.PasteEvent{Text: strings.Repeat("x", 200)}
	if got := r.read(t, area.Value); got != "a\nb    c[Text 200 characters]" {
		t.Fatalf("the textarea holds %q", got)
	}
}

func TestHotkeyRunsBeforeTheFocusedField(t *testing.T) {
	var hot []string
	var field *ui.Input
	r := start(func(rt *twi.Runtime) func() twi.Node {
		field = ui.NewInput(rt)
		field.Insert("abc")
		return func() twi.Node {
			return twi.Element(twi.OnHotkey(func(k input.KeyEvent) bool {
				if k.Rune == 'k' && k.Modifiers == input.ModCtrl {
					hot = append(hot, "palette")
					return true
				}
				return false
			}), field.Node(twi.AutoFocus()))
		}
	})
	t.Cleanup(func() {
		if err := r.stop(t); err != nil {
			t.Error(err)
		}
	})
	r.next(t)
	for _, k := range []input.KeyEvent{{Key: input.KeyHome}, {Rune: 'k', Modifiers: input.ModCtrl}} {
		r.b.events <- k
	}
	if got := r.read(t, func() string { return strings.Join(hot, ",") + " " + field.Value() }); got != "palette abc" {
		t.Fatalf("hotkey and field read %q, want the hotkey to take ctrl+k before the field", got)
	}
	for _, k := range []input.KeyEvent{{Rune: 'f', Modifiers: input.ModCtrl}, {Rune: 'k', Modifiers: input.ModShift | input.ModCtrl}, {Rune: 'u', Modifiers: input.ModCtrl}} {
		r.b.events <- k
	}
	if got := r.read(t, field.Value); got != "bc" {
		t.Fatalf("the field holds %q: keys the hotkey declines must reach it", got)
	}
	if !slices.Equal(hot, []string{"palette"}) {
		t.Fatalf("the hotkey heard %v", hot)
	}
}
