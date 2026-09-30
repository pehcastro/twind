package app

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPointer(t *testing.T) {
	d := open(t)
	at := func(s string) (int, int) {
		t.Helper()
		for y, line := range lines(d) {
			if before, _, ok := strings.Cut(line, s); ok {
				return utf8.RuneCountInString(before), y
			}
		}
		t.Fatalf("no %q in the frame:\n%s", s, d.Frame().Text())
		return 0, 0
	}
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s, status %q:\n%s", what, status(d), d.Frame().Text())
		}
	}
	d.Click(at("counter"))
	expect("a click on the counter tab opens its page", strings.HasSuffix(status(d), "focus counter counter"))
	d.Click(at("+ increment"))
	d.Click(at("+ increment"))
	expect("two clicks on increment count to two", strings.Contains(d.Frame().Text(), "▀▀█"))
	d.Move(at("theme zinc-dark"))
	expect("hovering the theme button shows its tooltip", strings.Contains(d.Frame().Text(), "t or a click opens the picker"))
	d.Click(at("theme zinc-dark"))
	expect("a click on the theme button opens the picker", strings.Contains(d.Frame().Text(), "Enter applies") && !strings.Contains(d.Frame().Text(), "opens the picker"))
	d.Move(at("slate-light"))
	d.Click(at("slate-light"))
	expect("a click on a row applies its theme", strings.Contains(status(d), "slate-light") && !strings.Contains(d.Frame().Text(), "Enter applies"))
	d.Click(at("theme slate-light"))
	d.Click(1, 1)
	expect("a click outside the picker closes it", !strings.Contains(d.Frame().Text(), "Enter applies"))
}
