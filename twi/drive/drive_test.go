package drive_test

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/runtime/testdata/hover"
	"github.com/twind-dev/twind/twi/testdata/counter"
	"github.com/twind-dev/twind/twi/testdata/hello"
)

func TestCounter(t *testing.T) {
	d := drive.New(counter.New, drive.Size(80, 24))
	for range 3 {
		d.Press("+")
	}
	if text := d.Frame().Text(); !strings.Contains(text, "count 3") {
		t.Errorf("after three presses of +:\n%s", text)
	}
	d.Resize(40, 10)
	if cells := d.Frame().Cells(); cells.Width() != 40 || cells.Height() != 10 {
		t.Errorf("after Resize(40, 10) the frame is %dx%d", cells.Width(), cells.Height())
	}
	if err := d.Err(); err != nil {
		t.Error(err)
	}
	if err := d.Close(); err != nil {
		t.Error(err)
	}
}

func recorder(keys *[]input.KeyEvent) drive.App {
	return func(*twi.Runtime) func() twi.Node {
		return func() twi.Node {
			return twi.Element(twi.OnKey(func(k input.KeyEvent) { *keys = append(*keys, k) }), twi.Text("keys"))
		}
	}
}

func TestPressKeys(t *testing.T) {
	cases := []struct {
		name string
		want input.KeyEvent
	}{
		{"a", input.KeyEvent{Rune: 'a'}},
		{"中", input.KeyEvent{Rune: '中'}},
		{"+", input.KeyEvent{Rune: '+'}},
		{"space", input.KeyEvent{Rune: ' '}},
		{"enter", input.KeyEvent{Key: input.KeyEnter}},
		{"Enter", input.KeyEvent{Key: input.KeyEnter}},
		{"tab", input.KeyEvent{Key: input.KeyTab}},
		{"backspace", input.KeyEvent{Key: input.KeyBackspace}},
		{"escape", input.KeyEvent{Key: input.KeyEscape}},
		{"up", input.KeyEvent{Key: input.KeyArrowUp}},
		{"down", input.KeyEvent{Key: input.KeyArrowDown}},
		{"left", input.KeyEvent{Key: input.KeyArrowLeft}},
		{"right", input.KeyEvent{Key: input.KeyArrowRight}},
		{"home", input.KeyEvent{Key: input.KeyHome}},
		{"end", input.KeyEvent{Key: input.KeyEnd}},
		{"pageup", input.KeyEvent{Key: input.KeyPageUp}},
		{"insert", input.KeyEvent{Key: input.KeyInsert}},
		{"f1", input.KeyEvent{Key: input.KeyF1}},
		{"f5", input.KeyEvent{Key: input.KeyF5}},
		{"f12", input.KeyEvent{Key: input.KeyF12}},
		{"alt+x", input.KeyEvent{Rune: 'x', Modifiers: input.ModAlt}},
		{"shift+tab", input.KeyEvent{Key: input.KeyTab, Modifiers: input.ModShift}},
		{"shift+f10", input.KeyEvent{Key: input.KeyF10, Modifiers: input.ModShift}},
		{"shift+enter", input.KeyEvent{Key: input.KeyEnter, Modifiers: input.ModShift}},
		{"ctrl+up", input.KeyEvent{Key: input.KeyArrowUp, Modifiers: input.ModCtrl}},
		{"shift+delete", input.KeyEvent{Key: input.KeyDelete, Modifiers: input.ModShift}},
		{"ctrl+space", input.KeyEvent{Rune: ' ', Modifiers: input.ModCtrl}},
		{"ctrl+k", input.KeyEvent{Rune: 'k', Modifiers: input.ModCtrl}},
		{"ctrl++", input.KeyEvent{Rune: '+', Modifiers: input.ModCtrl}},
		{"shift+a", input.KeyEvent{Rune: 'A', Modifiers: input.ModShift}},
		{"super+k", input.KeyEvent{Rune: 'k', Modifiers: input.ModMeta}},
		{"ctrl+alt+shift+f5", input.KeyEvent{Key: input.KeyF5, Modifiers: input.ModCtrl | input.ModAlt | input.ModShift}},
	}
	var keys []input.KeyEvent
	d := drive.New(recorder(&keys))
	for _, c := range cases {
		keys = nil
		d.Press(c.name)
		if !reflect.DeepEqual(keys, []input.KeyEvent{c.want}) {
			t.Errorf("Press(%q) reached the app as %#v, want %#v", c.name, keys, c.want)
		}
	}
	if err := d.Err(); err != nil {
		t.Error(err)
	}
	if err := d.Close(); err != nil {
		t.Error(err)
	}
}

