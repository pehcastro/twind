package main

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/theme"
	"github.com/twind-dev/twind/twi/ui"
)

func BenchmarkCommandKeyToFrame(b *testing.B) {
	sheet, err := Styles()
	if err != nil {
		b.Fatal(err)
	}
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(theme.Dark))
		palette := ui.NewCommandDialog(rt)
		palette.Open = true
		return func() twi.Node {
			items := make([]ui.CommandItem, 0, 1000)
			for i := range 1000 {
				items = append(items, palette.Item("Command number "+strconv.Itoa(i)))
			}
			return screen(twi.Class(""), palette.Node(palette.Input("Type a command or search..."), palette.List(palette.Group("Commands", items...))))
		}
	}, drive.Size(150, 45), drive.Styles(sheet))
	now := precise(b)
	keys := []string{"c", "o", "m", "backspace", "backspace", "backspace"}
	samples := make([]time.Duration, 0, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		start := now()
		if k := keys[i%len(keys)]; len(k) == 1 {
			d.Type(k)
		} else {
			d.Press(k)
		}
		samples = append(samples, now()-start)
	}
	b.StopTimer()
	if err := errors.Join(d.Err(), d.Close()); err != nil {
		b.Fatal(err)
	}
	slices.Sort(samples)
	b.ReportMetric(float64(samples[len(samples)/2])/float64(time.Millisecond), "p50-ms")
	b.ReportMetric(float64(samples[len(samples)*95/100])/float64(time.Millisecond), "p95-ms")
}

func TestWave3bScript(t *testing.T) {
	d := driven(t, "wave3b", theme.Light)
	defer func() {
		if err := d.Close(); err != nil {
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
	click := func(anchor, s string) {
		p := near(t, d.Frame(), anchor, s)
		d.Click(p.x, p.y)
	}

	expect("the first item is open like the demo", has("Our flagship product") && !has("worldwide shipping"))
	d.Press("tab")
	d.Press("down")
	d.Press("enter")
	expect("tab, down and enter open Shipping Details and close Product Information", has("worldwide shipping") && !has("Our flagship product"))
	click("Accordion", "Return Policy")
	expect("a click opens Return Policy", has("comprehensive 30-day return") && !has("worldwide shipping"))
	t.Logf("wave3b, two accordion items opened in turn, Return Policy open, zinc light, 150x45:\n%s", d.Frame().Text())

	click("Sonner", "Show Toast")
	click("Sonner", "Success")
	click("Sonner", "Error")
	expect("three toasts, the newest in front", has("Event has not been created") && !has("Sunday, December 03"))
	t.Logf("three toasts stacked at the bottom right:\n%s", d.Frame().Text())
	d.Advance(3900 * time.Millisecond)
	expect("still there at 3.9 s", has("Event has not been created"))
	d.Advance(100 * time.Millisecond)
	expect("all three gone at 4 s", !has("has been created") && !has("has not been created"))
	t.Logf("toasts waited out:\n%s", d.Frame().Text())

	d.Press("ctrl+k")
	d.Advance(settleTime)
	expect("ctrl+k opens the command dialog", strings.Count(d.Frame().Text(), "Type a command or search...") == 2)
	d.Type("bil")
	t.Logf("command dialog, typed bil:\n%s", d.Frame().Text())
	expect("typing filters the dialog to Billing", strings.Contains(d.Frame().Text(), "Billing") && strings.Count(d.Frame().Text(), "Calendar") == 1)
	d.Press("enter")
	d.Advance(settleTime)
	expect("enter chooses Billing and closes the dialog", has("Chosen: Billing") && strings.Count(d.Frame().Text(), "Billing") == 2 && strings.Count(d.Frame().Text(), "Type a command or search...") == 1)
	t.Logf("after choosing:\n%s", d.Frame().Text())
}
