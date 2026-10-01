package docsapp

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"
	"unicode/utf8"

	"github.com/twind-dev/twind/apps/documentation/blocks"
	"github.com/twind-dev/twind/apps/documentation/components"
	"github.com/twind-dev/twind/docs"
	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/markdown"
	"github.com/twind-dev/twind/twi/tailwind"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/theme"
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

func fresh() *site { return newSite(twi.New(), components.All(), blocks.All()) }

func checked(s *site, pages fs.FS) error {
	if err := s.load(pages); err != nil {
		return err
	}
	for i := range s.entries {
		if err := s.parse(i); err != nil {
			return err
		}
	}
	return nil
}

func TestPages(t *testing.T) {
	s := fresh()
	if err := checked(s, docs.Pages); err != nil {
		t.Fatalf("the pages in docs/ do not load: %v", err)
	}
	shown := map[string]bool{}
	for _, e := range s.entries {
		if len(e.page.Blocks) < 2 || e.title == "" {
			t.Errorf("%s: %d blocks, title %q", e.slug, len(e.page.Blocks), e.title)
		}
		for _, of := range tags(e.page.Blocks, "Props", "of") {
			shown[of] = true
		}
	}
	for of := range s.props {
		if !shown[of] {
			t.Errorf("the props table of %s is on no page", of)
		}
	}
	var problem *markdown.Error
	err := checked(fresh(), pagesWith(t, "button.md", "# Button\n\nText.\n\n<Chart of=\"sales\" />\n"))
	if !errors.As(err, &problem) || problem.Problem != markdown.UnknownTag || problem.File != "button.md" || problem.Line != 5 {
		t.Errorf("a page with <Chart />: got %v, want an unknown tag at button.md:5", err)
	}
	err = checked(fresh(), pagesWith(t, "button.md", "# Button\n\n<Preview name=\"button-demo\" style=\"x\" />\n"))
	if !errors.As(err, &problem) || problem.Problem != markdown.UnknownAttribute || problem.Line != 3 {
		t.Errorf("a Preview with an unknown attribute: got %v, want an unknown attribute at line 3", err)
	}
	err = checked(fresh(), pagesWith(t, "button.md", "# Button\n\n<Props of=\"Gallery\" />\n"))
	if err == nil || !strings.Contains(err.Error(), "button.md:3") {
		t.Errorf("Props of a component with no table: got %v, want an error at button.md:3", err)
	}
	err = checked(fresh(), pagesWith(t, "orphan.md", "# Orphan\n"))
	if err == nil || !strings.Contains(err.Error(), "orphan.md") {
		t.Errorf("a page no sidebar entry opens: got %v", err)
	}
	err = checked(fresh(), pagesWith(t, "theming.md", "Colours.\n\n## Tokens\n"))
	if err == nil || !strings.Contains(err.Error(), "theming.md") {
		t.Errorf("a page with no title: got %v", err)
	}
	err = checked(fresh(), pagesWith(t, "card.md", "# Button\n\nA second Button.\n"))
	if err == nil || !strings.Contains(err.Error(), "card.md") {
		t.Errorf("two pages titled Button, which the palette cannot tell apart: got %v", err)
	}
}

