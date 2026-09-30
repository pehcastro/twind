package main

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi/theme"
)

func TestOverlaysKeys(t *testing.T) {
	ring := zinc(theme.Light).Tokens[theme.Ring].RGBA
	d := driven(t, "overlays", theme.Light)
	defer func() {
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	has := func(s string) bool { return strings.Contains(d.Frame().Text(), s) }
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s, ring rows %v:\n%s", what, ringRows(d.Frame(), ring), d.Frame().Text())
		}
	}
	ringWithin := func(top, bottom int) bool {
		rows := ringRows(d.Frame(), ring)
		return len(rows) > 0 && slices.Min(rows) >= top && slices.Max(rows) <= bottom
	}
	press := func(key string, times int) {
		for range times {
			d.Press(key)
		}
	}

	press("tab", 5)
	expect("five tabs reach the menu trigger", ringWithin(find(t, d.Frame(), "Dropdown menu").y, find(t, d.Frame(), "Selected:").y))
	press("enter", 1)
	expect("enter opens the menu", has("My Account") && has("Profile"))
	press("down", 2)
	press("right", 1)
	expect("right opens Invite users", has("Email") && has("Message"))
	t.Logf("menu open, Down twice, Right into the sub menu, zinc light, 150x45:\n%s", d.Frame().Text())
	press("enter", 1)
	expect("enter on Email closes every level and shows the choice", !has("My Account") && !has("Message") && has("Selected: Email"))
	expect("focus is back on the menu trigger", ringWithin(find(t, d.Frame(), "Dropdown menu").y, find(t, d.Frame(), "Selected:").y))
	t.Logf("after Enter on Email:\n%s", d.Frame().Text())

	press("shift+tab", 4)
	press("enter", 1)
	expect("enter on Edit Profile opens the dialog", has("Edit profile") && has("Save changes"))
	top, bottom := find(t, d.Frame(), "Edit profile").y-2, find(t, d.Frame(), "Save changes").y+2
	for i := range 5 {
		d.Press("tab")
		if i == 3 {
			x := find(t, d.Frame(), "✕")
			above := []rune(strings.Split(d.Frame().Text(), "\n")[x.y-1])
			expect("tab 4 rings the X close, drawn at its opacity-70, and nothing else", len(ringRows(d.Frame(), ring)) == 0 && above[x.x] == '▁')
			continue
		}
		expect("tab "+string(rune('1'+i))+" keeps the focus ring inside the dialog", ringWithin(top, bottom))
	}
	t.Logf("dialog open, five tabs back to Name:\n%s", d.Frame().Text())
	press("escape", 1)
	expect("escape closes the dialog", !has("Edit profile"))
	expect("focus is back on Edit Profile", ringWithin(find(t, d.Frame(), "Dialog").y, find(t, d.Frame(), "Alert dialog").y-1))
	t.Logf("after Escape:\n%s", d.Frame().Text())
	if err := d.Err(); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkMenuKeyToFrame(b *testing.B) {
	d := driven(b, "overlays", theme.Dark)
	for range 5 {
		d.Press("tab")
	}
	samples := make([]time.Duration, 0, b.N)
	b.ResetTimer()
	for range b.N {
		start := time.Now()
		d.Press("enter")
		samples = append(samples, time.Since(start))
		b.StopTimer()
		d.Press("escape")
		b.StartTimer()
	}
	b.StopTimer()
	if err := d.Close(); err != nil {
		b.Fatal(err)
	}
	slices.Sort(samples)
	b.ReportMetric(float64(samples[len(samples)/2].Microseconds())/1000, "p50-ms")
	b.ReportMetric(float64(samples[len(samples)*95/100].Microseconds())/1000, "p95-ms")
}
