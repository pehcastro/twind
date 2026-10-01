package main

import (
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/theme"
)

func near(t *testing.T, f drive.Frame, anchor, s string) spot {
	t.Helper()
	a := find(t, f, anchor)
	for y, line := range strings.Split(f.Text(), "\n")[a.y:] {
		runes := []rune(line)
		if before, _, ok := strings.Cut(string(runes[min(a.x, len(runes)):]), s); ok {
			return spot{a.x + utf8.RuneCountInString(before), a.y + y}
		}
	}
	t.Fatalf("no %q at or below %q:\n%s", s, anchor, f.Text())
	return spot{}
}

func TestOverlaysPointer(t *testing.T) {
	light := theme.Default().WithScheme(theme.Light)
	ring, accent, background := light.Tokens[theme.Ring].RGBA, light.Tokens[theme.Accent].RGBA, light.Tokens[theme.Background].RGBA
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
			t.Errorf("%s:\n%s", what, d.Frame().Text())
		}
	}
	at := func(anchor, s string) (int, int) {
		p := near(t, d.Frame(), anchor, s)
		return p.x, p.y
	}
	bg := func(anchor, s string) color.RGBA {
		x, y := at(anchor, s)
		return d.Frame().Cells().At(x, y).Bg.RGBA
	}
	move := func(x, y int) {
		d.Move(x, y)
		d.Advance(settleTime)
	}
	click := func(x, y int) {
		d.Click(x, y)
		d.Advance(settleTime)
	}

	expect("Edit Profile rests on the background", bg("Dialog", "Edit Profile") == background)
	move(at("Dialog", "Edit Profile"))
	expect("hover paints Edit Profile with hover:bg-accent", bg("Dialog", "Edit Profile") == accent)
	t.Logf("hover on Edit Profile, twind light, 150x45:\n%s", d.Frame().ANSI())
	move(at("Dropdown menu", "Selected"))
	expect("leaving Edit Profile drops the hover colour", bg("Dialog", "Edit Profile") == background)

	click(at("Dropdown menu", "Open"))
	expect("a click on Open opens the menu", has("My Account") && has("Billing"))
	move(at("My Account", "Invite users"))
	expect("hovering Invite users opens its sub menu", has("Email") && has("Message"))
	move(at("My Account", "Billing"))
	expect("hovering Billing closes the sub menu", !has("Message"))
	t.Logf("menu opened by click, pointer on Billing:\n%s", d.Frame().Text())
	click(at("My Account", "Billing"))
	expect("a click on Billing chooses it and closes the menu", !has("My Account") && has("Selected: Billing"))
	expect("focus back on the menu trigger shows no ring after a click", len(ringRows(d.Frame(), ring)) == 0)

	click(at("Popover", "Open popover"))
	expect("a click on Open popover opens it", has("Dimensions"))
	click(at("Team Members", "Team Members"))
	expect("a click outside the popover closes it", !has("Dimensions"))

	move(at("Tooltip", "Hover"))
	expect("hovering the tooltip trigger shows the tooltip", has("Add to library"))
	move(at("Team Members", "Team Members"))
	expect("leaving the trigger hides the tooltip", !has("Add to library"))

	click(at("Dialog", "Edit Profile"))
	expect("a click on Edit Profile opens the dialog", has("Edit profile") && has("Save changes"))
	click(at("Edit profile", "✕"))
	expect("a click on the X closes the dialog", !has("Save changes"))
	expect("focus back on Edit Profile without focus-visible", len(ringRows(d.Frame(), ring)) == 0)
	t.Logf("after the X, focus back on Edit Profile:\n%s", d.Frame().Text())
	d.Press("enter")
	expect("Enter on the restored focus opens the dialog again", has("Save changes"))
	click(1, 1)
	expect("a click on the dimmed page closes the dialog", !has("Save changes"))
	if err := d.Err(); err != nil {
		t.Fatal(err)
	}
}

type counting struct {
	events  chan input.Event
	written chan int
}

func (c *counting) Write(p []byte) (int, error) {
	c.written <- len(p)
	return len(p), nil
}

func (c *counting) Events() <-chan input.Event           { return c.events }
func (c *counting) Size() (width, height int, err error) { return 150, 45, nil }
func (c *counting) Sync() bool                           { return true }
func (c *counting) Exit() error                          { return nil }

type stepping struct{ calls atomic.Int64 }

func (s *stepping) Now() time.Time {
	return time.Unix(0, 0).Add(time.Duration(s.calls.Add(1)) * time.Second)
}