func TestExamples(t *testing.T) {
	s := fresh()
	if err := checked(s, docs.Pages); err != nil {
		t.Fatal(err)
	}
	for dir, catalog := range map[string]components.Catalog{"components": components.All(), "blocks": blocks.All()} {
		for name := range catalog.Demos {
			p, shown := s.previews[name]
			if !shown {
				t.Errorf("demo %s is registered but no page shows it", name)
				continue
			}
			file := filepath.Join(dir, strings.ReplaceAll(name, "-", "_")+".go")
			want, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("demo %s: %v", name, err)
			}
			if p.source != string(want) || !strings.Contains(p.source, "func ") {
				t.Errorf("demo %s: the Code tab holds %d bytes, %s has %d", name, len(p.source), file, len(want))
			}
		}
	}
	missing := components.All()
	missing.Demos = maps.Clone(missing.Demos)
	delete(missing.Demos, "dialog-demo")
	err := checked(newSite(twi.New(), missing, blocks.All()), docs.Pages)
	if err == nil || !strings.Contains(err.Error(), "dialog.md:5") || !strings.Contains(err.Error(), "dialog-demo") {
		t.Errorf("a Preview of an unregistered demo: got %v, want an error at dialog.md:5", err)
	}
	hollow := components.All()
	names, err := fs.Glob(hollow.Source, "*.go")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	for _, name := range names {
		if name != "tabs_demo.go" {
			files[name] = &fstest.MapFile{}
		}
	}
	hollow.Source = files
	err = checked(newSite(twi.New(), hollow, blocks.All()), docs.Pages)
	if err == nil || !strings.Contains(err.Error(), "tabs.md:5") || !strings.Contains(err.Error(), "tabs_demo.go") {
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
	for part := range strings.SplitSeq(strings.Split(d.Frame().Text(), "\n")[y], "  ") {
		if strings.Contains(part, "Docs ›") {
			return strings.TrimSpace(part)
		}
	}
	return ""
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
	if got := breadcrumb(t, d); !strings.Contains(got, "Getting started › Installation") {
		t.Fatalf("tab past search, theme, scheme, the Getting started group and Introduction, then enter: breadcrumb %q\n%s", got, d.Frame().Text())
	}
	d.Click(spot(t, d, "Theming"))
	if got := breadcrumb(t, d); !strings.Contains(got, "Getting started › Theming") {
		t.Fatalf("a click on Theming in the sidebar: breadcrumb %q", got)
	}
	jump(t, d, "Button")
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

func jump(t *testing.T, d *drive.Driver, title string) {
	t.Helper()
	d.Press("ctrl+k")
	d.Type(title)
	d.Press("enter")
	d.Advance(settle)
	if got := breadcrumb(t, d); !strings.HasSuffix(got, "› "+title) {
		t.Fatalf("the palette on %q opened %q:\n%s", title, got, d.Frame().Text())
	}
}

func TestEveryPage(t *testing.T) {
	d := open(t)
	s := fresh()
	if err := s.load(docs.Pages); err != nil {
		t.Fatal(err)
	}
	logged := map[string]string{"layout": "", "motion": "", "card": "", "calendar": "", "dropdown-menu": "Open menu"}
	for _, e := range s.entries {
		jump(t, d, e.title)
		press, ok := logged[e.slug]
		if !ok {
			continue
		}
		if press != "" {
			x, y := spot(t, d, press)
			d.Click(x+1, y)
			d.Advance(settle)
		}
		t.Logf("%s through the palette:\n%s", e.title, d.Frame().Text())
	}
}

func TestSidebarScrolls(t *testing.T) {
	d := open(t)
	if strings.Contains(d.Frame().Text(), "Accordion") {
		t.Fatalf("the Components group starts open on the Introduction:\n%s", d.Frame().Text())
	}
	toComponents := 3 + 1 + 4 + 1 + 5 + 1
	for range toComponents {
		d.Press("tab")
	}
	d.Press("enter")
	for range 58 {
		d.Press("tab")
	}
	d.Press("enter")
	if got := breadcrumb(t, d); !strings.HasSuffix(got, "› Tooltip") {
		t.Fatalf("Tab to the last component, then Enter: breadcrumb %q\n%s", got, d.Frame().Text())
	}
	if !slices.ContainsFunc(strings.Split(d.Frame().Text(), "\n"), func(line string) bool { return strings.HasPrefix(strings.TrimSpace(line), "Tooltip ") }) {
		t.Errorf("the focused Tooltip entry is not in view in the sidebar:\n%s", d.Frame().Text())
	}
	t.Logf("tooltip through the sidebar:\n%s", d.Frame().Text())
}

func TestSelectText(t *testing.T) {
	d := open(t)
	x, y := spot(t, d, "Twind is a UI runtime")
	d.Down(x, y)
	d.Move(x+len("Twind is a UI runtime")-1, y)
	d.Up(x+len("Twind is a UI runtime")-1, y)
	d.Press("ctrl+c")
	if got := d.Clipboard(); got != "Twind is a UI runtime" {
		t.Fatalf("a drag over the first words of the introduction copied %q: twi/runtime/pointer.go refuses a selection that starts on a focusable element, and the page is a focusable ui.ScrollArea", got)
	}
	jump(t, d, "Button")
	x, y = spot(t, d, "Code")
	d.Click(x+1, y)
	x, y = spot(t, d, "package components")
	d.Down(x, y)
	d.Move(x+len("package components")-1, y)
	d.Up(x+len("package components")-1, y)
	d.Press("ctrl+c")
	if got := d.Clipboard(); got != "package components" {
		t.Errorf("a drag over the first line of the Code tab copied %q", got)
	}
}

func TestCopy(t *testing.T) {
	d := open(t)
	jump(t, d, "Button")
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

func TestCodeBlockCopy(t *testing.T) {
	d := open(t)
	jump(t, d, "Button")
	x, y := spot(t, d, "Copy")
	d.Click(x+1, y)
	if got, want := d.Clipboard(), `import "github.com/twind-dev/twind/twi/ui"`; got != want {
		t.Fatalf("the Usage block's copy area copied %q, want %q\n%s", got, want, d.Frame().Text())
	}
	if _, at := spot(t, d, "Copied"); at != y {
		t.Errorf("Copied shows on row %d, the clicked block is on row %d", at, y)
	}
	t.Logf("after the click:\n%s", d.Frame().Text())
	d.Advance(copiedFor)
	if strings.Contains(d.Frame().Text(), "Copied") {
		t.Errorf("the code block still says Copied %v later", copiedFor)
	}
}

var buttonSections = []string{"Usage", "Variants", "Sizes", "API reference"}

func column(d *drive.Driver, from, to int, text string) int {
	for y, line := range strings.Split(d.Frame().Text(), "\n") {
		if r := []rune(line); len(r) > from && strings.HasPrefix(strings.TrimSpace(string(r[from:min(to, len(r))])), text) {
			return y
		}
	}
	return -1
}

func onThisPage(t *testing.T, d *drive.Driver) (current string, rows []int) {
	t.Helper()
	x, _ := spot(t, d, "On this page")
	cells, seen := d.Frame().Cells(), map[string]int{}
	colours := make([]string, len(buttonSections))
	for i, s := range buttonSections {
		y := column(d, x, cells.Width(), s)
		if y < 0 {
			t.Fatalf("no %q under On this page:\n%s", s, d.Frame().Text())
		}
		rows = append(rows, y)
		for c := x; c < cells.Width(); c++ {
			if cell := cells.At(c, y); cell.Grapheme != " " {
				colours[i] = fmt.Sprint(cell.Fg, cell.Attr)
				break
			}
		}
		seen[colours[i]]++
	}
	for i, c := range colours {
		if seen[c] == 1 {
			current = buttonSections[i]
		}
	}
	return current, rows
}

func TestOnThisPageClickScrollsToTheHeading(t *testing.T) {
	d := open(t)
	jump(t, d, "Button")
	left, breadcrumb := spot(t, d, "Docs ›")
	right, _ := spot(t, d, "On this page")
	top := breadcrumb - 1
	got, rows := onThisPage(t, d)
	if got != "Usage" {
		t.Errorf("at the top of Button: current entry %q, want Usage", got)
	}
	t.Logf("before the click on Sizes:\n%s", d.Frame().Text())
	d.Click(right+1, rows[2])
	if got := column(d, left, right, "Sizes"); got != top {
		t.Errorf("a click on Sizes put its heading on row %d, want the content's top row %d:\n%s", got, top, d.Frame().Text())
	}
	if got, _ := onThisPage(t, d); got != "Sizes" {
		t.Errorf("after the click on Sizes: current entry %q, want Sizes", got)
	}
	t.Logf("after the click on Sizes:\n%s", d.Frame().Text())
	d.Click(right+1, rows[0])
	if got := column(d, left, right, "Usage"); got != top {
		t.Errorf("a click on Usage put its heading on row %d, want %d", got, top)
	}
	d.Move(left, top+10)
	d.Press("tab")
	d.Press("tab")
	d.Press("enter")
	if got, _ := onThisPage(t, d); got != "Sizes" || column(d, left, right, "Sizes") != top {
		t.Errorf("tab twice from Usage, then enter: current %q, Sizes on row %d, want Sizes on %d:\n%s", got, column(d, left, right, "Sizes"), top, d.Frame().Text())
	}
}

func TestOnThisPageFollowsTheWheel(t *testing.T) {
	d := open(t)
	jump(t, d, "Button")
	left, breadcrumb := spot(t, d, "Docs ›")
	right, _ := spot(t, d, "On this page")
	top := breadcrumb - 1
	for range 40 {
		if y := column(d, left, right, "Variants"); y >= 0 && y <= top {
			break
		}
		d.Wheel(left+2, top+2, 1)
	}
	if y := column(d, left, right, "Sizes"); y >= 0 && y <= top {
		t.Fatalf("one notch took Sizes past the top too; the check needs a finer step:\n%s", d.Frame().Text())
	}
	if got, _ := onThisPage(t, d); got != "Variants" {
		t.Errorf("wheel down until Variants reaches the top: current entry %q, want Variants:\n%s", got, d.Frame().Text())
	}
	d.Wheel(left+2, top+2, -40)
	current, rows := onThisPage(t, d)
	if current != "Usage" {
		t.Errorf("wheel back to the top: current entry %q, want Usage", current)
	}
	d.Click(right+1, rows[3])
	d.Move(left, top+10)
	if got, _ := onThisPage(t, d); got != "API reference" {
		t.Errorf("a click on API reference, which the page cannot scroll to the top: current entry %q, want API reference:\n%s", got, d.Frame().Text())
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

func highlightedSidebarRows(t *testing.T, d *drive.Driver) []string {
	t.Helper()
	crumb, y := spot(t, d, "Docs ›")
	right := slices.Index([]rune(strings.Split(d.Frame().Text(), "\n")[y])[:crumb], '│')
	_, header := spot(t, d, "Search documentation")
	type row struct {
		text string
		bg   color.Color
	}
	cells, count := d.Frame().Cells(), map[color.Color]int{}
	var rows []row
	for y := header + 2; y < cells.Height(); y++ {
		var text strings.Builder
		for x := range right {
			text.WriteString(cells.At(x, y).Grapheme)
		}
		if lead := len(text.String()) - len(strings.TrimLeft(text.String(), " ")); lead < right {
			rows = append(rows, row{strings.TrimSpace(text.String()), cells.At(lead, y).Bg})
			count[cells.At(lead, y).Bg]++
		}
	}
	var plain color.Color
	for bg, n := range count {
		if n > count[plain] {
			plain = bg
		}
	}
	var lit []string
	for _, r := range rows {
		if r.bg != plain {
			lit = append(lit, r.text)
		}
	}
	return lit
}

func TestPaletteMovesTheSidebarHighlight(t *testing.T) {
	d := open(t)
	d.Click(spot(t, d, "Installation"))
	if got := highlightedSidebarRows(t, d); !slices.Equal(got, []string{"Installation"}) {
		t.Fatalf("after a click on Installation: highlighted sidebar rows %q\n%s", got, d.Frame().Text())
	}
	d.Press("ctrl+k")
	d.Type("card")
	d.Advance(settle)
	d.Press("enter")
	d.Advance(settle)
	if got := breadcrumb(t, d); !strings.HasSuffix(got, "› Card") {
		t.Fatalf("Ctrl+K, card, Enter: breadcrumb %q\n%s", got, d.Frame().Text())
	}
	if got := highlightedSidebarRows(t, d); !slices.Equal(got, []string{"Card"}) {
		t.Errorf("Ctrl+K, card, Enter: highlighted sidebar rows %q, want only Card\n%s", got, d.Frame().Text())
	}
	t.Logf("card through the palette:\n%s", d.Frame().Text())
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

func pickerNames(t *testing.T, d *drive.Driver) []string {
	t.Helper()
	x, y := spot(t, d, "↑ ↓ preview")
	var names []string
	for _, l := range strings.Split(d.Frame().Text(), "\n")[y+1:] {
		if strings.Contains(l, "╰") {
			return names
		}
		for _, f := range strings.Fields(string([]rune(l)[x:])) {
			if slices.ContainsFunc(theme.Builtin(), func(th theme.Theme) bool { return th.Name == f }) {
				names = append(names, f)
			}
		}
	}
	return names
}

func TestThemePicker(t *testing.T) {
	d := open(t)
	d.Press("t")
	if got, want := pickerNames(t, d), []string{"twind", "dream", "mono", "minimal", "dew", "cloud", "sukuna"}; !slices.Equal(got, want) {
		t.Fatalf("picker lists %v, want the seven %v:\n%s", got, want, d.Frame().Text())
	}
	d.Press("down")
	if text := d.Frame().Text(); !strings.Contains(text, "● twind") || !strings.Contains(text, "Theme: twind-dark") {
		t.Fatalf("down in the picker: want twind-dark still applied and marked:\n%s", text)
	}
	d.Press("escape")
	if text := d.Frame().Text(); strings.Contains(text, "Enter keeps") || !strings.Contains(text, "Theme: twind-dark") {
		t.Fatalf("escape: want the picker closed and twind-dark kept:\n%s", text)
	}
	d.Press("t")
	d.Press("up")
	d.Press("enter")
	if text := d.Frame().Text(); !strings.Contains(text, "Theme: sukuna-dark") {
		t.Errorf("up, enter from twind-dark: want sukuna-dark applied:\n%s", text)
	}
}

func TestSchemeToggle(t *testing.T) {
	d := open(t)
	if text := d.Frame().Text(); !strings.Contains(text, "☾") || strings.Contains(text, "☼") {
		t.Fatalf("dark at start: want the moon in the top bar and no sun:\n%s", text)
	}
	d.Press("m")
	if text := d.Frame().Text(); !strings.Contains(text, "Theme: twind-light") || !strings.Contains(text, "☼") {
		t.Fatalf("m: want twind-light and the sun:\n%s", text)
	}
	d.Press("t")
	d.Press("down")
	d.Press("enter")
	if text := d.Frame().Text(); !strings.Contains(text, "Theme: dream-light") {
		t.Fatalf("dream picked while light: want dream-light:\n%s", text)
	}
	d.Click(spot(t, d, "☼"))
	if text := d.Frame().Text(); !strings.Contains(text, "Theme: dream-dark") || !strings.Contains(text, "☾") {
		t.Fatalf("a click on the sun: want dream-dark and the moon:\n%s", text)
	}
	d.Press("ctrl+k")
	d.Type("sukuna")
	if n := strings.Count(d.Frame().Text(), "sukuna"); n != 2 {
		t.Errorf("search for sukuna: %d matches on screen, want the query and one Theme item:\n%s", n, d.Frame().Text())
	}
	d.Press("enter")
	if text := d.Frame().Text(); !strings.Contains(text, "Theme: sukuna-dark") {
		t.Errorf("sukuna from the search while dark: want sukuna-dark:\n%s", text)
	}
}

func uiComponents(t *testing.T) []string {
	files, err := filepath.Glob(filepath.Join("..", "..", "twi", "ui", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var components []string
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		var roots, parts []string
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !fn.Name.IsExported() {
				continue
			}
			if root, ok := strings.CutPrefix(fn.Name.Name, "New"); ok {
				roots = append(roots, root)
				continue
			}
			if r := fn.Type.Results; r != nil && len(r.List) == 1 {
				if sel, ok := r.List[0].Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Node" {
					parts = append(parts, fn.Name.Name)
				}
			}
		}
		slices.SortFunc(parts, func(a, b string) int { return len(a) - len(b) })
		for _, p := range parts {
			if !slices.ContainsFunc(roots, func(r string) bool { return strings.HasPrefix(p, r) }) {
				roots = append(roots, p)
			}
		}
		components = append(components, roots...)
	}
	slices.Sort(components)
	return components
}

func tags(blocks []markdown.Block, name, attr string) []string {
	var found []string
	for _, b := range blocks {
		if b.Kind == markdown.Tag && b.Name == name {
			found = append(found, b.Attrs[attr])
		}
		for _, nested := range append([][]markdown.Block{b.Children}, b.Items...) {
			found = append(found, tags(nested, name, attr)...)
		}
	}
	return found
}

func TestEveryComponent(t *testing.T) {
	s := fresh()
	if err := checked(s, docs.Pages); err != nil {
		t.Fatal(err)
	}
	documented := map[string]bool{}
	for _, e := range s.entries {
		if e.group != "Components" {
			continue
		}
		previews, props := tags(e.page.Blocks, "Preview", "name"), tags(e.page.Blocks, "Props", "of")
		if len(previews) == 0 || len(props) == 0 {
			t.Errorf("%s.md: %d previews and %d props tables, want at least one of each", e.slug, len(previews), len(props))
		}
		for _, p := range props {
			documented[p] = true
		}
	}
	found := uiComponents(t)
	if len(found) < 45 {
		t.Fatalf("found %d twi/ui components, want the whole kit: %v", len(found), found)
	}
	for _, c := range found {
		if !documented[c] {
			t.Errorf("ui.%s has no docs page: no Components page carries <Props of=%q />", c, c)
		}
	}
	t.Logf("%d components: %s", len(found), strings.Join(found, ", "))
}
