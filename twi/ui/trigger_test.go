package ui

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
)

func opensFromAnywhere(t *testing.T, d *drive.Driver, label, chevron string, open func() bool, shut func(x, y int)) {
	t.Helper()
	x, y, ok := at(d.Frame(), label)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", label, d.Frame().Text())
	}
	type spot struct {
		name string
		x, y int
	}
	spots := []spot{{"the label text", x + 1, y}, {"the padding", x - 1, y}}
	if chevron != "" {
		cx, cy, _ := at(d.Frame(), chevron)
		spots = append(spots, spot{"the chevron", cx, cy})
	}
	for i, s := range spots {
		settledClick(d, s.x, s.y)
		if !open() {
			t.Errorf("a click on %s at %d,%d does not open:\n%s", s.name, s.x, s.y, d.Frame().Text())
			continue
		}
		if i == 0 {
			t.Logf("after a click on %s:\n%s", s.name, d.Frame().Text())
		}
		shut(s.x, s.y)
		if open() {
			t.Fatalf("still open after closing from %s:\n%s", s.name, d.Frame().Text())
		}
	}
}

func opensFromKeys(t *testing.T, d *drive.Driver, open func() bool, keys ...string) {
	t.Helper()
	for _, k := range keys {
		hit(d, k)
		if !open() {
			t.Errorf("%s on the focused trigger does not open:\n%s", k, d.Frame().Text())
			continue
		}
		hit(d, "escape")
	}
}

func TestNativeSelectOpensFromTheWholeTrigger(t *testing.T) {
	var (
		s       *NativeSelect
		changes []string
	)
	d := overlayDriver(t, 60, 14, func(rt *twi.Runtime) func() twi.Node {
		s = NewNativeSelect(rt)
		s.Options, s.Value = []string{"Todo", "In Progress", "Done", "Cancelled"}, "Todo"
		s.OnChange = func(v string) { changes = append(changes, v) }
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"), s.Node(twi.Class("w-30")), twi.Element(twi.Text("Elsewhere")))
		}
	})
	opens := func() bool {
		text := d.Frame().Text()
		return strings.Contains(text, "In Progress") && strings.Contains(text, "Cancelled")
	}
	opensFromAnywhere(t, d, "Todo", "⌄", opens, func(x, y int) { settledClick(d, x, y) })
	opensFromKeys(t, d, opens, "enter", "space", "alt+down")
	if s.Value != "Todo" || len(changes) > 0 {
		t.Fatalf("opening changed the value to %q, changes %v", s.Value, changes)
	}
	hit(d, "space down enter")
	if opens() || s.Value != "In Progress" {
		t.Errorf("down and enter in the open list: value %q:\n%s", s.Value, d.Frame().Text())
	}
	hit(d, "down")
	if opens() || s.Value != "Done" {
		t.Errorf("down on the closed trigger steps the value: value %q", s.Value)
	}
	x, y, _ := at(d.Frame(), "Done")
	settledClick(d, x, y)
	cx, cy, _ := at(d.Frame(), "Cancelled")
	settledClick(d, cx, cy)
	if opens() || s.Value != "Cancelled" {
		t.Errorf("a click on an option: value %q:\n%s", s.Value, d.Frame().Text())
	}
	if want := []string{"In Progress", "Done", "Cancelled"}; !slices.Equal(changes, want) {
		t.Errorf("OnChange calls %v, want %v", changes, want)
	}
}

func TestComboboxOpensFromTheTextArea(t *testing.T) {
	var c *Combobox
	d := overlayDriver(t, 60, 14, func(rt *twi.Runtime) func() twi.Node {
		c = NewCombobox(rt)
		c.Placeholder = "Select framework..."
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
				c.Node(twi.Class("w-40"), c.Input(), c.Content(c.Item("next", "Next.js"), c.Item("svelte", "SvelteKit"), c.Item("astro", "Astro"))))
		}
	})
	opens := func() bool { return c.Open && c.field.focused && strings.Contains(d.Frame().Text(), "SvelteKit") }
	opensFromAnywhere(t, d, "Select framework", "⌄", opens, func(int, int) { hit(d, "escape") })
	x, y, _ := at(d.Frame(), "Select framework")
	settledClick(d, x, y)
	hit(d, "type:ast")
	if text := d.Frame().Text(); !c.Open || c.field.Value() != "ast" || !strings.Contains(text, "Astro") || strings.Contains(text, "SvelteKit") {
		t.Errorf("typing after a click on the text area: value %q, open %v:\n%s", c.field.Value(), c.Open, text)
	}
}

func TestSelectOpensFromTheWholeTrigger(t *testing.T) {
	var s *Select
	d := overlayDriver(t, 60, 14, func(rt *twi.Runtime) func() twi.Node {
		s = NewSelect(rt)
		s.Placeholder = "Select a fruit"
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
				s.Node(s.Trigger(twi.Class("w-30")), s.Content(s.Item("apple", "Apple"), s.Item("grapes", "Grapes"))))
		}
	})
	opens := func() bool { return s.Open && strings.Contains(d.Frame().Text(), "Grapes") }
	opensFromAnywhere(t, d, "Select a fruit", "⌄", opens, func(x, y int) { settledClick(d, x, y) })
	opensFromKeys(t, d, opens, "enter", "space", "alt+down")
}

func TestDropdownMenuOpensFromTheWholeTrigger(t *testing.T) {
	var m *DropdownMenu
	d := overlayDriver(t, 60, 14, func(rt *twi.Runtime) func() twi.Node {
		m = NewDropdownMenu(rt)
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
				m.Node(m.Trigger(ButtonOutline, ButtonSizeDefault, twi.Text("Open menu"), icon("⌄", "")), m.Content(m.Item("Profile"), m.Item("Settings"))))
		}
	})
	opens := func() bool { return m.Open && strings.Contains(d.Frame().Text(), "Settings") }
	opensFromAnywhere(t, d, "Open menu", "⌄", opens, func(x, y int) { settledClick(d, x, y) })
	opensFromKeys(t, d, opens, "enter", "space", "alt+down")
}

