package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/tailwind"
	"github.com/twind-dev/twind/twi/theme"
)

func TestStylesFresh(t *testing.T) {
	stale, err := tailwind.Stale(".", konst.GeneratedFile)
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Errorf("%s is stale against the classes in this package: run twind build", konst.GeneratedFile)
	}
}

func app(t testing.TB) (drive.App, []drive.Option) {
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	s, err := parse("zinc-light", "dashboard", "")
	if err != nil {
		t.Fatal(err)
	}
	return func(rt *twi.Runtime) func() twi.Node { return gallery(rt, s) }, []drive.Option{drive.Styles(sheet), drive.Size(110, 34)}
}

func background(name string, token theme.Token) string {
	t, _ := builtin(name)
	c := t.Tokens[token].RGBA
	return fmt.Sprintf("48;2;%d;%d;%d", c.R, c.G, c.B)
}

func TestTour(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("testdata", "tour.twd"))
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	a, opts := app(t)
	if err := drive.RunScript(strings.NewReader(string(script)), a, out, opts...); err != nil {
		t.Fatal(err)
	}
	nav := []string{"Dashboard", "Forms", "Overlays", "Settings"}
	for _, c := range []struct {
		frame          string
		shown, missing []string
		ansi           map[string]bool
	}{
		{"dashboard", []string{"Acme › Dashboard", "Total Revenue", "$1,250", "↗ 12.5%", "New Customers", "Active Accounts", "Growth Rate", "Total Visitors", "Jan", "Jun", "Desktop", "Cover page", "✓ Done", "◌ In Process", "Page 1 of 3"}, []string{"Innovation"}, nil},
		{"dashboard-page-2", []string{"Capabilities", "Innovation", "Page 2 of 3"}, []string{"Cover page"}, nil},
		{"forms", []string{"Acme › Forms", "Profile", "Username", "shadcn", "Bio", "Accept terms", "Email updates", "Density", "Comfortable", "Volume", "Verification code", "Save changes"}, []string{"Total Revenue"}, nil},
		{"overlays", []string{"Acme › Overlays", "Edit Profile", "Open menu", "Dimensions", "Open sheet", "Focus me", "Delete account"}, []string{"@peduarte", "My Account"}, nil},
		{"dialog", []string{"Edit profile", "@peduarte", "Save changes", "Cancel"}, nil, nil},
		{"menu-sub", []string{"My Account", "Invite users", "Email", "Message", "More..."}, []string{"@peduarte"}, nil},
		{"settings", []string{"Acme › Settings", "Appearance", "Palette", "neutral", "violet", "Dark mode", "Preview: zinc Light"}, []string{"My Account"}, map[string]bool{background("zinc-light", theme.Background): true, background("zinc-dark", theme.Background): false}},
		{"dark", []string{"Preview: zinc Dark"}, nil, map[string]bool{background("zinc-dark", theme.Background): true, background("zinc-light", theme.Background): false}},
		{"slate", []string{"Preview: slate Dark"}, nil, map[string]bool{background("slate-dark", theme.Primary): true, background("zinc-dark", theme.Primary): false}},
	} {
		read := func(ext string) string {
			b, err := os.ReadFile(filepath.Join(out, c.frame+ext))
			if err != nil {
				t.Fatal(err)
			}
			return string(b)
		}
		text := read(".txt")
		t.Logf("frame %s:\n%s", c.frame, text)
		for _, s := range slices.Concat(nav, c.shown) {
			if !strings.Contains(text, s) {
				t.Errorf("frame %s: no %q", c.frame, s)
			}
		}
		for _, s := range c.missing {
			if strings.Contains(text, s) {
				t.Errorf("frame %s: %q shown, want it gone", c.frame, s)
			}
		}
		lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
		if len(lines) != 34 {
			t.Errorf("frame %s has %d rows, want 34", c.frame, len(lines))
		}
		for i, l := range lines {
			if w := len([]rune(strings.TrimRight(l, " "))); w > 110 {
				t.Errorf("frame %s: row %d is %d cells wide", c.frame, i, w)
			}
		}
		ansi := read(".ansi")
		for seq, want := range c.ansi {
			if strings.Contains(ansi, seq) != want {
				t.Errorf("frame %s: background %s present is %t, want %t", c.frame, seq, !want, want)
			}
		}
	}
}

func TestLargeTerminalShowsTypeColumn(t *testing.T) {
	a, opts := app(t)
	for size, shown := range map[[2]int]bool{{110, 34}: false, {150, 40}: true} {
		d := drive.New(a, append(opts, drive.Size(size[0], size[1]))...)
		text := d.Frame().Text()
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(text, "Narrative") != shown {
			t.Errorf("at %dx%d the Type column shown is %t, want %t:\n%s", size[0], size[1], !shown, shown, text)
		}
	}
}

func TestQuitsOnQOutsideFieldsAndDialogs(t *testing.T) {
	a, opts := app(t)
	d := drive.New(a, opts...)
	for _, k := range []string{"down", "enter", "tab"} {
		d.Press(k)
	}
	d.Type("q")
	if !strings.Contains(d.Frame().Text(), "shadcnq") {
		t.Errorf("q did not reach the Username field:\n%s", d.Frame().Text())
	}
	for _, k := range []string{"shift+tab", "down", "enter", "tab", "enter", "tab", "tab", "q", "escape"} {
		d.Press(k)
	}
	if err := d.Err(); err != nil {
		t.Fatalf("q in the Username field or on a dialog button quit the app: %v", err)
	}
	d.Press("q")
	d.Press("x")
	if d.Err() == nil {
		t.Error("the app still takes keys after q outside the dialog")
	}
}

func TestParseRejectsUnknownFlags(t *testing.T) {
	for _, c := range [][3]string{{"zinc-dusk", "dashboard", ""}, {"zinc-dark", "home", ""}, {"zinc-dark", "overlays", "drawer"}} {
		if _, err := parse(c[0], c[1], c[2]); err == nil {
			t.Errorf("parse%q accepted", c)
		}
	}
	if s, err := parse("violet-dark", "Overlays", "menu-sub"); err != nil || s.page != overlays || s.theme.Name != "violet" || s.theme.Scheme != theme.Dark {
		t.Errorf("parse violet-dark Overlays menu-sub: %+v, %v", s, err)
	}
}

func percentiles(b *testing.B, samples []time.Duration) {
	slices.Sort(samples)
	b.ReportMetric(float64(samples[len(samples)/2].Microseconds())/1000, "p50-ms")
	b.ReportMetric(float64(samples[len(samples)*95/100].Microseconds())/1000, "p95-ms")
}

func BenchmarkFirstFrame(b *testing.B) {
	a, opts := app(b)
	samples := make([]time.Duration, 0, b.N)
	for range b.N {
		start := time.Now()
		d := drive.New(a, opts...)
		samples = append(samples, time.Since(start))
		b.StopTimer()
		if err := d.Close(); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
	percentiles(b, samples)
}

func BenchmarkPageSwitch(b *testing.B) {
	a, opts := app(b)
	d := drive.New(a, opts...)
	samples := make([]time.Duration, 0, b.N)
	b.ResetTimer()
	for i := range b.N {
		b.StopTimer()
		d.Press([...]string{"down", "down", "down", "home"}[i%4])
		b.StartTimer()
		start := time.Now()
		d.Press("enter")
		samples = append(samples, time.Since(start))
	}
	b.StopTimer()
	if err := d.Close(); err != nil {
		b.Fatal(err)
	}
	percentiles(b, samples)
}
