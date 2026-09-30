package runtime_test

import (
	"strings"
	"testing"
	"time"

	rkonst "github.com/twind-dev/twind/internal/konst/runtime"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/runtime"
	"github.com/twind-dev/twind/twi/runtime/testdata/pill"
	"github.com/twind-dev/twind/twi/style"
)

type focusApp struct {
	log                           []string
	before, open, gone, autofocus bool
	focused                       *twi.Signal[string]
}

func (a *focusApp) record(entry string) { a.log = append(a.log, entry) }

func (a *focusApp) heard(who string, k input.KeyEvent) {
	switch k.Key {
	case input.KeyRune:
		a.record(who + " " + string(k.Rune))
	case input.KeyEnter:
		a.record(who + " enter")
	case input.KeyEscape:
		a.record(who + " escape")
	}
}

func (a *focusApp) take() string {
	out := strings.Join(a.log, ", ")
	a.log = nil
	return out
}

func (a *focusApp) button(name string, opts ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{
		twi.Key(name),
		twi.Focusable(),
		twi.OnFocus(func() {
			a.record("focus " + name)
			a.focused.Set(name)
		}),
		twi.OnBlur(func() { a.record("blur " + name) }),
		twi.OnKeyDown(func(e *twi.Event) { a.heard(name, e.Key) }),
		twi.Text(name),
	}, opts...)...)
}

func (a *focusApp) start(t *testing.T) *drive.Driver {
	t.Helper()
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		in := twi.NewInput(rt)
		redraw := twi.NewSignal(rt, 0)
		a.focused = twi.NewSignal(rt, "")
		return func() twi.Node {
			root := []twi.NodeOption{
				twi.Text("focused " + a.focused.Get()),
				twi.OnKeyDown(func(e *twi.Event) {
					a.heard("root", e.Key)
					switch {
					case e.Key.Rune == 'i':
						a.before = true
					case e.Key.Rune == 'r':
						a.gone = true
					case e.Key.Rune == 'f':
						a.autofocus = true
					case e.Key.Key == input.KeyEnter:
						a.open = true
					case e.Key.Key == input.KeyEscape:
						a.open = false
					}
					redraw.Set(redraw.Get() + 1)
				}),
				twi.OnKey(func(k input.KeyEvent) { a.heard("document", k) }),
				in.Node(twi.AutoFocus(), twi.Key("input")),
			}
			if a.before {
				root = append(root, a.button("c"))
			}
			root = append(root, a.button("a"), a.button("d", twi.Disabled()))
			if !a.gone {
				root = append(root, a.button("b"))
			}
			if a.autofocus {
				root = append(root, a.button("late", twi.AutoFocus()))
			}
			if a.open {
				root = append(root, twi.Element(twi.Key("dialog"), twi.FocusScope(), a.button("one"), a.button("two")))
			}
			return twi.Element(root...)
		}
	}, drive.Size(40, 12))
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

func expect(t *testing.T, a *focusApp, step, want string) {
	t.Helper()
	if got := a.take(); got != want {
		t.Errorf("%s: got %q, want %q", step, got, want)
	}
}

func TestFocusInputKeepsItsKeys(t *testing.T) {
	a := &focusApp{}
	d := a.start(t)
	d.Type("q")
	expect(t, a, "q in the autofocused input", "")
	if text := d.Frame().Text(); !strings.Contains(text, "\nq\n") {
		t.Errorf("the input does not show q:\n%s", text)
	}
	d.Press("tab")
	d.Press("x")
	expect(t, a, "tab then x", "focus a, a x, root x, document x")
	d.Press("tab")
	d.Press("x")
	expect(t, a, "tab skips the disabled d", "blur a, focus b, b x, root x, document x")
	d.Press("shift+tab")
	d.Press("shift+tab")
	d.Type("z")
	expect(t, a, "back in the input", "blur b, focus a, blur a")
	if text := d.Frame().Text(); !strings.Contains(text, "\nqz\n") {
		t.Errorf("the input lost its value or its focus:\n%s", text)
	}
}

