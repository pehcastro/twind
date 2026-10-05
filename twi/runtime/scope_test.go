package runtime_test

import (
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/input"
)

func (a *focusApp) popover(t *testing.T, modal bool) *drive.Driver {
	t.Helper()
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		redraw := twi.NewSignal(rt, 0)
		a.focused = twi.NewSignal(rt, "")
		set := func(open bool) {
			a.open = open
			redraw.Set(redraw.Get() + 1)
		}
		return func() twi.Node {
			anchor := []twi.NodeOption{twi.Key("anchor"), a.button("trigger", twi.OnKeyDown(func(e *twi.Event) {
				if e.Key.Key == input.KeyEnter {
					set(true)
				}
			}))}
			if a.open {
				content := []twi.NodeOption{
					twi.Key("popover"),
					twi.NonModalFocusScope(),
					twi.OnFocusOutside(func() {
						a.record("outside")
						set(false)
					}),
					twi.OnKeyDown(func(e *twi.Event) {
						if e.Key.Key == input.KeyEscape {
							set(false)
							e.StopPropagation()
						}
					}),
				}
				if a.gone {
					content = append(content, twi.Text("nothing to focus"))
				} else {
					content = append(content, a.button("one"), a.button("two"))
				}
				anchor = append(anchor, twi.Element(content...))
			}
			body := []twi.NodeOption{twi.Text("focused " + a.focused.Get()), a.button("before"), twi.Element(anchor...), a.button("after")}
			if modal {
				return twi.Element(twi.Element(append([]twi.NodeOption{twi.Key("dialog"), twi.FocusScope()}, body...)...), a.button("beyond"))
			}
			return twi.Element(body...)
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

func TestNonModalScopeTabLeaves(t *testing.T) {
	a := &focusApp{}
	d := a.popover(t, false)
	d.Press("tab")
	d.Press("tab")
	a.take()
	d.Press("enter")
	expect(t, a, "enter opens and focus enters", "trigger enter, blur trigger, focus one")
	d.Press("tab")
	expect(t, a, "tab inside", "blur one, focus two")
	d.Press("tab")
	expect(t, a, "tab leaves and calls the leave callback", "blur two, focus after, outside")
	d.Press("x")
	expect(t, a, "focus stays where tab put it", "after x")
	if text := d.Frame().Text(); strings.Contains(text, "one") || !strings.HasPrefix(text, "focused after") {
		t.Errorf("the popover is still open or focus moved:\n%s", text)
	}
}

func TestNonModalScopeEscapeReturnsFocus(t *testing.T) {
	a := &focusApp{}
	d := a.popover(t, false)
	d.Press("tab")
	d.Press("tab")
	d.Press("enter")
	a.take()
	d.Press("escape")
	d.Press("x")
	expect(t, a, "escape closes and returns focus", "one escape, blur one, focus trigger, trigger x")
	d.Press("enter")
	d.Press("shift+tab")
	expect(t, a, "shift+tab to the trigger leaves", "trigger enter, blur trigger, focus one, blur one, focus trigger, outside")
}

func TestNonModalScopeWithNothingFocusable(t *testing.T) {
	a := &focusApp{gone: true}
	d := a.popover(t, false)
	d.Press("tab")
	d.Press("tab")
	a.take()
	d.Press("enter")
	expect(t, a, "opening keeps focus on the trigger", "trigger enter")
	if text := d.Frame().Text(); !strings.Contains(text, "nothing to focus") {
		t.Errorf("the popover did not open:\n%s", text)
	}
	d.Press("tab")
	expect(t, a, "tab to after leaves", "blur trigger, focus after, outside")
}

func TestNonModalScopeInsideModal(t *testing.T) {
	a := &focusApp{}
	d := a.popover(t, true)
	expect(t, a, "the dialog takes focus", "focus before")
	d.Press("tab")
	d.Press("enter")
	a.take()
	d.Press("escape")
	expect(t, a, "escape returns focus to the trigger inside the dialog", "one escape, blur one, focus trigger")
	d.Press("enter")
	d.Press("tab")
	d.Press("tab")
	d.Press("tab")
	expect(t, a, "tab leaves the popover and stays trapped in the dialog", "trigger enter, blur trigger, focus one, blur one, focus two, blur two, focus after, outside, blur after, focus before")
}
