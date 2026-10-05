package runtime_test

import (
	"image"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/render/testdata/sheet"
	"github.com/pehcastro/twind/internal/runtime"
	"github.com/pehcastro/twind/internal/runtime/testdata/pill"
	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/style"
)

func TestGraphicsReachesRender(t *testing.T) {
	styles, err := sheet.Styles()
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range []terminal.Graphics{terminal.GraphicsSixel, terminal.GraphicsNone} {
		b := newBackend(20, 6)
		b.caps = terminal.Capabilities{Graphics: g, CellPixels: image.Pt(10, 20)}
		c := &clock{}
		rt := runtime.New(runtime.Config{Clock: c, Sheet: styles, Profile: color.TrueColor})
		tree := runtime.Tree{Root: render.Node{Children: []render.Node{{Classes: strings.Fields(sheet.Spinning)}}}}
		r := run{b: b, done: make(chan error, 1)}
		go func() { r.done <- rt.Run(b, func() runtime.Tree { return tree }) }()
		r.next(t)
		time.Sleep(50 * time.Millisecond)
		before := c.wakes.Load()
		time.Sleep(300 * time.Millisecond)
		wakes := c.wakes.Load() - before
		rt.Quit()
		if err := r.result(t); err != nil {
			t.Fatal(err)
		}
		t.Logf("graphics %v: %d wakes in 300 ms", g, wakes)
		if spinning := wakes > 3; spinning != (g != terminal.GraphicsNone) {
			t.Errorf("graphics %v: a spinner woke the loop %d times in 300 ms", g, wakes)
		}
	}
}

func TestFocusFromCodeHidesTheRing(t *testing.T) {
	styles, err := pill.Styles()
	if err != nil {
		t.Fatal(err)
	}
	lit := func(class string, state style.State) style.ComputedStyle {
		return styles.ComputeState(style.ComputedStyle{}, strings.Fields(class), style.NodeState{States: state})
	}
	ring := lit("focus-visible:ring-2 focus-visible:ring-sky-500", style.StateFocusVisible).Shadows[0].Color.RGBA
	within := lit("focus-within:bg-zinc-900", style.StateFocusWithin).Background.RGBA
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		pills := pill.App(rt)
		return func() twi.Node {
			return twi.Element(twi.OnKeyDown(func(e *twi.Event) { rt.Focus(map[rune]string{'o': "one", 't': "two"}[e.Key.Rune]) }), pills())
		}
	}, drive.Size(30, 9), drive.Styles(styles))
	defer func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}()
	painted := func(rgba color.RGBA) bool {
		cells := d.Frame().Cells()
		for y := range cells.Height() {
			for x := range cells.Width() {
				if c := cells.At(x, y); c.Fg.RGBA == rgba || c.Bg.RGBA == rgba {
					return true
				}
			}
		}
		return false
	}
	for _, step := range []struct {
		key             string
		focused, ringed bool
	}{{"t", true, false}, {"o", true, false}, {"tab", true, true}} {
		d.Press(step.key)
		if got := [2]bool{painted(within), painted(ring)}; got != [2]bool{step.focused, step.ringed} {
			t.Errorf("%q: focus-within row, focus-visible ring = %v, want %v:\n%s", step.key, got, [2]bool{step.focused, step.ringed}, d.Frame().Text())
		}
	}
}

func TestFocusSendsBlurThenFocus(t *testing.T) {
	a := &focusApp{}
	var took []bool
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		a.focused = twi.NewSignal(rt, "")
		return func() twi.Node {
			return twi.Element(
				twi.OnKeyDown(func(e *twi.Event) {
					switch e.Key.Rune {
					case 'b':
						took = append(took, rt.Focus("b"))
					case 'm':
						took = append(took, rt.Focus("missing"))
					case 'd':
						took = append(took, rt.Focus("d"))
					}
				}),
				twi.Text("focused "+a.focused.Get()),
				a.button("a"), a.button("b"), a.button("d", twi.Disabled()),
			)
		}
	}, drive.Size(30, 6))
	defer func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}()
	d.Press("tab")
	expect(t, a, "tab", "focus a")
	d.Press("b")
	expect(t, a, "focus b from code", "a b, blur a, focus b")
	if text := d.Frame().Text(); !strings.HasPrefix(text, "focused b") {
		t.Errorf("the frame after Focus does not show it:\n%s", text)
	}
	d.Press("b")
	expect(t, a, "focus b again", "b b")
	d.Press("m")
	d.Press("d")
	expect(t, a, "a missing key and a disabled one", "b m, b d")
	if want := []bool{true, true, false, false}; !slices.Equal(took, want) {
		t.Errorf("Focus returned %v, want %v", took, want)
	}
}