func TestPressRejects(t *testing.T) {
	for _, name := range []string{"", "ctrl+", "ctrl+foo", "hyper+a", "\x01", "ab"} {
		var keys []input.KeyEvent
		d := drive.New(recorder(&keys))
		d.Press(name)
		d.Press("a")
		if d.Err() == nil || len(keys) != 0 {
			t.Errorf("Press(%q): err %v, keys %#v", name, d.Err(), keys)
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}
}

func TestTypeText(t *testing.T) {
	var keys []input.KeyEvent
	d := drive.New(recorder(&keys))
	d.Type("Hé 中")
	want := []input.KeyEvent{{Rune: 'H'}, {Rune: 'é'}, {Rune: ' '}, {Rune: '中'}}
	if !reflect.DeepEqual(keys, want) || d.Err() != nil {
		t.Errorf("Type reached the app as %#v, err %v", keys, d.Err())
	}
	keys = nil
	d.Type("a\x1b[A")
	if d.Err() == nil || len(keys) != 0 {
		t.Errorf("Type with an escape: err %v, keys %#v", d.Err(), keys)
	}
	if err := d.Close(); err != nil {
		t.Error(err)
	}
}

func TestCtrlCQuits(t *testing.T) {
	var keys []input.KeyEvent
	d := drive.New(recorder(&keys))
	d.Press("ctrl+c")
	if d.Err() != nil {
		t.Errorf("ctrl+c: %v", d.Err())
	}
	d.Press("a")
	if err := d.Err(); err == nil || !strings.Contains(err.Error(), "exited") || len(keys) != 0 {
		t.Errorf("press after the app quit: err %v, keys %#v", err, keys)
	}
	if err := d.Close(); err != nil {
		t.Error(err)
	}
}

func TestWideGlyph(t *testing.T) {
	d := drive.New(func(*twi.Runtime) func() twi.Node {
		return func() twi.Node { return twi.Text("中x") }
	}, drive.Size(6, 1))
	f := d.Frame()
	cells := f.Cells()
	if f.Text() != "中x\n" || cells.At(0, 0).Width != buffer.Wide || cells.At(1, 0).Width != buffer.Continuation || cells.At(2, 0).Grapheme != "x" {
		t.Errorf("text %q, cells %#v", f.Text(), cells.Row(0))
	}
	if err := d.Close(); err != nil {
		t.Error(err)
	}
}

func styled(*twi.Runtime) func() twi.Node { return hello.App }

func script(t *testing.T, name string, app drive.App, opts ...drive.Option) map[string]string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("testdata", name+".twd"))
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := drive.RunScript(bytes.NewReader(src), app, out, opts...); err != nil {
		t.Fatal(err)
	}
	return files(t, out)
}

func files(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no files in %s: %v", dir, err)
	}
	contents := map[string]string{}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		contents[e.Name()] = string(b)
	}
	return contents
}

func sheet(t *testing.T) drive.Option {
	t.Helper()
	s, err := hello.Styles()
	if err != nil {
		t.Fatal(err)
	}
	return drive.Styles(s)
}

func TestScript(t *testing.T) {
	got, want := script(t, "counter", counter.New), map[string]string{}
	for name, golden := range files(t, filepath.Join("testdata", "counter")) {
		want[strings.TrimSuffix(name, ".golden")] = golden
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("frames differ from testdata/counter:\n%q", got)
	}
}

func TestDeterminism(t *testing.T) {
	first, firstCounter := script(t, "hello", styled, sheet(t)), script(t, "counter", counter.New)
	for run := 2; run <= 20; run++ {
		if !reflect.DeepEqual(script(t, "hello", styled, sheet(t)), first) || !reflect.DeepEqual(script(t, "counter", counter.New), firstCounter) {
			t.Fatalf("run %d gave different frames", run)
		}
	}
	t.Logf("20 runs identical, last frame later.ansi sha256 %x", sha256.Sum256([]byte(first["later.ansi"])))
}

