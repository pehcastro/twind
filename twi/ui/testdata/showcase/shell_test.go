package main

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/theme"
)

func TestShellScript(t *testing.T) {
	d := driven(t, "shell", theme.Light)
	defer func() {
		if err := errors.Join(d.Err(), d.Close()); err != nil {
			t.Fatal(err)
		}
	}()
	has := func(s string) bool { return strings.Contains(d.Frame().Text(), s) }
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s:\n%s", what, d.Frame().Text())
		}
	}
	press := func(keys ...string) {
		for _, k := range keys {
			d.Press(k)
			d.Advance(settleTime)
		}
	}
	click := func(s spot) {
		d.Click(s.x, s.y)
		d.Advance(settleTime)
	}

	icon := find(t, d.Frame(), "▸")
	press("ctrl+b")
	expect("ctrl+b folds the sidebar to its icons", !has("Playground") && !has("Platform") && !has("Acme Inc") && find(t, d.Frame(), "▸").x == icon.x)
	t.Logf("shell, sidebar folded by ctrl+b, zinc light, 150x45:\n%s", d.Frame().Text())

	click(find(t, d.Frame(), "File"))
	expect("a click on File opens it", has("New Tab") && has("Print..."))
	press("right")
	expect("right moves the open menu to Edit", has("Undo") && has("Paste") && !has("New Tab"))
	t.Logf("File opened by click, right moved to Edit:\n%s", d.Frame().Text())
	press("escape")
	expect("escape closes Edit", !has("Undo"))

	card := find(t, d.Frame(), "Right click here")
	d.ClickWith(input.MouseRight, card.x+2, card.y)
	d.Advance(settleTime)
	back := find(t, d.Frame(), "Back")
	expect("a right click on the card opens its menu at the pointer", back.x == card.x+2+4+2 && back.y == card.y+1)
	t.Logf("context menu opened by a right click on the card:\n%s", d.Frame().Text())
	click(find(t, d.Frame(), "Reload"))
	expect("a click on Reload chooses it and closes the menu", !has("Back") && has("Chosen: Reload"))
	t.Logf("after choosing Reload:\n%s", d.Frame().Text())
}

func BenchmarkShellKeyToFrame(b *testing.B) {
	d := driven(b, "shell", theme.Dark)
	d.Press("tab")
	now := precise(b)
	samples := make([]time.Duration, 0, b.N)
	b.ResetTimer()
	for i := range b.N {
		start := now()
		d.Press("ctrl+b")
		samples = append(samples, now()-start)
		if i%2 == 1 {
			b.StopTimer()
			d.Advance(settleTime)
			b.StartTimer()
		}
	}
	b.StopTimer()
	if err := errors.Join(d.Err(), d.Close()); err != nil {
		b.Fatal(err)
	}
	slices.Sort(samples)
	b.ReportMetric(float64(samples[len(samples)/2])/float64(time.Millisecond), "p50-ms")
	b.ReportMetric(float64(samples[len(samples)*95/100])/float64(time.Millisecond), "p95-ms")
}
