package runtime_test

import (
	"image"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	konst "github.com/pehcastro/twind/internal/konst/style"
	termkonst "github.com/pehcastro/twind/internal/konst/terminal"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/events"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/runtime"
	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/terminal"
)

const (
	listRows  = 20
	listView  = 5
	listWidth = 12
	consumer  = 2
)

type grid [][]rune

func blank(b *backend) grid {
	g := make(grid, b.height)
	for y := range g {
		g[y] = []rune(strings.Repeat(" ", b.width))
	}
	return g
}

func (g grid) apply(t *testing.T, frame string) {
	t.Helper()
	x, y := 0, 0
	for rest := frame; rest != ""; {
		if !strings.HasPrefix(rest, termkonst.CSI) {
			r, n := utf8.DecodeRuneInString(rest)
			g[y][x], x, rest = r, x+1, rest[n:]
			continue
		}
		end := strings.IndexFunc(rest[len(termkonst.CSI):], func(r rune) bool { return r >= '@' && r <= '~' }) + len(termkonst.CSI)
		params := strings.Split(rest[len(termkonst.CSI):end], ";")
		n := func(i int) int { v, _ := strconv.Atoi(params[i]); return v }
		switch rest[end] {
		case 'H':
			y, x = n(0)-1, n(1)-1
		case 'C':
			x += n(0)
		case 'm':
		default:
			t.Fatalf("frame has %q", rest[:end+1])
		}
		rest = rest[end+1:]
	}
}

func (g grid) text() string {
	var b strings.Builder
	for _, row := range g {
		b.WriteString(strings.TrimRight(string(row), " ") + "\n")
	}
	return b.String()
}

func (g grid) rows() []string {
	var rows []string
	for _, row := range g[:listView] {
		rows = append(rows, strings.TrimSpace(string(row[:listWidth-1])))
	}
	return rows
}

func (g grid) first() int {
	n, _ := strconv.Atoi(strings.Fields(g.rows()[0])[1])
	return n
}

func (g grid) side() string {
	var rows []string
	for _, row := range g {
		rows = append(rows, strings.TrimSpace(string(row[listWidth:])))
	}
	return strings.Join(rows, "|")
}

type scrollApp struct {
	rt      *runtime.Runtime
	b       *backend
	grid    grid
	focused int
}

