package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/theme"
)

func TestAccordionKeysAndClicks(t *testing.T) {
	light := zinc(t, theme.Light)
	var (
		single, multi *Accordion
		changes       [][]string
		field         *Input
		arrows        int
	)
	d := overlayDriver(t, 80, 30, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(light)
		single, multi, field = NewAccordion(rt), NewAccordion(rt), NewInput(rt)
		single.Collapsible, single.Value = true, []string{"product"}
		single.OnChange = func(v []string) { changes = append(changes, v) }
		multi.Type, multi.Value = Multiple, []string{"missing"}
		heard := twi.OnKeyDown(func(e *twi.Event) {
			if e.Key.Key == input.KeyArrowUp || e.Key.Key == input.KeyArrowDown {
				arrows++
			}
		})
		item := func(a *Accordion, value, title string, content ...twi.NodeOption) twi.Node {
			return a.Item(value, a.Trigger(value, twi.Text(title)), a.Content(value, content...))
		}
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"), heard,
				single.Node(
					item(single, "product", "Product Information", twi.Text("Product-panel"), field.Node()),
					item(single, "shipping", "Shipping Details", twi.Text("Shipping-panel")),
					item(single, "returns", "Return Policy", twi.Text("Returns-panel")),
				),
				multi.Node(
					item(multi, "a", "Alpha", twi.Text("Alpha-panel")),
					item(multi, "b", "Beta", twi.Text("Beta-panel")),
				),
			)
		}
	})
	has := func(s string) bool { return strings.Contains(d.Frame().Text(), s) }
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: single %v, multi %v, active %d, changes %v, arrows %d:\n%s", what, single.Value, multi.Value, single.active, changes, arrows, d.Frame().Text())
		}
	}
	ring := light.Tokens[theme.Ring].RGBA
	t.Logf("first frame, 80x30, twind light:\n%s", d.Frame().Text())
	expect("the default value opens its item only", has("Product-panel") && !has("Shipping-panel") && !has("Returns-panel"))
	expect("a value naming no item opens nothing", !has("Alpha-panel") && !has("Beta-panel"))
	expect("the open trigger shows the turned chevron, the closed ones the plain one", strings.Count(d.Frame().Text(), "⌃") == 1 && strings.Count(d.Frame().Text(), "⌄") == 4)
	hit(d, "tab")
	expect("tab focuses the accordion and rings the first trigger", single.focused && ringed(d.Frame(), ring))
	hit(d, "down")
	expect("down moves to the next trigger without opening it", single.active == 1 && has("Product-panel") && !has("Shipping-panel"))
	hit(d, "enter")
	expect("enter opens the active item and closes the open one in a single accordion", has("Shipping-panel") && !has("Product-panel") && slices.Equal(single.Value, []string{"shipping"}))
	hit(d, "space")
	expect("space closes it again in a collapsible accordion", !has("Shipping-panel") && len(single.Value) == 0)
	hit(d, "down down")
	expect("down wraps from the last trigger to the first", single.active == 0)
	hit(d, "up")
	expect("up wraps from the first trigger to the last", single.active == 2)
	hit(d, "home")
	expect("home moves to the first", single.active == 0)
	hit(d, "end")
	expect("end moves to the last", single.active == 2)
	hit(d, "ctrl+enter a")
	expect("ctrl+enter and a letter do nothing", len(single.Value) == 0 && len(changes) == 2)
	hit(d, "home enter")
	expect("home and enter open the first item again", has("Product-panel"))
	before := arrows
	hit(d, "tab")
	expect("tab reaches the input inside the open item", field.focused)
	hit(d, "down")
	expect("down inside the item's input is not taken by the accordion", single.active == 0 && arrows == before+1)
	hit(d, "tab")
	expect("the next tab leaves for the second accordion: the closed items hold no tab stop", multi.focused)
	hit(d, "enter down enter")
	expect("a multiple accordion keeps two items open", has("Alpha-panel") && has("Beta-panel") && slices.Equal(multi.Value, []string{"missing", "a", "b"}))
	hit(d, "up space")
	expect("one of them closes alone", !has("Alpha-panel") && has("Beta-panel"))
	click := func(s string) {
		x, y, _ := at(d.Frame(), s)
		d.Click(x, y)
	}
	click("Return Policy")
	expect("a click toggles its item and makes it active without the ring", has("Returns-panel") && !has("Product-panel") && single.active == 2 && single.focused && !ringed(d.Frame(), ring))
	click("Return Policy")
	expect("a second click closes it in a collapsible accordion", !has("Returns-panel") && len(single.Value) == 0)
	single.Collapsible = false
	click("Return Policy")
	click("Return Policy")
	expect("without Collapsible the open item stays open and OnChange does not fire again", has("Returns-panel") && len(changes) == 6)
	hit(d, "up")
	expect("a key after the click brings the ring back", single.active == 1 && ringed(d.Frame(), ring))
	expect("items do not pile up across frames", len(single.items) == 3 && len(multi.items) == 2)
	t.Logf("after the script:\n%s", d.Frame().Text())
}

