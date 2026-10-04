package app

import (
	"strings"
	"testing"
	"time"
)

func TestFormPageSignsUp(t *testing.T) {
	d := open(t)
	d.Press("ctrl+k")
	d.Type("form")
	d.Press("enter")
	d.Advance(time.Second)
	d.Click(spot(t, d, "shadcn"))
	d.Type("a")
	d.Press("tab")
	d.Advance(time.Second)
	if !strings.Contains(d.Frame().Text(), "Use 2 or more characters.") {
		t.Fatalf("leaving a short username showed no message:\n%s", d.Frame().Text())
	}
	t.Logf("one error shown:\n%s", d.Frame().Text())
	d.Press("enter")
	d.Advance(time.Second)
	if text := d.Frame().Text(); !strings.Contains(text, "Email is required.") || !strings.Contains(text, "Accept the terms first.") || !strings.Contains(text, "joined: not yet") {
		t.Fatalf("submitting an empty form did not show every error:\n%s", text)
	}
	t.Logf("submitted with errors:\n%s", d.Frame().Text())
	d.Type("da")
	press(d, "tab")
	d.Type("ada@example.com")
	press(d, "tab")
	d.Type("hunter22")
	for _, k := range []string{"tab", "enter", "enter", "tab", " ", "tab", "tab", "enter"} {
		d.Press(k)
		d.Advance(time.Second)
	}
	if !strings.Contains(d.Frame().Text(), "joined: ada on free") {
		t.Fatalf("a valid form did not sign up:\n%s", d.Frame().Text())
	}
	t.Logf("signed up:\n%s", d.Frame().Text())
}