func TestTabShowsFocusVisibleRing(t *testing.T) {
	sheet, err := pill.Styles()
	if err != nil {
		t.Fatal(err)
	}
	lit := func(classes string, states style.State) style.ComputedStyle {
		return sheet.ComputeState(style.ComputedStyle{}, strings.Fields(classes), style.NodeState{States: states})
	}
	ring := lit("focus-visible:ring-2 focus-visible:ring-sky-500", style.StateFocusVisible).Shadows[0].Color.RGBA
	within := lit("focus-within:bg-zinc-900", style.StateFocusWithin).Background.RGBA
	pillBg := lit("bg-zinc-700", 0).Background.RGBA
	d := drive.New(pill.App, drive.Size(30, 9), drive.Styles(sheet))
	defer func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}()
	locate := func(word string) (int, int) {
		cells := d.Frame().Cells()
		for y := range cells.Height() {
			for x := range cells.Width() - len(word) {
				found := true
				for i, r := range word {
					found = found && cells.At(x+i, y).Grapheme == string(r)
				}
				if found {
					return x, y
				}
			}
		}
		t.Fatalf("no %q in the frame:\n%s", word, d.Frame().Text())
		return 0, 0
	}
	ringed := func(word string) bool {
		cells := d.Frame().Cells()
		x0, y0 := locate(word)
		for y := max(y0-1, 0); y <= min(y0+1, cells.Height()-1); y++ {
			for x := max(x0-3, 0); x < min(x0+len(word)+3, cells.Width()); x++ {
				if c := cells.At(x, y); c.Fg.RGBA == ring || c.Bg.RGBA == ring {
					return true
				}
			}
		}
		return false
	}
	for _, step := range []struct {
		key           string
		one, two, row bool
	}{
		{"", false, false, false},
		{"tab", true, false, true},
		{"tab", false, true, true},
		{"tab", true, false, true},
	} {
		if step.key != "" {
			d.Press(step.key)
		}
		_, y := locate("one")
		row := d.Frame().Cells().At(1, y).Bg.RGBA == within
		if got := [3]bool{ringed("one"), ringed("two"), row}; got != [3]bool{step.one, step.two, step.row} {
			t.Errorf("after %q: ring on one, ring on two, focus-within row = %v, want %v:\n%s", step.key, got, [3]bool{step.one, step.two, step.row}, d.Frame().ANSI())
		}
		underlined := func(word string) bool {
			x, y := locate(word)
			return d.Frame().Cells().At(x, y).Attr&buffer.Underline != 0
		}
		if underlined("one") || !underlined("two") {
			t.Errorf("after %q: data-[state=on]:underline must mark two only, focused or not:\n%s", step.key, d.Frame().ANSI())
		}
	}
	x, y := locate("off")
	if bg := d.Frame().Cells().At(x, y).Bg.RGBA; bg == pillBg {
		t.Errorf("the disabled pill is not faded by disabled:opacity-50: background %v", bg)
	}
}

func TestFocusLeavesTheAppTreeUntouched(t *testing.T) {
	b := newBackend(20, 3)
	tree := runtime.Tree{
		Root:   render.Node{Children: []render.Node{{Text: "a"}, {Text: "b"}}},
		Events: runtime.Node{Children: []runtime.Node{{At: []int{0}, Focusable: true}, {At: []int{1}, Focusable: true}}},
	}
	rt := runtime.New(runtime.Config{Clock: &clock{}, Sheet: scrollSheet(t), Profile: color.None})
	r := run{b: b, done: make(chan error, 1)}
	go func() { r.done <- rt.Run(b, func() runtime.Tree { return tree }) }()
	r.next(t)
	for range 3 {
		time.Sleep(3 * rkonst.FrameInterval)
		b.events <- input.KeyEvent{Key: input.KeyTab}
		settled := make(chan struct{})
		rt.Dispatch(func() { rt.Dispatch(func() { close(settled) }) })
		<-settled
		if tree.Root.State != nil || tree.Root.Children[0].State != nil || tree.Root.Children[1].State != nil {
			t.Fatal("the runtime wrote focus states into the tree the app returned")
		}
	}
	rt.Quit()
	if err := r.result(t); err != nil {
		t.Fatal(err)
	}
}

func TestFocusFollowsIdentity(t *testing.T) {
	a := &focusApp{}
	d := a.start(t)
	d.Press("tab")
	d.Press("tab")
	d.Press("i")
	expect(t, a, "focus b, insert c before it", "focus a, blur a, focus b, b i, root i, document i")
	d.Press("f")
	d.Press("x")
	expect(t, a, "a later autofocus node", "b f, root f, document f, b x, root x, document x")
	d.Press("r")
	d.Press("x")
	expect(t, a, "b removed while focused", "b r, root r, document r, root x, document x")
}

func TestFocusScopeTrapsAndRestores(t *testing.T) {
	a := &focusApp{}
	d := a.start(t)
	d.Press("tab")
	d.Press("enter")
	expect(t, a, "enter opens the dialog", "focus a, a enter, root enter, document enter, blur a, focus one")
	if text := d.Frame().Text(); !strings.HasPrefix(text, "focused one") {
		t.Errorf("the frame that opens the dialog does not show the focus its opening set:\n%s", text)
	}
	d.Press("tab")
	d.Press("x")
	d.Press("tab")
	d.Press("tab")
	expect(t, a, "tab stays inside across redraws", "blur one, focus two, two x, root x, document x, blur two, focus one, blur one, focus two")
	d.Press("escape")
	d.Press("x")
	expect(t, a, "escape closes and restores a", "two escape, root escape, document escape, blur two, focus a, a x, root x, document x")
}