func TestCollapsibleKeysAndClicks(t *testing.T) {
	var (
		c       *Collapsible
		changes []bool
	)
	d := overlayDriver(t, 60, 12, func(rt *twi.Runtime) func() twi.Node {
		c = NewCollapsible(rt)
		c.OnOpenChange = func(open bool) { changes = append(changes, open) }
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col p-1 h-full bg-background text-foreground"), c.Node(twi.Class("gap-1"),
				twi.Element(twi.Class("flex flex-row gap-2"), twi.Text("@peduarte starred 3 repositories"), c.Trigger(Ghost, SizeIcon, twi.Text("↕"))),
				twi.Text("@radix-ui/primitives"),
				c.Content(twi.Text("@radix-ui/colors"), twi.Text("@stitches/react")),
			))
		}
	})
	has := func(s string) bool { return strings.Contains(d.Frame().Text(), s) }
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: open %v, changes %v:\n%s", what, c.Open, changes, d.Frame().Text())
		}
	}
	expect("closed at first, the content is not in the tree", !has("@radix-ui/colors") && has("@radix-ui/primitives"))
	hit(d, "tab enter")
	expect("enter on the trigger opens it", c.Open && has("@radix-ui/colors") && has("@stitches/react"))
	hit(d, "space")
	expect("space closes it", !c.Open && !has("@radix-ui/colors"))
	hit(d, "a ctrl+enter")
	expect("a letter and ctrl+enter do nothing", !c.Open)
	x, y, _ := at(d.Frame(), "↕")
	d.Click(x, y)
	expect("a click opens it", c.Open && has("@radix-ui/colors"))
	n := c.Node(c.Trigger(Ghost, SizeIcon), c.Content())
	if got := []string{dataState(n, nil), dataState(n, []int{0}), dataState(n, []int{1})}; !slices.Equal(got, []string{"open", "open", "open"}) {
		t.Errorf("open root, trigger and content data-state %v", got)
	}
	d.Click(x, y)
	expect("a second click closes it, OnOpenChange fired once per change", !c.Open && slices.Equal(changes, []bool{true, false, true, false}))
}

func dataState(n twi.Node, path []int) string {
	v := rendered(n)
	for _, i := range path {
		v = v.FieldByName("Children").Index(i)
	}
	if own := v.FieldByName("State"); !own.IsNil() {
		attrs := own.Elem().FieldByName("Attrs")
		for i := range attrs.Len() {
			if attrs.Index(i).FieldByName("Name").String() == "data-state" {
				return attrs.Index(i).FieldByName("Value").String()
			}
		}
	}
	return ""
}

func TestAccordionStateAndBorders(t *testing.T) {
	light := zinc(t, theme.Light)
	var acc *Accordion
	d := overlayDriver(t, 60, 12, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(light)
		acc = NewAccordion(rt)
		acc.Value = []string{"one"}
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col p-1 h-full bg-background text-foreground"), acc.Node(
				acc.Item("one", acc.Trigger("one", twi.Text("One")), acc.Content("one", twi.Text("First-panel"))),
				acc.Item("two", acc.Trigger("two", twi.Text("Two")), acc.Content("two", twi.Text("Second-panel"))),
			))
		}
	})
	n := acc.Node(acc.Item("one", acc.Trigger("one"), acc.Content("one")), acc.Item("two", acc.Trigger("two"), acc.Content("two")))
	states := func(i int) []string {
		var got []string
		for _, p := range [][]int{{i}, {i, 0}, {i, 1}} {
			got = append(got, dataState(n, p))
		}
		return got
	}
	if got := states(0); !slices.Equal(got, []string{"open", "open", "open"}) {
		t.Errorf("open item, trigger and content data-state %v", got)
	}
	if got := states(1); !slices.Equal(got, []string{"closed", "closed", ""}) {
		t.Errorf("closed item and trigger data-state %v, and its content is the hidden placeholder", got)
	}
	border := light.Tokens[theme.Border].RGBA
	rows := strings.Split(d.Frame().Text(), "\n")
	var lines []int
	cells := d.Frame().Cells()
	for y := range cells.Height() {
		if c := cells.At(20, y); c.Fg.RGBA == border && strings.TrimSpace(c.Grapheme) != "" {
			lines = append(lines, y)
		}
	}
	if len(lines) != 1 {
		t.Errorf("one border line between two items and none under the last, got rows %v:\n%s", lines, strings.Join(rows, "\n"))
	}
}
