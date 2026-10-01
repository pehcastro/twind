package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
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
	s, err := parse("twind-light", "dashboard", "")
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
		{"settings", []string{"Acme › Settings", "Appearance", "Palette", "twind", "sukuna", "Dark mode", "Preview: twind Light"}, []string{"My Account", "zinc", "violet"}, map[string]bool{background("twind-light", theme.Background): true, background("twind-dark", theme.Background): false}},
		{"dark", []string{"Preview: twind Dark"}, nil, map[string]bool{background("twind-dark", theme.Background): true, background("twind-light", theme.Background): false}},
		{"dream", []string{"Preview: dream Dark"}, nil, map[string]bool{background("dream-dark", theme.Primary): true, background("twind-dark", theme.Primary): false}},
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

func find(t *testing.T, d *drive.Driver, s string, from, to int) (int, int) {
	t.Helper()
	for y, line := range strings.Split(d.Frame().Text(), "\n") {
		r := []rune(line)
		if before, _, ok := strings.Cut(string(r[min(from, len(r)):min(to, len(r))]), s); ok {
			return from + utf8.RuneCountInString(before), y
		}
	}
	t.Fatalf("no %q in columns %d to %d:\n%s", s, from, to, d.Frame().Text())
	return 0, 0
}

func inSidebar(t *testing.T, d *drive.Driver, s string) (int, int) {
	t.Helper()
	return find(t, d, s, 0, sidebarWidth)
}

func inMain(t *testing.T, d *drive.Driver, s string) (int, int) {
	t.Helper()
	return find(t, d, s, sidebarWidth, math.MaxInt)
}

func sidebarClean(t *testing.T, d *drive.Driver, what string) {
	t.Helper()
	_, top := inSidebar(t, d, "Dashboard")
	_, bottom := inSidebar(t, d, "Settings")
	rows := strings.Split(d.Frame().Text(), "\n")
	for y := top - 1; y <= bottom+1; y++ {
		if nav := string([]rune(rows[y])[:sidebarWidth]); strings.ContainsAny(nav, "▁▔▕▏") {
			t.Errorf("%s: a highlight paints on row %d of the sidebar: %q", what, y, nav)
		}
	}
}

const sidebarWidth = 19

func TestClicks(t *testing.T) {
	a, opts := app(t)
	d := drive.New(a, opts...)
	defer func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}()
	d.Press("down")
	sidebarClean(t, d, "down from Dashboard")
	t.Logf("keyboard cursor on Forms:\n%s", d.Frame().Text())
	for _, name := range []string{"Settings", "Overlays", "Forms", "Dashboard"} {
		d.Click(inSidebar(t, d, name))
		if text := d.Frame().Text(); !strings.Contains(text, "Acme › "+name) {
			t.Errorf("a click on %s did not open it:\n%s", name, text)
		}
		sidebarClean(t, d, "a click on "+name)
	}
	t.Logf("after the clicks, back on Dashboard:\n%s", d.Frame().Text())
	for _, doc := range []string{"Data Library", "Reports", "Assistant"} {
		d.Click(inSidebar(t, d, doc))
		if text := d.Frame().Text(); !strings.Contains(text, doc+" opens in the full app") {
			t.Errorf("a click on %s showed nothing:\n%s", doc, text)
		}
	}
	d.Press("escape")
	d.Press("escape")
	d.Press("escape")
	x, y := inMain(t, d, "Previous")
	d.Click(x+slices.Index([]rune(strings.Split(d.Frame().Text(), "\n")[y])[x:], '2'), y)
	if text := d.Frame().Text(); !strings.Contains(text, "Page 2 of 3") || !strings.Contains(text, "Capabilities") {
		t.Errorf("a click on page 2 did not turn the table:\n%s", text)
	}
	d.Click(inMain(t, d, "3m"))
	if text := d.Frame().Text(); !strings.Contains(text, "Last 3 months") {
		t.Errorf("a click on the card action 3m did not change the chart:\n%s", text)
	}
	x, y = inMain(t, d, "Capabilities")
	d.Click(x, y)
	d.Move(0, 0)
	light, _ := builtin("twind-light")
	if got, want := d.Frame().Cells().Row(y)[x+len("Capabilities")+1].Bg.RGBA, light.Tokens[theme.Muted].RGBA; got != want {
		t.Errorf("a click on a row did not select it: background %v, want muted %v", got, want)
	}
	t.Logf("page 2, 3m and the Capabilities row selected:\n%s", d.Frame().Text())
}