func TestNoTerminal(t *testing.T) {
	want := script(t, "hello", styled, sheet(t))
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(r.Close(), w.Close()); err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = stdout })
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "dumb")
	t.Setenv("COLORTERM", "")
	if got := script(t, "hello", styled, sheet(t)); !reflect.DeepEqual(got, want) {
		t.Errorf("frames changed with stdout closed and NO_COLOR set:\n%v", got)
	}
}

func TestPointerVerbs(t *testing.T) {
	s, err := hover.Styles()
	if err != nil {
		t.Fatal(err)
	}
	trace := &hover.Trace{}
	script := "size 30x8\nmove 2 1\ndown 2 1\nmove 9 1\nup 9 1\nclick 9 1\n"
	if err := drive.RunScript(strings.NewReader(script), trace.App, t.TempDir(), drive.Styles(s)); err != nil {
		t.Fatal(err)
	}
	want := "enter row, enter one, leave one, enter two, click two, click row"
	if got := strings.Join(trace.Events, ", "); got != want {
		t.Errorf("events %q, want %q", got, want)
	}
}

func TestScriptWidthsAndButtons(t *testing.T) {
	var seen []string
	app := func(rt *twi.Runtime) func() twi.Node {
		return func() twi.Node {
			return twi.Element(
				twi.OnPointerDown(func(e *twi.Event) { seen = append(seen, fmt.Sprintf("%d@%d,%d", e.Mouse.Button, e.Mouse.X, e.Mouse.Y)) }),
				twi.Text(fmt.Sprint(rt.Widths())),
			)
		}
	}
	out := t.TempDir()
	script := "widths flag=1 keycap=1\nsize 12x2\nclick right 3 0\ndown middle 1 0\nup middle 1 0\nclick left 2 0\nclick 4 0\nframe f\n"
	if err := drive.RunScript(strings.NewReader(script), app, out); err != nil {
		t.Fatal(err)
	}
	if got := files(t, out)["f.txt"]; got != "[1 0 0 0 1]\n\n" {
		t.Errorf("the app saw widths %q, want [1 0 0 0 1]", got)
	}
	want := []string{
		fmt.Sprintf("%d@3,0", input.MouseRight), fmt.Sprintf("%d@1,0", input.MouseMiddle),
		fmt.Sprintf("%d@2,0", input.MouseLeft), fmt.Sprintf("%d@4,0", input.MouseLeft),
	}
	if !slices.Equal(seen, want) {
		t.Errorf("pointer downs %v, want %v", seen, want)
	}
}

func TestScriptErrors(t *testing.T) {
	cases := []struct{ script, want string }{
		{"size 20x3\npress +\nfly away\n", "line 3: drive: unknown verb \"fly\""},
		{"size 20x3\n\nfly\n", "line 3:"},
		{"press +\nsize 20x3\n", "line 2: drive: size comes before"},
		{"size 20\n", "line 1: drive: size"},
		{"wait 5\n", "line 1: time: missing unit"},
		{"press hyper+a\n", "line 1: drive: unknown modifier"},
		{"frame ../x\n", "line 1: drive: frame name"},
		{"frame a/b\n", "line 1: drive: frame name"},
		{"resize 0x3\n", "line 1: drive: size"},
		{"wheel left 1 1\n", "line 1: drive: wheel \"left 1 1\" is not up|down X Y"},
		{"wheel down 1\n", "line 1: drive: wheel"},
		{"size 20x3\nwheel up 20 0\n", "line 2: drive: wheel at 20,0 is off the 20x3 screen"},
		{"move 1\n", "line 1: drive: move \"1\" is not X Y"},
		{"click 1 y\n", "line 1: drive: click \"1 y\" is not X Y"},
		{"size 20x3\nclick 0 3\n", "line 2: drive: down at 0,3 is off the 20x3 screen"},
		{"click up 1 1\n", "line 1: drive: click \"up 1 1\" is not X Y"},
		{"down right 1\n", "line 1: drive: down \"1\" is not X Y"},
		{"widths flag=0\n", "line 1: drive: widths \"flag=0\" is not CLASS=CELLS"},
		{"widths emoji=1\n", "line 1: drive: widths \"emoji=1\""},
		{"widths flag\n", "line 1: drive: widths \"flag\""},
		{"press +\nwidths flag=1\n", "line 2: drive: widths comes before"},
	}
	for _, c := range cases {
		err := drive.RunScript(strings.NewReader(c.script), counter.New, t.TempDir())
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("script %q: error %v, want %q", c.script, err, c.want)
		}
	}
}
