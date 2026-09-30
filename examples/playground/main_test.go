package main

import (
	"strings"
	"testing"
	"unicode"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi"
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
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		return playground(rt, env{cwd: "/home/user/twind", profile: "truecolor", size: func() string { return "headless" }}, state{theme: themeIndex("zinc-dark")})
	}, drive.Size(100, 30), drive.Styles(sheet))
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

func status(d *drive.Driver) string {
	lines := strings.Split(strings.TrimRight(d.Frame().Text(), "\n"), "\n")
	return strings.Join(strings.Fields(lines[len(lines)-1]), " ")
}

func press(d *drive.Driver, keys ...string) {
	for _, k := range keys {
		d.Press(k)
	}
}

func TestPagesWrap(t *testing.T) {
	d := open(t)
	for _, want := range []string{"text", "layout", "counter", "surfaces"} {
		press(d, "tab")
		if got := status(d); !strings.HasSuffix(got, want) {
			t.Errorf("after tab want page %s, status %q", want, got)
		}
	}
	press(d, "shift+tab")
	if got := status(d); !strings.HasSuffix(got, "counter") {
		t.Errorf("shift+tab on the first page: status %q", got)
	}
	press(d, "2", "5", "0")
	if got := status(d); !strings.HasSuffix(got, "text") {
		t.Errorf("2 then 5 and 0: status %q", got)
	}
}

func TestPickerOwnsKeys(t *testing.T) {
	d := open(t)
	press(d, "t")
	if text := d.Frame().Text(); !strings.Contains(text, "● zinc-dark") {
		t.Fatalf("picker open does not mark the applied theme:\n%s", text)
	}
	press(d, "tab", "4", "+", "escape")
	if got := status(d); got != "● fullscreen headless truecolor zinc-dark surfaces" {
		t.Errorf("keys reached the page under the picker, or escape applied: status %q", got)
	}
	press(d, "t", "down", "enter")
	if got := status(d); !strings.Contains(got, "slate-light") || strings.Contains(d.Frame().Text(), "Enter applies") {
		t.Errorf("down, enter: want slate-light applied and the picker closed, status %q", got)
	}
	press(d, "t", "up", "up", "escape", "t", "enter")
	if got := status(d); !strings.Contains(got, "slate-light") {
		t.Errorf("reopened picker did not start on the applied theme: status %q", got)
	}
	press(d, "t", "up", "up", "up", "up", "up")
	if text := d.Frame().Text(); !strings.Contains(text, "violet-dark") {
		t.Errorf("cursor wrapped past the first theme to violet-dark but the list does not show it:\n%s", text)
	}
}

func TestCounter(t *testing.T) {
	d := open(t)
	press(d, "+", "+", "4")
	if text := d.Frame().Text(); !strings.Contains(text, "█"+nbsp+"█") {
		t.Fatalf("+ on surfaces changed the count:\n%s", text)
	}
	press(d, "+", "+", "+", "1", "4")
	if text := d.Frame().Text(); !strings.Contains(text, nbsp+"▀█") {
		t.Errorf("want 3 in the big digits after three presses and a page round trip:\n%s", text)
	}
}

func TestResizeReflows(t *testing.T) {
	d := open(t)
	press(d, "4")
	d.Resize(80, 24)
	text := d.Frame().Text()
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) != 24 || !strings.Contains(lines[1], "counter") || !strings.HasSuffix(status(d), "counter") || !strings.Contains(lines[21], "Ask twind") {
		t.Errorf("80x24 frame lost the top bar, input bar or status line:\n%s", text)
	}
	for _, l := range lines {
		if w := len([]rune(strings.TrimRight(l, " "))); w > 80 {
			t.Errorf("a line is %d runes wide after the resize: %q", w, l)
		}
	}
}

func TestHostileTextInert(t *testing.T) {
	d := open(t)
	press(d, "2")
	text := d.Frame().Text()
	if !strings.Contains(text, "clear bell bidiexe.txt") {
		t.Errorf("hostile string not shown as plain text:\n%s", text)
	}
	if i := strings.IndexFunc(text, func(r rune) bool { return r != '\n' && unicode.IsControl(r) || r == 0x202e }); i >= 0 {
		t.Errorf("a control or bidi character reached the screen at byte %d", i)
	}
}
