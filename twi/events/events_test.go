package events

import (
	"image"
	"slices"
	"testing"

	"github.com/twind-dev/twind/twi/input"
)

type testNode struct {
	parent    string
	children  []string
	tab       int
	focusable bool
	disabled  bool
	listeners []Listener[string]
}

type testTree map[string]testNode

func (t testTree) Root() string                          { return "root" }
func (t testTree) Parent(n string) (string, bool)        { return t[n].parent, t[n].parent != "" }
func (t testTree) Children(n string) []string            { return t[n].children }
func (t testTree) Listeners(n string) []Listener[string] { return t[n].listeners }
func (t testTree) Focusable(n string) bool               { return t[n].focusable }
func (t testTree) TabIndex(n string) int                 { return t[n].tab }
func (t testTree) Disabled(n string) bool                { return t[n].disabled }
func (t testTree) Origin(string) image.Point             { return image.Point{} }

func (t testTree) add(parent, id string, n testNode) {
	n.parent = parent
	t[id] = n
	p := t[parent]
	p.children = append(p.children, id)
	t[parent] = p
}

func (t testTree) detach(id string) {
	n := t[id]
	p := t[n.parent]
	p.children = slices.DeleteFunc(p.children, func(c string) bool { return c == id })
	t[n.parent] = p
	n.parent = ""
	t[id] = n
}

func (t testTree) listen(id string, k Type, capture bool, handle func(*Event[string])) {
	n := t[id]
	n.listeners = append(n.listeners, Listener[string]{Type: k, Capture: capture, Handle: handle})
	t[id] = n
}

func traced(k Type, stopAt int) (testTree, *[]string) {
	tr := testTree{"root": {}}
	tr.add("root", "a", testNode{})
	tr.add("a", "b", testNode{})
	phases := []string{"", "capture", "target", "bubble"}
	var log []string
	for _, id := range []string{"root", "a", "b"} {
		for _, capture := range []bool{false, true} {
			tr.listen(id, k, capture, func(e *Event[string]) {
				log = append(log, phases[e.Phase()]+" "+e.Current()+" "+e.Target())
				if len(log) == stopAt {
					e.StopPropagation()
				}
			})
		}
	}
	return tr, &log
}

func fullPath() []string {
	return []string{"capture root b", "capture a b", "target b b", "target b b", "bubble a b", "bubble root b"}
}

func TestDispatchOrder(t *testing.T) {
	tr, log := traced(KeyDown, 0)
	Dispatch(tr, "b", &Event[string]{Type: KeyDown})
	if !slices.Equal(*log, fullPath()) {
		t.Fatalf("got %q, want %q", *log, fullPath())
	}
}

func TestDispatchNonBubbling(t *testing.T) {
	for _, k := range []Type{Focus, Blur} {
		tr, log := traced(k, 0)
		Dispatch(tr, "b", &Event[string]{Type: k})
		if !slices.Equal(*log, fullPath()[:4]) {
			t.Fatalf("type %d: got %q, want %q", k, *log, fullPath()[:4])
		}
	}
}

func TestDispatchStop(t *testing.T) {
	for stopAt := 1; stopAt < len(fullPath()); stopAt++ {
		tr, log := traced(KeyDown, stopAt)
		e := &Event[string]{Type: KeyDown}
		Dispatch(tr, "b", e)
		if !slices.Equal(*log, fullPath()[:stopAt]) {
			t.Errorf("stop at %d: got %q, want %q", stopAt, *log, fullPath()[:stopAt])
		}
	}

	tr := testTree{"root": {}}
	ran := false
	tr.listen("root", KeyDown, false, func(e *Event[string]) { e.StopPropagation() })
	tr.listen("root", KeyDown, false, func(*Event[string]) { ran = true })
	Dispatch(tr, "root", &Event[string]{Type: KeyDown})
	if ran {
		t.Error("a later listener on the stopping node ran")
	}
}

func TestDispatchPreventDefault(t *testing.T) {
	tr, _ := traced(KeyDown, 0)
	e := &Event[string]{Type: KeyDown}
	Dispatch(tr, "b", e)
	if e.DefaultPrevented() {
		t.Fatal("prevented with no listener asking")
	}
	tr.listen("root", KeyDown, false, func(e *Event[string]) { e.PreventDefault() })
	e = &Event[string]{Type: KeyDown}
	Dispatch(tr, "b", e)
	if !e.DefaultPrevented() {
		t.Fatal("PreventDefault in the bubble phase not visible to the caller")
	}
}

func tabs(tr testTree, f *FocusManager[string], k input.KeyEvent, n int) []string {
	var got []string
	for range n {
		f.Key(tr, k)
		c, _ := f.Current()
		got = append(got, c)
	}
	return got
}