func TestSettingsPreview(t *testing.T) {
	a, opts := app(t)
	d := drive.New(a, opts...)
	defer func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}()
	page := func() color.RGBA {
		x, y := inMain(t, d, "Primary")
		return d.Frame().Cells().Row(y)[x].Bg.RGBA
	}
	shows := func(name string) color.RGBA {
		th, _ := builtin(name)
		return th.Tokens[theme.Primary].RGBA
	}
	d.Click(inSidebar(t, d, "Settings"))
	d.Move(inMain(t, d, "sukuna"))
	if got := page(); got != shows("sukuna-light") {
		t.Errorf("hovering sukuna: the page is %v, want sukuna-light's %v", got, shows("sukuna-light"))
	}
	d.Move(inMain(t, d, "Twind gallery"))
	if got := page(); got != shows("twind-light") {
		t.Errorf("leaving the palette: the page is %v, want twind-light's %v back", got, shows("twind-light"))
	}
	d.Press("tab")
	d.Press("down")
	d.Press("down")
	if text := d.Frame().Text(); !strings.Contains(text, "Preview: mono Light") {
		t.Errorf("down twice from twind did not choose mono:\n%s", text)
	}
	d.Press("escape")
	if text := d.Frame().Text(); !strings.Contains(text, "Preview: twind Light") || page() != shows("twind-light") {
		t.Errorf("escape did not bring twind back:\n%s", text)
	}
}

func TestSchemeToggle(t *testing.T) {
	a, opts := app(t)
	d := drive.New(a, opts...)
	defer func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}()
	canvas := func() color.RGBA {
		x, y := inMain(t, d, "Twind gallery")
		return d.Frame().Cells().Row(y)[x].Bg.RGBA
	}
	want := func(name string) color.RGBA {
		th, _ := builtin(name)
		return th.Tokens[theme.Background].RGBA
	}
	if text := d.Frame().Text(); !strings.Contains(text, "☼") || strings.Contains(text, "☾") || canvas() != want("twind-light") {
		t.Fatalf("light at start: want the sun in the top bar and twind-light:\n%s", text)
	}
	d.Click(inMain(t, d, "☼"))
	if text := d.Frame().Text(); !strings.Contains(text, "☾") || canvas() != want("twind-dark") {
		t.Errorf("a click on the sun: want the moon and twind-dark, canvas %v:\n%s", canvas(), text)
	}
	d.Click(inSidebar(t, d, "Settings"))
	d.Click(inMain(t, d, "dream"))
	if text := d.Frame().Text(); !strings.Contains(text, "Preview: dream Dark") || canvas() != want("dream-dark") {
		t.Errorf("dream picked while dark: want dream-dark, canvas %v:\n%s", canvas(), text)
	}
	d.Click(inSidebar(t, d, "Dashboard"))
	d.Press("m")
	if text := d.Frame().Text(); !strings.Contains(text, "☼") || canvas() != want("dream-light") {
		t.Errorf("m: want the sun and dream-light, canvas %v:\n%s", canvas(), text)
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

func TestFormsPlanSelectAndSave(t *testing.T) {
	a, opts := app(t)
	d := drive.New(a, opts...)
	defer func() { _ = d.Close() }()
	for _, k := range []string{"down", "enter", "tab", "tab", "tab", "tab", "enter"} {
		d.Press(k)
	}
	d.Advance(300 * time.Millisecond)
	if text := d.Frame().Text(); !strings.Contains(text, "Enterprise") || !strings.Contains(text, "Free") {
		t.Errorf("enter on the plan trigger did not open its list:\n%s", text)
	}
	d.Press("down")
	d.Press("enter")
	d.Advance(300 * time.Millisecond)
	if text := d.Frame().Text(); !strings.Contains(text, "Team") || strings.Contains(text, "Enterprise") {
		t.Errorf("down and enter did not choose Team and close:\n%s", text)
	}
	for y, line := range strings.Split(d.Frame().Text(), "\n") {
		if x := strings.Index(line, "Save changes"); x >= 0 {
			d.Click(len([]rune(line[:x])), y)
		}
	}
	if text := d.Frame().Text(); !strings.Contains(text, "Profile updated.") {
		t.Errorf("a click on Save changes did not save:\n%s", text)
	}
}

func TestParseRejectsUnknownFlags(t *testing.T) {
	for _, c := range [][3]string{{"zinc-dark", "dashboard", ""}, {"twind-dusk", "dashboard", ""}, {"twind-dark", "home", ""}, {"twind-dark", "overlays", "drawer"}} {
		if _, err := parse(c[0], c[1], c[2]); err == nil {
			t.Errorf("parse%q accepted", c)
		}
	}
	if s, err := parse("sukuna-dark", "Overlays", "menu-sub"); err != nil || s.page != overlays || s.theme.Name != "sukuna" || s.theme.Scheme != theme.Dark {
		t.Errorf("parse sukuna-dark Overlays menu-sub: %+v, %v", s, err)
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
