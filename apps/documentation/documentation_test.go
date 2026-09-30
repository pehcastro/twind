package docsapp

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"
	"unicode/utf8"

	"github.com/twind-dev/twind/apps/documentation/components"
	"github.com/twind-dev/twind/docs"
	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/markdown"
	"github.com/twind-dev/twind/twi/tailwind"
	"github.com/twind-dev/twind/twi/terminal"
)

func TestStylesFresh(t *testing.T) {
	stale, err := tailwind.Stale(".", konst.GeneratedFile)
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Errorf("%s is stale against the classes it compiles: run twind build ./apps/documentation", konst.GeneratedFile)
	}
}

func pagesWith(t *testing.T, file, text string) fs.FS {
	t.Helper()
	pages := fstest.MapFS{}
	err := fs.WalkDir(docs.Pages, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		src, err := fs.ReadFile(docs.Pages, name)
		pages[name] = &fstest.MapFile{Data: src}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if file != "" {
		pages[file] = &fstest.MapFile{Data: []byte(text)}
	}
	return pages
}

func TestPages(t *testing.T) {
	s := newSite(twi.New(), components.All(), components.Source)
	if err := s.load(docs.Pages); err != nil {
		t.Fatalf("the pages in docs/ do not load: %v", err)
	}
	for _, e := range s.entries {
		if len(e.page.Blocks) < 2 || e.title == "" {
			t.Errorf("%s: %d blocks, title %q", e.slug, len(e.page.Blocks), e.title)
		}
	}
	var problem *markdown.Error
	err := newSite(twi.New(), components.All(), components.Source).load(pagesWith(t, "button.md", "# Button\n\nText.\n\n<Chart of=\"sales\" />\n"))
	if !errors.As(err, &problem) || problem.Problem != markdown.UnknownTag || problem.File != "button.md" || problem.Line != 5 {
		t.Errorf("a page with <Chart />: got %v, want an unknown tag at button.md:5", err)
	}
	err = newSite(twi.New(), components.All(), components.Source).load(pagesWith(t, "button.md", "# Button\n\n<Preview name=\"button-demo\" style=\"x\" />\n"))
	if !errors.As(err, &problem) || problem.Problem != markdown.UnknownAttribute || problem.Line != 3 {
		t.Errorf("a Preview with an unknown attribute: got %v, want an unknown attribute at line 3", err)
	}
	err = newSite(twi.New(), components.All(), components.Source).load(pagesWith(t, "button.md", "# Button\n\n<Props of=\"Slider\" />\n"))
	if err == nil || !strings.Contains(err.Error(), "button.md:3") {
		t.Errorf("Props of a component with no table: got %v, want an error at button.md:3", err)
	}
	err = newSite(twi.New(), components.All(), components.Source).load(pagesWith(t, "orphan.md", "# Orphan\n"))
	if err == nil || !strings.Contains(err.Error(), "orphan.md") {
		t.Errorf("a page no sidebar entry opens: got %v", err)
	}
	err = newSite(twi.New(), components.All(), components.Source).load(pagesWith(t, "theming.md", "Colours.\n\n## Tokens\n"))
	if err == nil || !strings.Contains(err.Error(), "theming.md") {
		t.Errorf("a page with no title: got %v", err)
	}
}

func TestExamples(t *testing.T) {
	s := newSite(twi.New(), components.All(), components.Source)
	if err := s.load(docs.Pages); err != nil {
		t.Fatal(err)
	}
	for name, demo := range components.All() {
		p, shown := s.previews[name]
		if !shown {
			t.Errorf("component demo %s is registered but no page shows it", name)
			continue
		}
		want, err := os.ReadFile("components/" + demo.File)
		if err != nil {
			t.Fatalf("component demo %s: %v", name, err)
		}
		if p.source != string(want) || !strings.Contains(p.source, "func ") {
			t.Errorf("component demo %s: the Code tab holds %d bytes, %s has %d", name, len(p.source), demo.File, len(want))
		}
	}
	missing := maps.Clone(components.All())
	delete(missing, "dialog-demo")
	err := newSite(twi.New(), missing, components.Source).load(docs.Pages)
	if err == nil || !strings.Contains(err.Error(), "dialog.md:5") || !strings.Contains(err.Error(), "dialog-demo") {
		t.Errorf("a Preview of an unregistered demo: got %v, want an error at dialog.md:5", err)
	}
	renamed := maps.Clone(components.All())
	renamed["tabs-demo"] = components.Demo{File: "tabs.go", New: renamed["tabs-demo"].New}
	err = newSite(twi.New(), renamed, components.Source).load(docs.Pages)
	if err == nil || !strings.Contains(err.Error(), "tabs.md:5") || !strings.Contains(err.Error(), "tabs.go") {
		t.Errorf("a demo whose source file is not embedded: got %v, want an error at tabs.md:5", err)
	}
}

const settle = 500 * time.Millisecond

func open(t *testing.T) *drive.Driver {
	t.Helper()
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	d := drive.New(App, drive.Size(120, 36), drive.Styles(sheet))
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

func spot(t *testing.T, d *drive.Driver, s string) (int, int) {
	t.Helper()
	for y, line := range strings.Split(d.Frame().Text(), "\n") {
		if before, _, ok := strings.Cut(line, s); ok {
			return utf8.RuneCountInString(before), y
		}
	}
	t.Fatalf("no %q in the frame:\n%s", s, d.Frame().Text())
	return 0, 0
}

func breadcrumb(t *testing.T, d *drive.Driver) string {
	t.Helper()
	_, y := spot(t, d, "Docs ›")
	return strings.Join(strings.Fields(strings.Split(d.Frame().Text(), "\n")[y]), " ")
}

func TestTour(t *testing.T) {
	d := open(t)
	if got := breadcrumb(t, d); !strings.Contains(got, "Docs › Getting started › Introduction") {
		t.Fatalf("first page: breadcrumb %q", got)
	}
	t.Logf("introduction:\n%s", d.Frame().Text())
	for range 6 {
		d.Press("tab")
	}
	d.Press("enter")
	if got := breadcrumb(t, d); !strings.Contains(got, "Components › Button") {
		t.Fatalf("tab past search, theme and four pages, then enter: breadcrumb %q\n%s", got, d.Frame().Text())
	}
	t.Logf("button through the keyboard:\n%s", d.Frame().Text())
	d.Click(spot(t, d, "Dialog"))
	d.Click(spot(t, d, "Button"))
	if got := breadcrumb(t, d); !strings.Contains(got, "Components › Button") {
		t.Fatalf("a click on Dialog then Button in the sidebar: breadcrumb %q", got)
	}
	x, y := spot(t, d, "Code")
	d.Click(x+1, y)
	if !strings.Contains(d.Frame().Text(), "func ButtonDemo(*twi.Runtime) func() twi.Node {") {
		t.Fatalf("Code tab does not show button_demo.go:\n%s", d.Frame().Text())
	}
	t.Logf("code tab:\n%s", d.Frame().Text())
	d.Press("ctrl+k")
	d.Type("dialog")
	d.Advance(settle)
	t.Logf("palette:\n%s", d.Frame().Text())
	d.Press("enter")
	d.Advance(settle)
	if got := breadcrumb(t, d); !strings.Contains(got, "Components › Dialog") || strings.Contains(d.Frame().Text(), "⌕") {
		t.Fatalf("palette jump: breadcrumb %q\n%s", got, d.Frame().Text())
	}
	t.Logf("dialog through the palette:\n%s", d.Frame().Text())
}

func TestCopy(t *testing.T) {
	d := open(t)
	d.Press("ctrl+k")
	d.Type("button")
	d.Press("enter")
	d.Advance(settle)
	x, y := spot(t, d, "Code")
	d.Click(x+1, y)
	x, y = spot(t, d, "Copy")
	d.Click(x+1, y)
	want, err := os.ReadFile("components/button_demo.go")
	if err != nil {
		t.Fatal(err)
	}
	osc := terminal.Clipboard(string(want))
	t.Logf("the OSC 52 write for button_demo.go, %d bytes: %q ... %q\ndecoded by the driver's screen, %d bytes:\n%s", len(osc), osc[:24], osc[len(osc)-8:], len(d.Clipboard()), d.Clipboard())
	if !strings.Contains(d.Frame().Text(), "Copied") {
		t.Errorf("copy: the button does not say Copied:\n%s", d.Frame().Text())
	}
	d.Advance(copiedFor)
	if strings.Contains(d.Frame().Text(), "Copied") {
		t.Errorf("copy: the button still says Copied %v later", copiedFor)
	}
	if got := d.Clipboard(); got != string(want) {
		t.Errorf("copy: the OSC 52 payload is %q, want button_demo.go byte for byte (%d bytes)", got, len(want))
	}
}

func TestLink(t *testing.T) {
	d := open(t)
	x, y := spot(t, d, "read Installation")
	d.Click(x+len("read "), y)
	if got := breadcrumb(t, d); !strings.Contains(got, "Getting started › Installation") {
		t.Errorf("a click on the Installation link: breadcrumb %q", got)
	}
}

func BenchmarkFirstFrame(b *testing.B) {
	sheet, err := Styles()
	if err != nil {
		b.Fatal(err)
	}
	for range b.N {
		if err := drive.New(App, drive.Size(120, 36), drive.Styles(sheet)).Close(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestThemePicker(t *testing.T) {
	d := open(t)
	d.Press("t")
	d.Press("down")
	if text := d.Frame().Text(); !strings.Contains(text, "● zinc-dark") || !strings.Contains(text, "◐ zinc-dark") {
		t.Fatalf("down in the picker: want zinc-dark still applied and marked:\n%s", text)
	}
	d.Press("escape")
	if text := d.Frame().Text(); strings.Contains(text, "Enter keeps") || !strings.Contains(text, "◐ zinc-dark") {
		t.Fatalf("escape: want the picker closed and zinc-dark kept:\n%s", text)
	}
	d.Press("t")
	d.Press("up")
	d.Press("enter")
	if text := d.Frame().Text(); !strings.Contains(text, "◐ zinc-light") {
		t.Errorf("up, enter from zinc-dark: want zinc-light applied:\n%s", text)
	}
}
