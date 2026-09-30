package main

import (
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/theme"
)

func zinc(scheme theme.Scheme) theme.Theme {
	for _, th := range theme.Builtin() {
		if th.Name == "zinc" && th.Scheme == scheme {
			return th
		}
	}
	panic("no zinc theme")
}

func driven(t testing.TB, name string, scheme theme.Scheme) *drive.Driver {
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	return drive.New(func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(scheme))
		body, ok := page(rt, name, "", "")
		if !ok {
			t.Fatalf("no page %q", name)
		}
		return func() twi.Node { return screen(twi.Class(""), body()) }
	}, drive.Size(150, 45), drive.Styles(sheet))
}

func TestDrivenFrame(t *testing.T) {
	for name, want := range map[string][]string{
		"wave1":         {"Destructive", "Login to your account", "Sign Up", "Unable to process your payment.", "No Projects Yet", "Ctrl", "CN", "LR"},
		"tables":        {"Invoice", "INV007", "Bank Transfer", "$2,500.00", "A list of your recent invoices."},
		"breadcrumbs":   {"Home", "›", "…", "Components", "/", "Breadcrumb"},
		"pagination":    {"‹ Previous", "Next ›", "…"},
		"items":         {"Basic Item", "Your profile has been verified.", "Muted Variant", "evilrabbit@vercel.com"},
		"button-groups": {"Archive", "Report", "Snooze", "Copy", "Paste", "https://"},
		"fields":        {"Payment Method", "Card Number", "Billing Address", "Submit", "Enter a valid email address.", "Or continue with"},
		"form":          {"Email", "shadcn!", "Type your message here.", "https://", "Select status", "Accept terms and conditions", "✓", "Airplane Mode", "●", "Bookmark", "50"},
		"overlays":      {"Edit Profile", "Show Dialog", "Open Drawer", "Open popover", "Hover", "@nextjs", "Selected: nothing yet", "Team Members"},
		"tabs":          {"Account", "Password", "Make changes to your account here.", "@peduarte", "Save changes", "Select a fruit", "Coordinated Universal Time"},
	} {
		for _, scheme := range []theme.Scheme{theme.Light, theme.Dark} {
			d := driven(t, name, scheme)
			frame := d.Frame().Text()
			if err := d.Close(); err != nil {
				t.Fatal(err)
			}
			t.Logf("%s, scheme %d:\n%s", name, scheme, frame)
			for _, s := range want {
				if !strings.Contains(frame, s) {
					t.Errorf("%s: no %q in the frame", name, s)
				}
			}
		}
	}
	if _, ok := page(nil, "nothing", "", ""); ok {
		t.Error("a page that does not exist was found")
	}
}

type spot struct{ x, y int }

func find(t *testing.T, f drive.Frame, s string) spot {
	t.Helper()
	for y, line := range strings.Split(f.Text(), "\n") {
		if at := strings.Index(line, s); at >= 0 {
			return spot{len([]rune(line[:at])), y}
		}
	}
	t.Fatalf("no %q in the frame:\n%s", s, f.Text())
	return spot{}
}

func ringRows(f drive.Frame, ring color.RGBA) []int {
	var rows []int
	cells := f.Cells()
	for y := range cells.Height() {
		for x := range cells.Width() {
			if c := cells.At(x, y); strings.ContainsAny(c.Grapheme, "▁▂▃▄▅▆▇▏▎▍▌▋▊▉▔▕") && c.Fg.RGBA == ring && !slices.Contains(rows, y) {
				rows = append(rows, y)
			}
		}
	}
	return rows
}

