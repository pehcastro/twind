package app

import (
	"slices"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/pehcastro/twind/internal/buffer"
	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/tailwind"
	"github.com/pehcastro/twind/twi/theme"
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
	d := drive.New(App, drive.Size(100, 30), drive.With(twi.Styles(sheet)))
	t.Cleanup(func() {
		if err := d.Err(); err != nil {
			t.Error(err)
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	return d
}

func lines(d *drive.Driver) []string {
	return strings.Split(strings.TrimRight(d.Frame().Text(), "\n"), "\n")
}

func status(d *drive.Driver) string {
	all := lines(d)
	return strings.Join(strings.Fields(all[len(all)-1]), " ")
}

func inputLine(d *drive.Driver) string {
	for _, l := range lines(d) {
		if before, _, ok := strings.Cut(l, "tab focus"); ok {
			return strings.TrimSpace(strings.SplitN(before, "▌", 2)[1])
		}
	}
	return "no input line"
}

func cursorColumn(t *testing.T, d *drive.Driver) (int, string) {
	t.Helper()
	cells := d.Frame().Cells()
	foreground := theme.Default().Tokens[theme.Foreground].RGBA
	for y := range cells.Height() {
		row := cells.Row(y)
		start := -1
		for x, c := range row {
			if c.Grapheme == "▌" {
				start = x + 2
			}
			if start >= 0 && c.Bg.RGBA == foreground {
				return x - start, c.Grapheme
			}
		}
	}
	return -1, ""
}

func press(d *drive.Driver, keys ...string) {
	for _, k := range keys {
		d.Press(k)
	}
}

func run(d *drive.Driver, command string) {
	d.Type(command)
	d.Press("enter")
}

func TestTypeDeleteWordUndoThenTrapFocus(t *testing.T) {
	d := open(t)
	steps := []struct {
		act            func()
		input, focus   string
		cursor         int
		cursorGrapheme string
	}{
		{func() { d.Type("hello") }, "hello", "input", 5, " "},
		{func() { d.Press("ctrl+w") }, "Ask twind: a page, theme, dialog, toast, load, later or quit", "input", 0, " "},
		{func() { d.Type("world") }, "world", "input", 5, " "},
		{func() { d.Press("ctrl+z") }, "Ask twind: a page, theme, dialog, toast, load, later or quit", "input", 0, " "},
		{func() { d.Press("ctrl+z") }, "hello", "input", 5, " "},
		{func() { d.Type(" 中b"); press(d, "left", "left") }, "hello 中b", "input", 6, "中"},
		{func() { d.Press("right") }, "hello 中b", "input", 8, "b"},
		{func() { d.Press("ctrl+g") }, "hello 中b", "input", -1, ""},
		{func() { press(d, "tab", "tab", "tab", "tab", "tab", "tab", "tab", "tab", "tab") }, "hello 中b", "theme", -1, ""},
	}
	for i, s := range steps {
		s.act()
		if got := inputLine(d); got != s.input {
			t.Errorf("step %d: input shows %q, want %q", i, got, s.input)
		}
		if got := status(d); !strings.HasSuffix(got, "focus "+s.focus+" surfaces") {
			t.Errorf("step %d: status %q, want focus %s", i, got, s.focus)
		}
		if col, g := cursorColumn(t, d); col != s.cursor || g != s.cursorGrapheme {
			t.Errorf("step %d: cursor cell at column %d on %q, want %d on %q", i, col, g, s.cursor, s.cursorGrapheme)
		}
	}
	t.Logf("focus on the theme button:\n%s", d.Frame().Text())
	press(d, "enter")
	if text := d.Frame().Text(); !strings.Contains(text, "Enter keeps") || !strings.HasSuffix(status(d), "focus themes surfaces") {
		t.Fatalf("enter on the theme button did not open the picker with focus on the list:\n%s", text)
	}
	for _, want := range []string{"close", "themes", "close"} {
		press(d, "tab")
		if got := status(d); !strings.HasSuffix(got, "focus "+want+" surfaces") {
			t.Errorf("tab inside the picker: status %q, want focus %s", got, want)
		}
	}
	t.Logf("picker open, focus on close:\n%s", d.Frame().Text())
	press(d, "shift+tab", "4", "escape")
	if text := d.Frame().Text(); strings.Contains(text, "Enter keeps") || status(d) != "● fullscreen headless truecolor twind-dark focus theme surfaces" {
		t.Errorf("escape: want the picker closed, focus back on the theme button and no key leaked, status %q:\n%s", status(d), text)
	}
	t.Logf("after escape:\n%s", d.Frame().Text())
}

func TestPages(t *testing.T) {
	d := open(t)
	run(d, "layout")
	if got := status(d); !strings.HasSuffix(got, "layout") || inputLine(d) != "Ask twind: a page, theme, dialog, toast, load, later or quit" {
		t.Errorf("layout, enter: status %q, input %q", got, inputLine(d))
	}
	press(d, "tab", "enter")
	if got := status(d); !strings.HasSuffix(got, "focus surfaces surfaces") {
		t.Errorf("enter on the first pill: status %q", got)
	}
	press(d, "2", "tab", "space")
	if got := status(d); !strings.HasSuffix(got, "focus text text") {
		t.Errorf("2 then space on the second pill: status %q", got)
	}
}

func themeNames() []string {
	var names []string
	for _, t := range theme.Builtin() {
		if t.Scheme == theme.Light {
			names = append(names, t.Name)
		}
	}
	return names
}

func pickerNames(d *drive.Driver) []string {
	var shown []string
	in := false
	for _, l := range lines(d) {
		in = in && !strings.Contains(l, "close")
		for _, w := range strings.Fields(l) {
			if in && slices.Contains(themeNames(), w) {
				shown = append(shown, w)
			}
		}
		in = in || strings.Contains(l, "Enter keeps")
	}
	return shown
}

func pickerSpot(t *testing.T, d *drive.Driver, name string) (int, int) {
	t.Helper()
	all := lines(d)
	_, top := spot(t, d, "Enter keeps")
	for y := top + 1; y < len(all); y++ {
		if before, _, ok := strings.Cut(all[y], " "+name+" "); ok {
			return utf8.RuneCountInString(before) + 1, y
		}
	}
	t.Fatalf("no %s row in the picker:\n%s", name, d.Frame().Text())
	return 0, 0
}

func pickerRow(t *testing.T, d *drive.Driver, name string) buffer.Cell {
	t.Helper()
	x, y := pickerSpot(t, d, name)
	return d.Frame().Cells().Row(y)[x]
}

func builtinTheme(t *testing.T, name string) theme.Theme {
	t.Helper()
	for _, th := range theme.Builtin() {
		if themeName(th) == name {
			return th
		}
	}
	t.Fatalf("%s: not built in", name)
	return theme.Theme{}
}

func TestPickerOwnsKeys(t *testing.T) {
	d := open(t)
	run(d, "theme")
	if got := pickerNames(d); len(got) != 7 || !slices.Equal(got, themeNames()) || !strings.Contains(d.Frame().Text(), "● twind ") {
		t.Fatalf("picker lists %v, want the seven %v once each with twind marked:\n%s", got, themeNames(), d.Frame().Text())
	}
	press(d, "4", "+", "escape")
	if got := status(d); got != "● fullscreen headless truecolor twind-dark focus input surfaces" {
		t.Errorf("keys reached the page under the picker, or escape applied: status %q", got)
	}
	run(d, "t")
	press(d, "down", "enter")
	if got := status(d); !strings.Contains(got, "dream-dark") || strings.Contains(d.Frame().Text(), "Enter keeps") {
		t.Errorf("down, enter: want dream-dark applied and the picker closed, status %q", got)
	}
	run(d, "t")
	press(d, "up", "up", "escape")
	run(d, "t")
	press(d, "enter")
	if got := status(d); !strings.Contains(got, "dream-dark") {
		t.Errorf("reopened picker did not start on the applied theme: status %q", got)
	}
	run(d, "t")
	press(d, "up", "up")
	if got := pickerRow(t, d, "sukuna").Bg.RGBA; got != builtinTheme(t, "sukuna-dark").Tokens[theme.Accent].RGBA {
		t.Errorf("up twice from dream: sukuna, the last, is not the highlighted row (bg %v):\n%s", got, d.Frame().Text())
	}
}

func TestSchemeToggle(t *testing.T) {
	d := open(t)
	background := func() color.RGBA { return cellAt(t, d, "fullscreen").Bg.RGBA }
	if text := d.Frame().Text(); !strings.Contains(text, "☾") || strings.Contains(text, "☼") {
		t.Fatalf("dark at start: want the moon in the top bar and no sun:\n%s", text)
	}
	run(d, "scheme")
	if got := status(d); !strings.Contains(got, "twind-light") || background() != builtinTheme(t, "twind-light").Tokens[theme.Background].RGBA || !strings.Contains(d.Frame().Text(), "☼") {
		t.Errorf("scheme command: status %q, background %v, want twind-light with the sun:\n%s", got, background(), d.Frame().Text())
	}
	d.Click(spot(t, d, "☼"))
	if got := status(d); !strings.Contains(got, "twind-dark") || background() != builtinTheme(t, "twind-dark").Tokens[theme.Background].RGBA {
		t.Errorf("click on the sun: status %q, background %v, want twind-dark", got, background())
	}
	press(d, "t", "down", "enter")
	if got := status(d); !strings.Contains(got, "dream-dark") {
		t.Errorf("picking dream while dark: status %q, want dream-dark", got)
	}
	press(d, "m")
	if got := status(d); !strings.Contains(got, "dream-light") || background() != builtinTheme(t, "dream-light").Tokens[theme.Background].RGBA {
		t.Errorf("m off the input: status %q, want dream-light", got)
	}
	press(d, "t")
	if got := pickerNames(d); !slices.Equal(got, themeNames()) || !strings.Contains(d.Frame().Text(), "● dream ") {
		t.Errorf("picker in light lists %v with dream marked, want %v:\n%s", got, themeNames(), d.Frame().Text())
	}
	press(d, "escape")
	d.Click(spot(t, d, "Ask twind"))
	d.Type("m")
	if got := status(d); !strings.Contains(got, "dream-light") || inputLine(d) != "m" {
		t.Errorf("m typed in the input: status %q, input %q, want the scheme kept and the letter typed", got, inputLine(d))
	}
}

func spot(t *testing.T, d *drive.Driver, s string) (int, int) {
	t.Helper()
	for y, line := range lines(d) {
		if before, _, ok := strings.Cut(line, s); ok {
			return utf8.RuneCountInString(before), y
		}
	}
	t.Fatalf("no %q in the frame:\n%s", s, d.Frame().Text())
	return 0, 0
}

func cellAt(t *testing.T, d *drive.Driver, s string) buffer.Cell {
	t.Helper()
	x, y := spot(t, d, s)
	return d.Frame().Cells().Row(y)[x]
}

func TestPickerPreviews(t *testing.T) {
	d := open(t)
	run(d, "t")
	for step := 1; step <= 3; step++ {
		press(d, "down")
		name := themeNames()[step]
		want := builtinTheme(t, name+"-dark")
		if got := cellAt(t, d, "Theme").Bg.RGBA; got != want.Tokens[theme.Popover].RGBA {
			t.Errorf("down %d: the picker is drawn in %v, want %s's popover %v", step, got, themeName(want), want.Tokens[theme.Popover].RGBA)
		}
		if got := pickerRow(t, d, name).Bg.RGBA; got != want.Tokens[theme.Accent].RGBA {
			t.Errorf("down %d: the highlighted row is %v, want %s's accent %v", step, got, themeName(want), want.Tokens[theme.Accent].RGBA)
		}
		t.Logf("down %d, previewing %s:\n%s", step, themeName(want), d.Frame().Text())
	}
	press(d, "escape")
	if got, want := cellAt(t, d, "fullscreen").Bg.RGBA, theme.Default().Tokens[theme.Background].RGBA; got != want || !strings.Contains(status(d), "twind-dark") {
		t.Errorf("escape: the page is drawn in %v, want twind-dark's background %v back, status %q", got, want, status(d))
	}
	t.Logf("after escape:\n%s", d.Frame().Text())
}

func BenchmarkThemePreview(b *testing.B) {
	sheet, err := Styles()
	if err != nil {
		b.Fatal(err)
	}
	d := drive.New(App, drive.Size(100, 30), drive.With(twi.Styles(sheet)))
	run(d, "t")
	samples := make([]time.Duration, 0, b.N)
	b.ResetTimer()
	for range b.N {
		start := time.Now()
		d.Press("down")
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

func TestCounter(t *testing.T) {
	d := open(t)
	run(d, "counter")
	press(d, "shift+tab", "shift+tab")
	if got := status(d); !strings.HasSuffix(got, "focus scheme counter") {
		t.Fatalf("shift+tab from increment skips the decrement disabled at 0: status %q", got)
	}
	press(d, "tab", "enter", "enter", "shift+tab")
	if got := status(d); !strings.HasSuffix(got, "focus decrement counter") || !strings.Contains(d.Frame().Text(), "▀▀█") {
		t.Fatalf("want 2 after enter twice on increment and decrement enabled:\n%s", d.Frame().Text())
	}
	press(d, "enter", "enter", "enter", "1", "4")
	if text := d.Frame().Text(); !strings.Contains(text, "█"+nbsp+"█") {
		t.Errorf("want 0 after three presses of decrement from 2, and a page round trip:\n%s", text)
	}
}

func TestResizeReflows(t *testing.T) {
	d := open(t)
	run(d, "4")
	d.Resize(80, 24)
	text := d.Frame().Text()
	all := lines(d)
	if len(all) != 24 || !strings.Contains(all[1], "counter") || !strings.HasSuffix(status(d), "counter") || !strings.Contains(all[21], "Ask twind") {
		t.Errorf("80x24 frame lost the top bar, input bar or status line:\n%s", text)
	}
	for _, l := range all {
		if w := len([]rune(strings.TrimRight(l, " "))); w > 80 {
			t.Errorf("a line is %d runes wide after the resize: %q", w, l)
		}
	}
}

func TestHostileTextInert(t *testing.T) {
	d := open(t)
	run(d, "text")
	text := d.Frame().Text()
	if !strings.Contains(text, "clear bell bidiexe.txt") {
		t.Errorf("hostile string not shown as plain text:\n%s", text)
	}
	if i := strings.IndexFunc(text, func(r rune) bool { return r != '\n' && unicode.IsControl(r) || r == 0x202e }); i >= 0 {
		t.Errorf("a control or bidi character reached the screen at byte %d", i)
	}
}
