package twi

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/runtime"
)

func paths(n runtime.Node) []string {
	var out []string
	for _, c := range n.Children {
		var parts []string
		for _, i := range c.At {
			parts = append(parts, strconv.Itoa(i))
		}
		out = append(out, strings.Join(parts, "."))
	}
	return out
}

func TestZeroNodeIsAnEmptyElement(t *testing.T) {
	var zero Node
	if tree := zero.runtimeTree(); tree.Root.Children != nil || tree.Root.Classes != nil || tree.Keys != nil {
		t.Errorf("zero node tree %+v, want empty", tree)
	}
	tree := Element(Class("a"), zero, Text("t")).runtimeTree()
	if len(tree.Root.Children) != 2 || tree.Root.Children[0].Classes != nil || tree.Root.Children[1].Text != "t" {
		t.Errorf("children %+v, want an empty element then the text", tree.Root.Children)
	}
}

func TestReusedChildKeepsItsOwnPaths(t *testing.T) {
	behaving := Element(Class("c"), Focusable())
	plain := Element(Text("p"), behaving)
	first := Element(behaving, plain, behaving)
	want := []string{"0", "1.1", "2"}
	if got := paths(first.runtimeTree().Events); !slices.Equal(got, want) {
		t.Fatalf("paths %q, want %q", got, want)
	}
	second := Element(Text("x"), Text("y"), plain)
	if got, want2 := paths(second.runtimeTree().Events), []string{"2.1"}; !slices.Equal(got, want2) {
		t.Errorf("second parent paths %q, want %q", got, want2)
	}
	if got := paths(first.runtimeTree().Events); !slices.Equal(got, want) {
		t.Errorf("first parent paths after reuse %q, want %q", got, want)
	}
	if got := paths(plain.runtimeTree().Events); !slices.Equal(got, []string{"1"}) {
		t.Errorf("the reused child's own paths %q, want [1]", got)
	}
	if !behaving.runtimeTree().Events.Focusable {
		t.Error("the behaving child lost its own events")
	}
}

func TestReusedClassOptionIsNotShared(t *testing.T) {
	shared := Class("a b")
	one := Element(shared)
	two := Element(shared, Class("c"))
	three := Element(shared, Class("d"))
	for _, c := range []struct {
		n    Node
		want []string
	}{{one, []string{"a", "b"}}, {two, []string{"a", "b", "c"}}, {three, []string{"a", "b", "d"}}} {
		if got := c.n.runtimeTree().Root.Classes; !slices.Equal(got, c.want) {
			t.Errorf("classes %q, want %q", got, c.want)
		}
	}
}

func TestClassesKeepTheirOrder(t *testing.T) {
	got := Element(Class("a"), Text("t"), Class(" b  c "), Class("d", "e f"), Class(""), Class("  ")).runtimeTree().Root.Classes
	if want := []string{"a", "b", "c", "d", "e", "f"}; !slices.Equal(got, want) {
		t.Errorf("classes %q, want %q", got, want)
	}
}

func TestEmptyClassesAndLeavesStayNil(t *testing.T) {
	for name, n := range map[string]Node{"text": Text("x"), "element": Element(), "blank class": Element(Class(""), Class("   "))} {
		if root := n.runtimeTree().Root; root.Classes != nil || root.Children != nil {
			t.Errorf("%s: classes %#v children %#v, want both nil", name, root.Classes, root.Children)
		}
	}
}

func TestEachHandlerLandsInItsOwnList(t *testing.T) {
	var got []string
	call := func(name string) func() { return func() { got = append(got, name) } }
	event := func(name string) func(*Event) { return func(*Event) { got = append(got, name) } }
	e := Element(OnKeyDown(event("keydown")), OnFocus(call("focus")), OnBlur(call("blur")), OnClick(event("click")),
		OnPointerDown(event("pointerdown")), OnPointerEnter(call("enter")), OnPointerLeave(call("leave")),
		OnPointerDownOutside(call("downoutside")), OnFocusOutside(call("focusoutside")), TopLayer(), Disabled()).runtimeTree()
	ev := e.Events
	for _, list := range [][]listener{ev.KeyDown, ev.Focus, ev.Blur, ev.Click, ev.PointerDown, ev.Enter, ev.Leave} {
		if len(list) != 1 {
			t.Fatalf("a listener list holds %d, want 1", len(list))
		}
		list[0].Handle(nil)
	}
	for _, list := range [][]func(){ev.PointerDownOutside, ev.FocusOutside} {
		if len(list) != 1 {
			t.Fatalf("an outside list holds %d, want 1", len(list))
		}
		list[0]()
	}
	if want := []string{"keydown", "focus", "blur", "click", "pointerdown", "enter", "leave", "downoutside", "focusoutside"}; !slices.Equal(got, want) {
		t.Errorf("handlers ran %q, want %q", got, want)
	}
	if !ev.TopLayer || !ev.Disabled || e.Root.TopLayer != 1 || e.Root.State == nil {
		t.Errorf("top layer %v disabled %v root %+v, want both set on the events and the tree", ev.TopLayer, ev.Disabled, e.Root)
	}
}

func TestOwnKeysRunBeforeChildKeys(t *testing.T) {
	var got []string
	key := func(name string) NodeOption { return OnKey(func(input.KeyEvent) { got = append(got, name) }) }
	tree := Element(key("p1"), Element(key("c1"), Element(key("g1"))), key("p2"), Element(key("c2"))).runtimeTree()
	for _, k := range tree.Keys {
		k(input.KeyEvent{})
	}
	if want := []string{"p1", "p2", "c1", "g1", "c2"}; !slices.Equal(got, want) {
		t.Errorf("keys ran %q, want %q", got, want)
	}
}