func TestFormKeys(t *testing.T) {
	light := zinc(theme.Light)
	ring, accent := light.Tokens[theme.Ring].RGBA, light.Tokens[theme.Accent].RGBA
	d := driven(t, "form", theme.Light)
	defer func() {
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	if rows := ringRows(d.Frame(), ring); len(rows) != 0 {
		t.Fatalf("a ring before any Tab, rows %v", rows)
	}
	ringOn := func(anchor string, below, height int) {
		t.Helper()
		f := d.Frame()
		top := find(t, f, anchor).y + below
		rows := ringRows(f, ring)
		if len(rows) == 0 || slices.Min(rows) < top-1 || slices.Max(rows) != top+height {
			t.Errorf("the focus ring is on rows %v, want it closed by row %d and nothing above %d, around %q:\n%s", rows, top+height, top-1, anchor, f.Text())
		}
	}
	focused := func(anchor string, below, height int) {
		t.Helper()
		d.Press("tab")
		ringOn(anchor, below, height)
	}
	line := func(anchor string) string {
		f := d.Frame()
		return strings.Split(f.Text(), "\n")[find(t, f, anchor).y]
	}
	bg := func(anchor string, below int, s string) color.RGBA {
		f := d.Frame()
		y := find(t, f, anchor).y + below
		row := strings.Split(f.Text(), "\n")[y]
		before, _, _ := strings.Cut(row, s)
		return f.Cells().At(len([]rune(before)), y).Bg.RGBA
	}
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s:\n%s", what, d.Frame().Text())
		}
	}

	focused("Email", 2, 1)
	d.Type("m@example.com")
	expect("the email input holds what was typed", regexp.MustCompile(`[▕▋▊] m@example\.com`).MatchString(line("m@example.com")))

	d.Press("tab")
	expect("the invalid username takes focus with its destructive ring, so no ring-coloured cell is left", len(ringRows(d.Frame(), ring)) == 0)

	focused("Your message", 2, 4)
	d.Type("Hello")
	d.Press("enter")
	d.Type("world")
	hello, world := find(t, d.Frame(), "Hello"), find(t, d.Frame(), "world")
	expect("the textarea breaks the line at Enter", world.y == hello.y+1 && world.x == hello.x)

	focused("Website", 2, 1)
	d.Type("twind")
	expect("the group input sits between its addons", strings.Contains(line("https://"), "https:// twind") && strings.Contains(line("https://"), ".com"))

	focused("Status", 2, 1)
	d.Press("down")
	expect("down picks the next status", strings.Contains(line("Todo"), "⌄"))

	focused("Accept terms", 0, 1)
	d.Press("space")
	expect("space checks the terms box", strings.Contains(line("Accept terms"), "✓"))

	focused("Enable notifications", 0, 1)

	focused("Airplane Mode", 0, 1)
	off := line("Airplane Mode")
	d.Press("space")
	expect("space moves the switch thumb", line("Airplane Mode") != off)

	focused("Comfortable", 0, 1)
	d.Press("down")
	expect("down checks Compact", strings.Contains(line("Compact"), "●") && !strings.Contains(line("Comfortable"), "●"))
	ringOn("Compact", 0, 1)

	focused("Bookmark", 0, 1)
	d.Press("space")
	expect("space presses the toggle", bg("Bookmark", 0, "Bookmark") == accent)

	focused("Toggle group", 2, 1)
	d.Press("right")
	d.Press("space")
	expect("right then space presses I and keeps B", bg("Toggle group", 2, "B") == accent && bg("Toggle group", 2, "I") == accent && bg("Toggle group", 2, "U") != accent)

	focused("Slider", 2, 1)
	before := line("50")
	d.Press("right")
	d.Press("right")
	d.Press("right")
	expect("three rights take the slider to 53", strings.Contains(line("53"), "53"))
	d.Press("pageup")
	expect("page up takes it ten further and the thumb moves", line("63") != strings.Replace(before, "50", "63", 1))

	focused("Input OTP", 2, 1)
	d.Type("123456")
	otp := strings.Split(d.Frame().Text(), "\n")[find(t, d.Frame(), "Input OTP").y+2]
	expect("typing fills the six slots", strings.Contains(strings.Map(func(r rune) rune {
		if strings.ContainsRune("▁▂▃▄▅▆▇▏▎▍▌▋▊▉▔▕ ", r) {
			return -1
		}
		return r
	}, otp), "123-456"))

	focused("Email", 2, 1)
	t.Logf("after the script, zinc light, 150x45:\n%s", d.Frame().Text())
	if err := d.Err(); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkFormKeyToFrame(b *testing.B) {
	d := driven(b, "form", theme.Dark)
	d.Press("tab")
	samples := make([]time.Duration, 0, b.N)
	b.ResetTimer()
	for i := range b.N {
		start := time.Now()
		d.Press([]string{"a", "backspace"}[i%2])
		samples = append(samples, time.Since(start))
	}
	b.StopTimer()
	if err := d.Close(); err != nil {
		b.Fatal(err)
	}
	slices.Sort(samples)
	b.ReportMetric(float64(samples[len(samples)/2].Microseconds())/1000, "p50-ms")
	b.ReportMetric(float64(samples[len(samples)*95/100].Microseconds())/1000, "p95-ms")
}