func scrollSheet(t testing.TB) style.Sheet {
	t.Helper()
	cells := func(p style.Property, n float64) style.Declaration {
		return style.Declaration{Property: p, Length: style.Length{Unit: style.Cells, Value: n}}
	}
	flex := func(d style.Direction) []style.Declaration {
		return []style.Declaration{{Property: style.PropDisplay, Display: style.DisplayFlex}, {Property: style.PropDirection, Direction: d}}
	}
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "row", Decls: flex(style.Row)},
		{Class: "col", Decls: flex(style.Column)},
		{Class: "h-3", Decls: []style.Declaration{cells(style.PropHeight, 3)}},
		{Class: "h-5", Decls: []style.Declaration{cells(style.PropHeight, listView)}},
		{Class: "h-20", Decls: []style.Declaration{cells(style.PropHeight, 20)}},
		{Class: "w-12", Decls: []style.Declaration{cells(style.PropWidth, listWidth)}},
		{Class: "shrink-0", Decls: []style.Declaration{{Property: style.PropShrink}}},
		{Class: "pt-h", Decls: []style.Declaration{cells(style.PropPaddingTop, 0.5)}},
		{Class: "overflow-y-auto", Decls: []style.Declaration{{Property: style.PropOverflowY, Overflow: style.OverflowAuto}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}

func (a *scrollApp) tree() runtime.Tree {
	rows := make([]render.Node, listRows)
	elems := make([]runtime.Node, listRows)
	for i := range rows {
		rows[i] = render.Node{Text: "row " + strconv.Itoa(i)}
		if i == a.focused {
			rows[i].Text += " *"
		}
		elems[i] = runtime.Node{At: []int{i}, Focusable: true, Listeners: []events.Listener[*runtime.Elem]{{Type: events.Focus, Handle: func(*events.Event[*runtime.Elem]) {
			a.focused = i
			a.rt.Invalidate()
		}}}}
	}
	elems[consumer].Listeners = append(elems[consumer].Listeners, events.Listener[*runtime.Elem]{Type: events.KeyDown, Handle: func(e *events.Event[*runtime.Elem]) {
		if e.Key.Key == input.KeyPageDown {
			e.PreventDefault()
		}
	}})
	return runtime.Tree{
		Root: render.Node{Classes: []string{"row"}, Children: []render.Node{
			{Classes: []string{"col", "h-5", "w-12", "shrink-0", "overflow-y-auto"}, Children: rows},
			{Classes: []string{"col"}, Children: []render.Node{{Text: "fixed one"}, {Text: "fixed two"}, {Text: "fixed three"}}},
		}},
		Events: runtime.Node{Children: []runtime.Node{{At: []int{0}, Focusable: true, Children: elems}}},
	}
}

func startScroll(t *testing.T, b *backend, profile color.Profile) *scrollApp {
	t.Helper()
	a := &scrollApp{b: b, focused: -1, grid: blank(b)}
	a.rt = runtime.New(runtime.Config{Clock: &clock{}, Sheet: scrollSheet(t), Profile: profile})
	done := make(chan error, 1)
	go func() { done <- a.rt.Run(b, a.tree) }()
	t.Cleanup(func() {
		a.rt.Quit()
		if err := (run{done: done}).result(t); err != nil {
			t.Error(err)
		}
	})
	return a
}

func (a *scrollApp) frame(t *testing.T) string {
	t.Helper()
	f := run{b: a.b}.next(t)
	if a.b.caps.Graphics == terminal.GraphicsNone {
		a.grid.apply(t, f)
	}
	return f
}

func (a *scrollApp) send(t *testing.T, ev input.Event) {
	t.Helper()
	a.b.events <- ev
	a.frame(t)
}

func (a *scrollApp) nothing(t *testing.T, what string, ev input.Event) {
	t.Helper()
	a.b.events <- ev
	select {
	case f := <-a.b.frames:
		a.grid.apply(t, f)
		t.Fatalf("%s drew a frame:\n%s", what, a.grid.text())
	case <-time.After(100 * time.Millisecond):
	}
}

func wheel(x, y int, b input.MouseButton) input.MouseEvent {
	return input.MouseEvent{X: x, Y: y, Button: b, Action: input.MouseScroll}
}

func press(k input.Key) input.KeyEvent { return input.KeyEvent{Key: k} }

func TestWheelAndPageDownMoveOnlyTheList(t *testing.T) {
	a := startScroll(t, newBackend(30, 7), color.None)
	a.frame(t)
	side := a.grid.side()
	a.nothing(t, "a wheel under the list, over rows it clips", wheel(2, listView+1, input.MouseWheelDown))
	a.nothing(t, "tab onto the list itself", press(input.KeyTab))
	a.send(t, press(input.KeyTab))
	for notch := 1; notch <= 3; notch++ {
		a.send(t, wheel(2, 1, input.MouseWheelDown))
		if got, want := a.grid.first(), notch*3; got != want {
			t.Fatalf("after notch %d the list starts at row %d, want %d:\n%s", notch, got, want, a.grid.text())
		}
	}
	a.nothing(t, "a wheel over the fixed text", wheel(listWidth+2, 1, input.MouseWheelDown))
	a.nothing(t, "a mouse move over the list", input.MouseEvent{X: 2, Y: 1, Action: input.MouseMove})
	a.send(t, press(input.KeyPageDown))
	if got, want := a.grid.first(), 9+listView; got != want {
		t.Fatalf("after PageDown the list starts at row %d, want %d:\n%s", got, want, a.grid.text())
	}
	if got := a.grid.side(); got != side {
		t.Errorf("the text beside the list moved: %q, was %q", got, side)
	}
	t.Logf("three notches and a PageDown:\n%s", a.grid.text())
}

func TestScrollKeys(t *testing.T) {
	a := startScroll(t, newBackend(30, 7), color.None)
	a.frame(t)
	a.nothing(t, "tab onto the list itself", press(input.KeyTab))
	for _, step := range []struct {
		key   input.Key
		first int
	}{
		{input.KeyArrowDown, 1}, {input.KeyEnd, listRows - listView}, {input.KeyArrowUp, listRows - listView - 1},
		{input.KeyHome, 0}, {input.KeyPageDown, listView}, {input.KeyPageUp, 0},
	} {
		a.send(t, press(step.key))
		if got := a.grid.first(); got != step.first {
			t.Fatalf("key %d on the focused list: starts at row %d, want %d:\n%s", step.key, got, step.first, a.grid.text())
		}
	}
	a.send(t, press(input.KeyTab))
	a.nothing(t, "an arrow on a row inside the list", press(input.KeyArrowDown))
	a.nothing(t, "ctrl+PageDown", input.KeyEvent{Key: input.KeyPageDown, Modifiers: input.ModCtrl})
	a.nothing(t, "a PageDown release", input.KeyEvent{Key: input.KeyPageDown, Release: true})
	a.send(t, press(input.KeyTab))
	a.send(t, press(input.KeyTab))
	a.nothing(t, "a PageDown the focused row prevented", press(input.KeyPageDown))
}

func TestTabRevealsTheFocusedRow(t *testing.T) {
	a := startScroll(t, newBackend(30, 7), color.None)
	a.frame(t)
	a.nothing(t, "tab onto the list itself", press(input.KeyTab))
	inView := func(row int) bool {
		for _, r := range a.grid.rows() {
			if r == "row "+strconv.Itoa(row)+" *" {
				return true
			}
		}
		return false
	}
	for row := range listRows {
		a.send(t, press(input.KeyTab))
		if !inView(row) {
			t.Fatalf("tab to row %d: not in view:\n%s", row, a.grid.text())
		}
	}
	t.Logf("tabbed to the last row:\n%s", a.grid.text())
	a.send(t, wheel(2, 1, input.MouseWheelUp))
	a.send(t, wheel(2, 1, input.MouseWheelUp))
	if a.grid.first() != listRows-listView-6 {
		t.Fatalf("the wheel away from the focused row was undone:\n%s", a.grid.text())
	}
	a.send(t, input.KeyEvent{Key: input.KeyTab, Modifiers: input.ModShift})
	if !inView(listRows - 2) {
		t.Fatalf("shift+tab to row %d: not in view:\n%s", listRows-2, a.grid.text())
	}
}

func TestWheelChainsToTheOuterScroller(t *testing.T) {
	b := newBackend(20, 6)
	inner := make([]render.Node, 6)
	for i := range inner {
		inner[i] = render.Node{Text: "inner " + strconv.Itoa(i)}
	}
	outer := []render.Node{{Classes: []string{"col", "h-3", "shrink-0", "overflow-y-auto"}, Children: inner}}
	for i := range 10 {
		outer = append(outer, render.Node{Text: "outer " + strconv.Itoa(i)})
	}
	g := blank(b)
	rt := runtime.New(runtime.Config{Clock: &clock{}, Sheet: scrollSheet(t), Profile: color.None})
	done := make(chan error, 1)
	go func() {
		done <- rt.Run(b, func() runtime.Tree {
			return runtime.Tree{Root: render.Node{Classes: []string{"col", "h-5", "w-12", "overflow-y-auto"}, Children: outer}}
		})
	}()
	r := run{b: b, done: done}
	g.apply(t, r.next(t))
	top := func() string { return strings.TrimSpace(string(g[0][:listWidth-1])) }
	for _, want := range []string{"inner 3", "outer 0", "outer 3", "outer 5"} {
		b.events <- wheel(1, 0, input.MouseWheelDown)
		g.apply(t, r.next(t))
		if got := top(); got != want {
			t.Fatalf("wheel down: the top row is %q, want %q:\n%s", got, want, g.text())
		}
	}
	b.events <- wheel(1, 0, input.MouseWheelUp)
	g.apply(t, r.next(t))
	if got := top(); got != "outer 2" {
		t.Fatalf("wheel up over a row of the outer list: the top row is %q, want outer 2:\n%s", got, g.text())
	}
	rt.Quit()
	if err := r.result(t); err != nil {
		t.Fatal(err)
	}
}

func TestRegionScrollAsksForMarginsOnlyWhenTheTerminalHasThem(t *testing.T) {
	for _, margins := range []bool{false, true} {
		b := newBackend(30, 7)
		b.caps = terminal.Capabilities{Graphics: terminal.GraphicsSixel, CellPixels: image.Pt(10, 20), Margins: margins}
		a := startScroll(t, b, color.TrueColor)
		a.frame(t)
		a.b.events <- wheel(2, 1, input.MouseWheelDown)
		f := a.frame(t)
		if !strings.Contains(f, termkonst.CSI+"1;5"+termkonst.RegionRows) {
			t.Fatalf("margins %v: a one-notch wheel wrote no region scroll: %q", margins, f)
		}
		if got := strings.Contains(f, termkonst.MarginsOn); got != margins {
			t.Errorf("margins %v: wrote %q, left and right margins asked %v", margins, termkonst.MarginsOn, got)
		}
	}
}