func nested(before int) runtime.Tree {
	rows := func(prefix string, n int) []render.Node {
		out := make([]render.Node, n)
		for i := range out {
			out[i] = render.Node{Text: prefix + " " + strconv.Itoa(i)}
		}
		return out
	}
	inner := render.Node{Classes: []string{"col", "h-3", "shrink-0", "overflow-y-auto"}, Children: rows("row", 100)}
	outer := render.Node{Classes: []string{"col", "h-5", "w-12", "shrink-0", "overflow-y-auto"}, Children: append(append(rows("before", before), inner), rows("after", 10)...)}
	return runtime.Tree{
		Root:   render.Node{Classes: []string{"col"}, Children: []render.Node{outer}},
		Events: runtime.Node{Children: []runtime.Node{{Key: "outer", At: []int{0}, Children: []runtime.Node{{Key: "row 50", At: []int{before, 50}}, {Key: "row 80", At: []int{before, 80}}, {Key: "row 99", At: []int{before, 99}}, {Key: "after 9", At: []int{before + 10}}}}}},
	}
}

type intoViewStep struct {
	key  string
	rows []string
}

func TestScrollIntoView(t *testing.T) {
	scrollIntoView(t, 10, []intoViewStep{
		{"row 80", []string{"before 6", "before 7", "before 8", "before 9", "row 80"}},
		{"row 99", []string{"before 8", "before 9", "row 97", "row 98", "row 99"}},
		{"after 9", []string{"after 5", "after 6", "after 7", "after 8", "after 9"}},
	})
}

func TestScrollIntoViewKeepsAShownScroller(t *testing.T) {
	scrollIntoView(t, 1, []intoViewStep{{"row 50", []string{"before 0", "row 50", "row 51", "row 52", "after 0"}}})
}

func scrollIntoView(t *testing.T, before int, steps []intoViewStep) {
	t.Helper()
	b := newBackend(20, 6)
	tree := nested(before)
	rt := runtime.New(runtime.Config{Clock: &clock{}, Sheet: scrollSheet(t), Profile: color.None})
	r := run{b: b, done: make(chan error, 1)}
	go func() { r.done <- rt.Run(b, func() runtime.Tree { return tree }) }()
	g := blank(b)
	g.apply(t, r.next(t))
	t.Logf("before:\n%s", g.text())
	quiet := func(what string) {
		select {
		case f := <-b.frames:
			g.apply(t, f)
			t.Errorf("%s drew a frame:\n%s", what, g.text())
		case <-time.After(100 * time.Millisecond):
		}
	}
	for _, step := range steps {
		rt.Dispatch(func() { rt.ScrollIntoView(step.key) })
		g.apply(t, r.next(t))
		t.Logf("%s:\n%s", step.key, g.text())
		for i, want := range step.rows {
			if got := strings.TrimSpace(string(g[i][:listWidth-1])); got != want {
				t.Errorf("%s: row %d is %q, want %q:\n%s", step.key, i, got, want, g.text())
				break
			}
		}
	}
	rt.Dispatch(func() { rt.ScrollIntoView("outer") })
	quiet("the scroll area itself")
	rt.Dispatch(func() { rt.ScrollIntoView("missing") })
	quiet("a missing key")
	rt.Quit()
	if err := r.result(t); err != nil {
		t.Fatal(err)
	}
}
