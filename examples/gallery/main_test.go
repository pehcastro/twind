package main

import (
	"strings"
	"testing"

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
	light, ok := builtin("zinc-light")
	if !ok {
		t.Fatal("no zinc-light theme")
	}
	return drive.New(func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(light)
		return gallery(rt)
	}, drive.Size(110, 34), drive.Styles(sheet))
}

func row(lines []string, s string) (int, int) {
	for i, l := range lines {
		if col := strings.Index(l, s); col >= 0 {
			return i, len([]rune(l[:col]))
		}
	}
	return -1, -1
}

func TestFrameHoldsEverySection(t *testing.T) {
	d := open(t)
	text := d.Frame().Text()
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if len(lines) != 34 {
		t.Errorf("frame has %d rows, want 34", len(lines))
	}
	for i, l := range lines {
		if w := len([]rune(strings.TrimRight(l, " "))); w > 110 {
			t.Errorf("row %d is %d runes wide", i, w)
		}
	}
	for _, s := range []string{"Edit", "+1 -1", "6718", "it.each<{", "Welcome back!", "Tips", "Recent sessions", "Use /tan", "Using a mermaid", "64%", ":thumbsdown:", ":thumbsup:", ":tada:", "95 lines up", "59.5 tok/s", ":thumbs▏", "omp ", "GPT-6 Luna", "$ 0.01"} {
		if y, _ := row(lines, s); y < 0 {
			t.Errorf("no %q in the frame:\n%s", s, text)
		}
	}
	input, _ := row(lines, ":thumbs▏")
	status, _ := row(lines, "GPT-6 Luna")
	if status != 33 || input != 30 {
		t.Errorf("input text on row %d and status on row %d, want 30 and 33:\n%s", input, status, text)
	}
	for i, s := range []string{":tada:", ":thumbsup:", ":thumbsdown:"} {
		if y, _ := row(lines, s); y != input-3-i {
			t.Errorf("popover row %q on row %d, want %d, just above the input's border:\n%s", s, y, input-3-i, text)
		}
	}
	_, popover := row(lines, ":thumbsdown:")
	if y, chip := row(lines, "95 lines up"); y != input-5 || chip <= popover+20 {
		t.Errorf("chip at row %d column %d, want row %d right of the popover at column %d:\n%s", y, chip, input-5, popover, text)
	}
	_, now := row(lines, "just now")
	_, ago := row(lines, "3h ago")
	if now+len("just now") != ago+len("3h ago") {
		t.Errorf("timestamps not right-aligned: just now ends at %d, 3h ago at %d", now+len("just now"), ago+len("3h ago"))
	}
}

func TestOnlyQQuits(t *testing.T) {
	d := open(t)
	d.Press("x")
	d.Press("enter")
	if err := d.Err(); err != nil {
		t.Fatalf("a key other than q stopped the app: %v", err)
	}
	d.Press("q")
	d.Press("x")
	if d.Err() == nil {
		t.Error("the app still takes keys after q")
	}
}
