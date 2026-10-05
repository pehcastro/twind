package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/terminal"
)

func TestTour(t *testing.T) {
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join("testdata", "tour.twd"))
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := drive.RunScript(bytes.NewReader(src), App, out, drive.Styles(sheet)); err != nil {
		t.Fatal(err)
	}
	read := func(name, ext string) string {
		b, err := os.ReadFile(filepath.Join(out, name+ext))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	text := func(name string) string { return read(name, ".txt") }
	ansi := func(name string) string { return read(name, ".ansi") }
	for _, name := range []string{"surfaces", "text", "layout", "counter", "list", "motion", "selection"} {
		all := strings.Split(strings.TrimRight(text(name), "\n"), "\n")
		if !strings.HasSuffix(strings.TrimSpace(all[len(all)-1]), " "+name) {
			t.Errorf("%s: the tour is on another page:\n%s", name, text(name))
		}
		t.Logf("%s:\n%s", name, text(name))
	}
	expect := func(what string, ok bool, frames ...string) {
		t.Helper()
		if ok {
			return
		}
		t.Errorf("%s", what)
		for _, f := range frames {
			t.Logf("%s:\n%s", f, text(f))
		}
	}
	moving := func(what string, frames ...string) {
		t.Helper()
		for i := 1; i < len(frames); i++ {
			expect(what+": "+frames[i-1]+" and "+frames[i]+" are the same colours", ansi(frames[i-1]) != ansi(frames[i]), frames[i-1], frames[i])
		}
	}
	hover := []string{"hover-017ms", "hover-100ms", "hover-200ms"}
	moving("hover", hover...)
	expect("hover: only colours move", text("motion") == text(hover[2]), "motion", hover[2])
	open := []string{"hover-200ms", "open-017ms", "open-040ms", "open-075ms", "open-150ms"}
	dialog := "plays it back out"
	moving("open", open[0], open[1], open[3], open[4])
	expect("open: the panel fades in at its settled cells, zoom-in-95 moves no cell, and it has settled at 150 ms", text(open[3]) == text(open[4]) && strings.Contains(text(open[4]), dialog), open[3], open[4])
	closing := []string{"open-150ms", "close-017ms", "close-040ms", "close-075ms", "close-150ms"}
	moving("close", closing...)
	expect("close: gone after the 100 ms exit", !strings.Contains(text(closing[4]), dialog), closing[4])
	toast := "Saved to the playground"
	expect("toast: shown until 4 s, gone after", strings.Contains(text("toast-017ms"), toast) && strings.Contains(text("toast-3917ms"), toast) && !strings.Contains(text("toast-4017ms"), toast), "toast-017ms", "toast-3917ms", "toast-4017ms")
	loaded := "loaded, still"
	expect("load: a skeleton for 2 s, then the content", !strings.Contains(text("load-017ms"), loaded) && strings.Contains(text("load-2017ms"), loaded), "load-017ms", "load-2017ms")
	expect("selection: the drag lights cells without changing the text", text("selection") == text("selected") && ansi("selection") != ansi("selected"), "selected")
	for _, f := range [][]string{hover, open, closing, {"toast-017ms", "toast-4017ms", "load-017ms", "load-2017ms", "selected"}} {
		for _, name := range f {
			t.Logf("%s:\n%s", name, text(name))
		}
	}

	d := drive.New(App, drive.Size(100, 30), drive.Styles(sheet))
	defer func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}()
	run(d, "selection")
	d.Down(6, 12)
	d.Move(16, 13)
	d.Move(22, 14)
	d.Up(22, 14)
	d.Press("ctrl+c")
	if err := d.Err(); err != nil {
		t.Fatal(err)
	}
	if got, want := d.Clipboard(), "Application code thinks in nodes, classes, events and focus."; got != want {
		t.Errorf("ctrl+c copied %q, want %q", got, want)
	}
	t.Logf("OSC 52 decoded by the driver's screen, re-encoded: %q", terminal.Clipboard(d.Clipboard()))
}
