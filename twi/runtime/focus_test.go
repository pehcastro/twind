package runtime_test

import (
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
)

type focusApp struct {
	log                           []string
	before, open, gone, autofocus bool
	focused                       *twi.Signal[string]
}

func (a *focusApp) record(entry string) { a.log = append(a.log, entry) }

func (a *focusApp) heard(who string, k input.KeyEvent) {
	switch k.Key {
	case input.KeyRune:
		a.record(who + " " + string(k.Rune))
	case input.KeyEnter:
		a.record(who + " enter")
	case input.KeyEscape:
		a.record(who + " escape")
	}
}

func (a *focusApp) take() string {
	out := strings.Join(a.log, ", ")
	a.log = nil
	return out
}

func (a *focusApp) button(name string, opts ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{
		twi.Key(name),
		twi.Focusable(),
		twi.OnFocus(func() {
			a.record("focus " + name)
			a.focused.Set(name)
		}),
		twi.OnBlur(func() { a.record("blur " + name) }),
		twi.OnKeyDown(func(e *twi.Event) { a.heard(name, e.Key) }),
		twi.Text(name),
	}, opts...)...)
}

func (a *focusApp) start(t *testing.T) *drive.Driver {
	t.Helper()
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		in := twi.NewInput(rt)
		redraw := twi.NewSignal(rt, 0)
		a.focused = twi.NewSignal(rt, "")
		return func() twi.Node {
			root := []twi.NodeOption{
				twi.Text("focused " + a.focused.Get()),
				twi.OnKeyDown(func(e *twi.Event) {
					a.heard("root", e.Key)
					switch {
					case e.Key.Rune == 'i':
						a.before = true
					case e.Key.Rune == 'r':
						a.gone = true
					case e.Key.Rune == 'f':
						a.autofocus = true
					case e.Key.Key == input.KeyEnter:
						a.open = true
					case e.Key.Key == input.KeyEscape:
						a.open = false
					}
					redraw.Set(redraw.Get() + 1)
				}),
				twi.OnKey(func(k input.KeyEvent) { a.heard("document", k) }),
				in.Node(twi.AutoFocus(), twi.Key("input")),
			}
			if a.before {
				root = append(root, a.button("c"))
			}
			root = append(root, a.button("a"), a.button("d", twi.Disabled()))
			if !a.gone {
				root = append(root, a.button("b"))
			}
			if a.autofocus {
				root = append(root, a.button("late", twi.AutoFocus()))
			}
			if a.open {
				root = append(root, twi.Element(twi.Key("dialog"), twi.FocusScope(), a.button("one"), a.button("two")))
			}
			return twi.Element(root...)
		}
	}, drive.Size(40, 12))
	t.Cleanup(func() {
		if err := d.Err(); err != nil {
			t.Error(err)
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	return d
}

func expect(t *testing.T, a *focusApp, step, want string) {
	t.Helper()
	if got := a.take(); got != want {
		t.Errorf("%s: got %q, want %q", step, got, want)
	}
}

func TestFocusInputKeepsItsKeys(t *testing.T) {
	a := &focusApp{}
	d := a.start(t)
	d.Type("q")
	expect(t, a, "q in the autofocused input", "")
	if text := d.Frame().Text(); !strings.Contains(text, "\nq\n") {
		t.Errorf("the input does not show q:\n%s", text)
	}
	d.Press("tab")
	d.Press("x")
	expect(t, a, "tab then x", "focus a, a x, root x, document x")
	d.Press("tab")
	d.Press("x")
	expect(t, a, "tab skips the disabled d", "blur a, focus b, b x, root x, document x")
	d.Press("shift+tab")
	d.Press("shift+tab")
	d.Type("z")
	expect(t, a, "back in the input", "blur b, focus a, blur a")
	if text := d.Frame().Text(); !strings.Contains(text, "\nqz\n") {
		t.Errorf("the input lost its value or its focus:\n%s", text)
	}
}

func TestFocusFollowsIdentity(t *testing.T) {
	a := &focusApp{}
	d := a.start(t)
	d.Press("tab")
	d.Press("tab")
	d.Press("i")
	expect(t, a, "focus b, insert c before it", "focus a, blur a, focus b, b i, root i, document i")
	d.Press("f")
	d.Press("x")
	expect(t, a, "a later autofocus node", "b f, root f, document f, b x, root x, document x")
	d.Press("r")
	d.Press("x")
	expect(t, a, "b removed while focused", "b r, root r, document r, root x, document x")
}

func TestFocusScopeTrapsAndRestores(t *testing.T) {
	a := &focusApp{}
	d := a.start(t)
	d.Press("tab")
	d.Press("enter")
	expect(t, a, "enter opens the dialog", "focus a, a enter, root enter, document enter, blur a, focus one")
	if text := d.Frame().Text(); !strings.HasPrefix(text, "focused one") {
		t.Errorf("the frame that opens the dialog does not show the focus its opening set:\n%s", text)
	}
	d.Press("tab")
	d.Press("x")
	d.Press("tab")
	d.Press("tab")
	expect(t, a, "tab stays inside across redraws", "blur one, focus two, two x, root x, document x, blur two, focus one, blur one, focus two")
	d.Press("escape")
	d.Press("x")
	expect(t, a, "escape closes and restores a", "two escape, root escape, document escape, blur two, focus a, a x, root x, document x")
}