func (s *stepping) After(d time.Duration) <-chan time.Time { return time.After(d) }

func BenchmarkPointerToFrame(b *testing.B) {
	sheet, err := Styles()
	if err != nil {
		b.Fatal(err)
	}
	probe := driven(b, "overlays", theme.Dark)
	spots := map[string]spot{}
	for _, s := range []string{"Edit Profile", "Show Dialog", "Selected", "Team Members"} {
		for y, line := range strings.Split(probe.Frame().Text(), "\n") {
			if before, _, ok := strings.Cut(line, s); ok {
				spots[s] = spot{utf8.RuneCountInString(before), y}
			}
		}
	}
	if err := probe.Close(); err != nil {
		b.Fatal(err)
	}
	be := &counting{events: make(chan input.Event), written: make(chan int, 1)}
	rt := twi.New(twi.Backend(be, &stepping{}), twi.Styles(sheet), twi.Theme(theme.Default()), twi.ColorProfile(color.TrueColor))
	body, _ := page(rt, "overlays", "", "")
	done := make(chan error, 1)
	go func() { done <- rt.Run(func() twi.Node { return screen(twi.Class(""), body()) }) }()
	<-be.written
	now := precise(b)
	send := func(ev input.Event) (time.Duration, int) {
		start := now()
		be.events <- ev
		n := <-be.written
		return now() - start, n
	}
	move := func(s string) input.MouseEvent {
		return input.MouseEvent{X: spots[s].x, Y: spots[s].y, Action: input.MouseMove, Button: input.MouseNone}
	}
	report := func(name string, samples []time.Duration, bytes int) {
		slices.Sort(samples)
		b.ReportMetric(float64(samples[len(samples)/2])/float64(time.Millisecond), name+"-p50-ms")
		b.ReportMetric(float64(samples[len(samples)*95/100])/float64(time.Millisecond), name+"-p95-ms")
		b.ReportMetric(float64(bytes)/float64(len(samples)), name+"-bytes")
	}
	var moves, keys []time.Duration
	moved, typed := 0, 0
	b.ResetTimer()
	for i := range b.N {
		took, n := send(move([]string{"Edit Profile", "Show Dialog"}[i%2]))
		moves, moved = append(moves, took), moved+n
		took, n = send(input.KeyEvent{Key: input.KeyTab})
		keys, typed = append(keys, took), typed+n
	}
	b.StopTimer()
	report("move", moves, moved)
	report("key", keys, typed)
	send(move("Selected"))
	empty := 0
	for i := range 100 {
		be.events <- move([]string{"Selected", "Team Members"}[i%2])
		settled := make(chan struct{})
		rt.Dispatch(func() { rt.Dispatch(func() { close(settled) }) })
		<-settled
		select {
		case n := <-be.written:
			empty += n
			b.Errorf("a move over text without a hover style wrote %d bytes", n)
		default:
		}
	}
	b.ReportMetric(float64(empty), "empty-bytes")
	rt.Quit()
	if err := <-done; err != nil {
		b.Fatal(err)
	}
}

func TestFormPointer(t *testing.T) {
	d := driven(t, "form", theme.Light)
	defer func() {
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	line := func(anchor string) string {
		return strings.Split(d.Frame().Text(), "\n")[find(t, d.Frame(), anchor).y]
	}
	click := func(anchor string, dx int) {
		p := find(t, d.Frame(), anchor)
		d.Click(p.x+dx, p.y)
	}
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s:\n%s", what, d.Frame().Text())
		}
	}
	expect("the terms box starts empty", !strings.Contains(line("Accept terms"), "✓"))
	click("Accept terms", -4)
	expect("a click on the terms box checks it", strings.Contains(line("Accept terms"), "✓"))
	click("Disabled", -4)
	expect("a click on the disabled box does nothing", !strings.Contains(line("Disabled"), "✓"))
	click("Compact", 0)
	expect("a click on the Compact label picks it", strings.Contains(line("Compact"), "●") && !strings.Contains(line("Comfortable"), "●"))
	off := cellRow(t, d.Frame(), "Airplane Mode")
	click("Airplane Mode", -5)
	expect("a click on the switch moves its thumb", !slices.Equal(cellRow(t, d.Frame(), "Airplane Mode"), off))
	t.Logf("after clicks on terms, Compact and the switch, twind light, 150x45:\n%s", d.Frame().Text())
	if err := d.Err(); err != nil {
		t.Fatal(err)
	}
}