func TestFocusTabOrder(t *testing.T) {
	tr := testTree{"root": {}}
	tr.add("root", "a", testNode{focusable: true})
	tr.add("root", "b", testNode{focusable: true, tab: 2})
	tr.add("root", "c", testNode{focusable: true, disabled: true})
	tr.add("root", "d", testNode{focusable: true, tab: 1})
	tr.add("root", "group", testNode{})
	tr.add("group", "e", testNode{focusable: true})
	tr.add("group", "f", testNode{focusable: true, tab: -1})
	tr.add("group", "g", testNode{focusable: true, tab: 2})
	tr.add("root", "h", testNode{})

	var forward, backward FocusManager[string]
	if got, want := tabs(tr, &forward, input.KeyEvent{Key: input.KeyTab}, 6), []string{"d", "b", "g", "a", "e", "d"}; !slices.Equal(got, want) {
		t.Errorf("Tab: got %q, want %q", got, want)
	}
	shiftTab := input.KeyEvent{Key: input.KeyTab, Modifiers: input.ModShift}
	if got, want := tabs(tr, &backward, shiftTab, 6), []string{"e", "a", "g", "b", "d", "e"}; !slices.Equal(got, want) {
		t.Errorf("Shift+Tab: got %q, want %q", got, want)
	}

	if backward.Set(tr, "c") || backward.Set(tr, "h") {
		t.Error("focused a disabled or unfocusable node")
	}
	if !backward.Set(tr, "f") {
		t.Error("refused a tabindex -1 node set by the program")
	}
}

func TestFocusPreventTab(t *testing.T) {
	tr := testTree{"root": {}}
	tr.add("root", "a", testNode{focusable: true})
	tr.add("root", "b", testNode{focusable: true})
	tr.listen("a", KeyDown, false, func(e *Event[string]) { e.PreventDefault() })
	var f FocusManager[string]
	f.Set(tr, "a")
	e := f.Key(tr, input.KeyEvent{Key: input.KeyTab})
	if c, _ := f.Current(); !e.DefaultPrevented() || e.Target() != "a" || c != "a" {
		t.Fatalf("prevented %v, target %q, focus %q: want true, a, a", e.DefaultPrevented(), e.Target(), c)
	}
}

func dialogTree() (testTree, *[]string) {
	tr := testTree{"root": {}}
	tr.add("root", "before", testNode{focusable: true})
	tr.add("root", "dialog", testNode{})
	tr.add("dialog", "x", testNode{focusable: true})
	tr.add("dialog", "y", testNode{focusable: true})
	tr.add("root", "after", testNode{focusable: true})
	var log []string
	for _, id := range []string{"before", "x", "y", "after"} {
		tr.listen(id, Focus, false, func(e *Event[string]) { log = append(log, "focus "+e.Target()) })
		tr.listen(id, Blur, false, func(e *Event[string]) { log = append(log, "blur "+e.Target()) })
	}
	return tr, &log
}

func TestFocusScope(t *testing.T) {
	tr, log := dialogTree()
	var f FocusManager[string]
	f.Set(tr, "after")
	f.Set(tr, "after")
	f.Open(tr, "dialog")
	if got, want := tabs(tr, &f, input.KeyEvent{Key: input.KeyTab}, 3), []string{"y", "x", "y"}; !slices.Equal(got, want) {
		t.Errorf("Tab in scope: got %q, want %q", got, want)
	}
	if f.Set(tr, "before") {
		t.Error("focused a node outside the open scope")
	}

	tr.detach("after")
	tr.add("root", "panel", testNode{})
	root := tr["root"]
	root.children = []string{"panel", "before", "dialog"}
	tr["root"] = root
	tr.add("panel", "after", tr["after"])

	f.Close(tr)
	if c, ok := f.Current(); !ok || c != "after" {
		t.Errorf("after close: focus %q %v, want after", c, ok)
	}
	want := []string{"focus after", "blur after", "focus x", "blur x", "focus y", "blur y", "focus x", "blur x", "focus y", "blur y", "focus after"}
	if !slices.Equal(*log, want) {
		t.Errorf("events: got %q, want %q", *log, want)
	}
}

func TestFocusCloseWithoutPrevious(t *testing.T) {
	for _, change := range []string{"removed", "disabled"} {
		tr, log := dialogTree()
		var f FocusManager[string]
		f.Set(tr, "after")
		f.Open(tr, "dialog")
		tr.detach("dialog")
		if change == "removed" {
			tr.detach("after")
		} else {
			after := tr["after"]
			after.disabled = true
			tr["after"] = after
		}
		f.Close(tr)
		if c, ok := f.Current(); ok {
			t.Errorf("%s: focus %q, want none", change, c)
		}
		if last := (*log)[len(*log)-1]; last != "blur x" {
			t.Errorf("%s: last event %q, want blur x", change, last)
		}
	}
}
