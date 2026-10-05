package runtime_test

import (
	"testing"

	"github.com/pehcastro/twind/internal/runtime/testdata/hover"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/style"
)

func startRings(t *testing.T, hidden bool) *pointed {
	sheet, err := hover.Styles()
	if err != nil {
		t.Fatal(err)
	}
	p := &pointed{t: t, trace: &hover.Trace{}, sheet: sheet}
	p.d = drive.New(func(rt *twi.Runtime) func() twi.Node {
		if hidden {
			rt.HideFocusRings()
		}
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-black text-white"),
				twi.Element(twi.Class("flex flex-row gap-2"),
					twi.Element(twi.Focusable(), twi.Class(hover.Pill), twi.Text("one")),
					twi.Element(twi.Focusable(), twi.Class(hover.Pill), twi.Text("two")),
					twi.Element(twi.Focusable(), twi.Tag(style.ElementInput), twi.Class(hover.Pill), twi.Text("field")),
				),
				twi.Element(twi.Class("text-zinc-400"), twi.Text("plain text")),
			)
		}
	}, drive.Size(30, 8), drive.Styles(sheet))
	t.Cleanup(func() {
		if err := p.d.Err(); err != nil {
			t.Error(err)
		}
		if err := p.d.Close(); err != nil {
			t.Error(err)
		}
	})
	return p
}

func TestFocusRingOnlyFromTheKeyboard(t *testing.T) {
	p := startRings(t, false)
	ring := func(step, word string, want bool) {
		t.Helper()
		if got := p.underlined(word); got != want {
			t.Errorf("%s: %s focus-visible %v, want %v:\n%s", step, word, got, want, p.d.Frame().ANSI())
		}
	}
	p.d.Press("tab")
	ring("tab onto one", "one", true)
	p.d.Click(p.at("one"))
	ring("a click on the focused one", "one", false)
	p.d.Press("tab")
	ring("tab onto two", "two", true)
	p.d.Click(p.at("plain"))
	ring("a press on plain text", "two", false)
	p.d.Press("x")
	ring("a key after the press", "two", true)
	x, y := p.at("one")
	p.d.Down(x, y)
	ring("the press frame on one", "one", false)
	p.d.Up(p.at("plain"))
	ring("a release off one", "one", false)
	p.d.Click(p.at("field"))
	ring("a click on a text field", "field", true)
}

func TestHideFocusRings(t *testing.T) {
	p := startRings(t, true)
	p.d.Press("tab")
	if p.underlined("one") {
		t.Errorf("tab with rings hidden shows focus-visible on one:\n%s", p.d.Frame().ANSI())
	}
	p.d.Click(p.at("field"))
	if p.underlined("field") {
		t.Errorf("a text field click with rings hidden shows focus-visible:\n%s", p.d.Frame().ANSI())
	}
}
