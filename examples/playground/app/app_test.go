package app

import (
	"strings"
	"testing"
	"unicode"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/tailwind"
)

func TestStylesFresh(t *testing.T) {
	stale, err := tailwind.Stale(".", konst.GeneratedFile)
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Errorf("%s is stale against the classes in this package: run go generate", konst.GeneratedFile)
	}
}

func open(t *testing.T) *drive.Driver {
	t.Helper()
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	d := drive.New(App, drive.Size(100, 30), drive.Styles(sheet))
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

func lines(d *drive.Driver) []string {
	return strings.Split(strings.TrimRight(d.Frame().Text(), "\n"), "\n")
}

func status(d *drive.Driver) string {
	all := lines(d)
	return strings.Join(strings.Fields(all[len(all)-1]), " ")
}

func inputLine(d *drive.Driver) string {
	for _, l := range lines(d) {
		if before, _, ok := strings.Cut(l, "tab focus"); ok {
			return strings.TrimSpace(strings.SplitN(before, "▌", 2)[1])
		}
	}
	return "no input line"
}

func cursorColumn(t *testing.T, d *drive.Driver) (int, string) {
	t.Helper()
	cells := d.Frame().Cells()
	foreground := color.RGBA{R: 250, G: 250, B: 250, A: 255}
	for y := range cells.Height() {
		row := cells.Row(y)
		start := -1
		for x, c := range row {
			if c.Grapheme == "▌" {
				start = x + 2
			}
			if start >= 0 && c.Bg.RGBA == foreground {
				return x - start, c.Grapheme
			}
		}
	}
	return -1, ""
}

func press(d *drive.Driver, keys ...string) {
	for _, k := range keys {
		d.Press(k)
	}
}

func run(d *drive.Driver, command string) {
	d.Type(command)
	d.Press("enter")
}

func TestTypeDeleteWordUndoThenTrapFocus(t *testing.T) {
	d := open(t)
	steps := []struct {
		act            func()
		input, focus   string
		cursor         int
		cursorGrapheme string
	}{
		{func() { d.Type("hello") }, "hello", "input", 5, nbsp},
		{func() { d.Press("ctrl+w") }, "Ask twind: a page, theme, dialog, toast, load, later or quit", "input", 0, nbsp},
		{func() { d.Type("world") }, "world", "input", 5, nbsp},
		{func() { d.Press("ctrl+z") }, "Ask twind: a page, theme, dialog, toast, load, later or quit", "input", 0, nbsp},
		{func() { d.Press("ctrl+z") }, "hello", "input", 5, nbsp},
		{func() { d.Type(" 中b"); press(d, "left", "left") }, "hello" + nbsp + "中b", "input", 6, "中"},
		{func() { d.Press("right") }, "hello" + nbsp + "中b", "input", 8, "b"},
		{func() { d.Press("ctrl+a") }, "hello" + nbsp + "中b", "input", -1, ""},
		{func() { press(d, "tab", "tab", "tab", "tab", "tab", "tab", "tab", "tab") }, "hello" + nbsp + "中b", "theme", -1, ""},
	}
	for i, s := range steps {
		s.act()
		if got := inputLine(d); got != s.input {
			t.Errorf("step %d: input shows %q, want %q", i, got, s.input)
		}
		if got := status(d); !strings.HasSuffix(got, "focus "+s.focus+" surfaces") {
			t.Errorf("step %d: status %q, want focus %s", i, got, s.focus)
		}
		if col, g := cursorColumn(t, d); col != s.cursor || g != s.cursorGrapheme {
			t.Errorf("step %d: cursor cell at column %d on %q, want %d on %q", i, col, g, s.cursor, s.cursorGrapheme)
		}
	}
	t.Logf("focus on the theme button:\n%s", d.Frame().Text())
	press(d, "enter")
	if text := d.Frame().Text(); !strings.Contains(text, "Enter applies") || !strings.HasSuffix(status(d), "focus themes surfaces") {
		t.Fatalf("enter on the theme button did not open the picker with focus on the list:\n%s", text)
	}
	for _, want := range []string{"close", "themes", "close"} {
		press(d, "tab")
		if got := status(d); !strings.HasSuffix(got, "focus "+want+" surfaces") {
			t.Errorf("tab inside the picker: status %q, want focus %s", got, want)
		}
	}
	t.Logf("picker open, focus on close:\n%s", d.Frame().Text())
	press(d, "shift+tab", "4", "escape")
	if text := d.Frame().Text(); strings.Contains(text, "Enter applies") || status(d) != "● fullscreen headless truecolor zinc-dark focus theme surfaces" {
		t.Errorf("escape: want the picker closed, focus back on the theme button and no key leaked, status %q:\n%s", status(d), text)
	}
	t.Logf("after escape:\n%s", d.Frame().Text())
}

func TestPages(t *testing.T) {
	d := open(t)
	run(d, "layout")
	if got := status(d); !strings.HasSuffix(got, "layout") || inputLine(d) != "Ask twind: a page, theme, dialog, toast, load, later or quit" {
		t.Errorf("layout, enter: status %q, input %q", got, inputLine(d))
	}
	press(d, "tab", "enter")
	if got := status(d); !strings.HasSuffix(got, "focus surfaces surfaces") {
		t.Errorf("enter on the first pill: status %q", got)
	}
	press(d, "2", "tab", "space")
	if got := status(d); !strings.HasSuffix(got, "focus text text") {
		t.Errorf("2 then space on the second pill: status %q", got)
	}
}

func TestPickerOwnsKeys(t *testing.T) {
	d := open(t)
	run(d, "theme")
	if text := d.Frame().Text(); !strings.Contains(text, "● zinc-dark") {
		t.Fatalf("picker open does not mark the applied theme:\n%s", text)
	}
	press(d, "4", "+", "escape")
	if got := status(d); got != "● fullscreen headless truecolor zinc-dark focus input surfaces" {
		t.Errorf("keys reached the page under the picker, or escape applied: status %q", got)
	}
	run(d, "t")
	press(d, "down", "enter")
	if got := status(d); !strings.Contains(got, "slate-light") || strings.Contains(d.Frame().Text(), "Enter applies") {
		t.Errorf("down, enter: want slate-light applied and the picker closed, status %q", got)
	}
	run(d, "t")
	press(d, "up", "up", "escape")
	run(d, "t")
	press(d, "enter")
	if got := status(d); !strings.Contains(got, "slate-light") {
		t.Errorf("reopened picker did not start on the applied theme: status %q", got)
	}
	run(d, "t")
	press(d, "up", "up", "up", "up", "up")
	if text := d.Frame().Text(); !strings.Contains(text, "violet-dark") {
		t.Errorf("cursor wrapped past the first theme to violet-dark but the list does not show it:\n%s", text)
	}
}

func TestCounter(t *testing.T) {
	d := open(t)
	run(d, "counter")
	press(d, "shift+tab", "shift+tab")
	if got := status(d); !strings.HasSuffix(got, "focus theme counter") {
		t.Fatalf("shift+tab from increment skips the decrement disabled at 0: status %q", got)
	}
	press(d, "tab", "enter", "enter", "shift+tab")
	if got := status(d); !strings.HasSuffix(got, "focus decrement counter") || !strings.Contains(d.Frame().Text(), "▀▀█") {
		t.Fatalf("want 2 after enter twice on increment and decrement enabled:\n%s", d.Frame().Text())
	}
	press(d, "enter", "enter", "enter", "1", "4")
	if text := d.Frame().Text(); !strings.Contains(text, "█"+nbsp+"█") {
		t.Errorf("want 0 after three presses of decrement from 2, and a page round trip:\n%s", text)
	}
}

func TestResizeReflows(t *testing.T) {
	d := open(t)
	run(d, "4")
	d.Resize(80, 24)
	text := d.Frame().Text()
	all := lines(d)
	if len(all) != 24 || !strings.Contains(all[1], "counter") || !strings.HasSuffix(status(d), "counter") || !strings.Contains(all[21], "Ask twind") {
		t.Errorf("80x24 frame lost the top bar, input bar or status line:\n%s", text)
	}
	for _, l := range all {
		if w := len([]rune(strings.TrimRight(l, " "))); w > 80 {
			t.Errorf("a line is %d runes wide after the resize: %q", w, l)
		}
	}
}

func TestHostileTextInert(t *testing.T) {
	d := open(t)
	run(d, "text")
	text := d.Frame().Text()
	if !strings.Contains(text, "clear bell bidiexe.txt") {
		t.Errorf("hostile string not shown as plain text:\n%s", text)
	}
	if i := strings.IndexFunc(text, func(r rune) bool { return r != '\n' && unicode.IsControl(r) || r == 0x202e }); i >= 0 {
		t.Errorf("a control or bidi character reached the screen at byte %d", i)
	}
}