func TestMenubarOpensFromTheWholeTrigger(t *testing.T) {
	var file *MenubarMenu
	d := overlayDriver(t, 60, 14, func(rt *twi.Runtime) func() twi.Node {
		bar := NewMenubar(rt)
		file = bar.Menu()
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
				bar.Node(file.Node(file.Trigger(twi.Text("File")), file.Content(file.Item("New Tab"), file.Item("Print")))))
		}
	})
	opensFromAnywhere(t, d, "File", "", func() bool { return file.Open && strings.Contains(d.Frame().Text(), "Print") }, func(x, y int) { settledClick(d, x, y) })
}

func TestNavigationMenuOpensFromTheWholeTrigger(t *testing.T) {
	var n *NavigationMenu
	d := overlayDriver(t, 60, 14, func(rt *twi.Runtime) func() twi.Node {
		n = NewNavigationMenu(rt)
		docs := n.Item("docs")
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
				n.Node(n.List(docs.Node(docs.Trigger(twi.Text("Getting started")), docs.Content(docs.Link("/install", twi.Text("Installation")))))))
		}
	})
	opensFromAnywhere(t, d, "Getting started", "⌄", func() bool { return n.Value == "docs" && strings.Contains(d.Frame().Text(), "Installation") }, func(x, y int) {
		settledClick(d, x, y)
		d.Move(0, 0)
		d.Advance(time.Second)
	})
}

func TestListsLeaveAClippingCard(t *testing.T) {
	var (
		native *NativeSelect
		sel    *Select
		combo  *Combobox
	)
	d := overlayDriver(t, 80, 16, func(rt *twi.Runtime) func() twi.Node {
		native, sel, combo = NewNativeSelect(rt), NewSelect(rt), NewCombobox(rt)
		native.Options, native.Value = []string{"Todo", "In Progress", "Done", "Cancelled"}, "Todo"
		sel.Placeholder, combo.Placeholder = "Pick a fruit", "Pick a framework"
		card := func(child twi.Node) twi.Node {
			return twi.Element(twi.Class("flex flex-row h-5 items-center overflow-hidden rounded-lg border px-1"), child)
		}
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-row gap-1 p-1 h-full bg-background text-foreground"),
				card(native.Node(twi.Class("w-20"))),
				card(sel.Node(sel.Trigger(twi.Class("w-20")), sel.Content(sel.Item("apple", "Apple"), sel.Item("kiwi", "Kiwi"), sel.Item("mango", "Mango"), sel.Item("plum", "Plum")))),
				card(combo.Node(twi.Class("w-24"), combo.Input(), combo.Content(combo.Item("next", "Next.js"), combo.Item("nuxt", "Nuxt.js"), combo.Item("remix", "Remix"), combo.Item("astro", "Astro")))),
			)
		}
	})
	for _, c := range []struct{ label, last string }{{"Todo", "Cancelled"}, {"Pick a fruit", "Plum"}, {"Pick a framework", "Astro"}} {
		x, y, _ := at(d.Frame(), c.label)
		settledClick(d, x+1, y)
		lx, ly, ok := at(d.Frame(), c.last)
		switch {
		case !ok || ly < 6:
			t.Errorf("the %s list is clipped by its card:\n%s", c.label, d.Frame().Text())
		case lx < x-1:
			t.Errorf("the %s list starts left of its trigger at %d, not %d:\n%s", c.label, lx, x, d.Frame().Text())
		default:
			t.Logf("the %s list over its card:\n%s", c.label, d.Frame().Text())
		}
		hit(d, "escape")
	}
}

func TestNativeSelectListFlipsAboveNearTheBottom(t *testing.T) {
	d := overlayDriver(t, 40, 10, func(rt *twi.Runtime) func() twi.Node {
		s := NewNativeSelect(rt)
		s.Options, s.Value = []string{"Todo", "In Progress", "Done", "Cancelled"}, "Todo"
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col justify-end p-1 h-full bg-background text-foreground"),
				twi.Element(twi.Class("flex flex-row overflow-hidden rounded-lg border px-1"), s.Node(twi.Class("w-20"))))
		}
	})
	x, y, _ := at(d.Frame(), "Todo")
	settledClick(d, x+1, y)
	if _, cy, ok := at(d.Frame(), "Cancelled"); !ok || cy >= y {
		t.Errorf("the list near the bottom does not open above its trigger on row %d:\n%s", y, d.Frame().Text())
	}
	t.Logf("flipped above:\n%s", d.Frame().Text())
}

func TestDatePickerOpensFromTheWholeTrigger(t *testing.T) {
	var pop *Popover
	d := overlayDriver(t, 60, 40, func(rt *twi.Runtime) func() twi.Node {
		cal := NewCalendar(rt)
		pop = NewPopover(rt)
		pop.Align = AlignStart
		cal.Today = time.Date(2026, time.October, 4, 0, 0, 0, 0, time.UTC)
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
				pop.Node(pop.Trigger(ButtonOutline, ButtonSizeDefault, twi.Text("Pick a date"), icon("⌄", "")), pop.Content(cal.Node())))
		}
	})
	opens := func() bool { return pop.Open && strings.Contains(d.Frame().Text(), "October 2026") }
	opensFromAnywhere(t, d, "Pick a date", "⌄", opens, func(x, y int) { settledClick(d, x, y) })
	opensFromKeys(t, d, opens, "enter", "space")
}
