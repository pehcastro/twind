package ui

import (
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/theme"
)

func ringApp(t *testing.T, hidden bool) *drive.Driver {
	return overlayDriver(t, 80, 20, func(rt *twi.Runtime) func() twi.Node {
		if hidden {
			rt.HideFocusRings()
		}
		tabs, sw, cb, tg, menu, field := NewTabs(rt), NewSwitch(rt), NewCheckbox(rt), NewToggle(rt), NewDropdownMenu(rt), NewInput(rt)
		radio, group, slider, search := NewRadioGroup(rt), NewToggleGroup(rt), NewSlider(rt), NewInput(rt)
		field.Placeholder, search.Placeholder = "Field", "Search"
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
				tabs.Node(tabs.List(tabs.Trigger("account", twi.Text("Account")), tabs.Trigger("password", twi.Text("Password")))),
				twi.Element(twi.Class("flex flex-row gap-2"), Button(ButtonOutline, ButtonSizeDefault, twi.Text("Press")), sw.Node(twi.Text("S")), cb.Node(twi.Text("C")), tg.Node(twi.Text("Tg"))),
				menu.Node(menu.Trigger(ButtonOutline, ButtonSizeDefault, twi.Text("Menu")), menu.Content(menu.Item("Profile"), menu.Item("Billing"))),
				twi.Element(twi.Class("flex flex-row gap-2"), radio.Node(radio.Item("r", twi.Text("Radio"))), group.Node(group.Item("g", twi.Text("Group"))), twi.Element(twi.Class("w-10"), slider.Node(twi.Text("Slide")))),
				twi.Element(twi.Class("flex flex-row gap-2"), twi.Element(twi.Class("w-20"), field.Node()), twi.Element(twi.Class("w-20"), search.Group())),
				twi.Text("plain"),
			)
		}
	})
}

func TestRingsOnlyFromTheKeyboard(t *testing.T) {
	ring := zinc(t, theme.Light).Tokens[theme.Ring].RGBA
	d := ringApp(t, false)
	expect := func(step string, want bool) {
		t.Helper()
		if got := ringed(d.Frame(), ring); got != want {
			t.Errorf("%s: ring %v, want %v:\n%s", step, got, want, d.Frame().Text())
		}
	}
	for _, name := range []string{"Password", "Press", "S", "C", "Tg", "Menu", "Radio", "Group", "Slide"} {
		x, y, _ := at(d.Frame(), name)
		d.Down(x, y)
		d.Advance(settleTime)
		expect(name+": the press frame", false)
		d.Up(x, y)
		d.Advance(settleTime)
		expect(name+": the release frame", false)
		if name == "Menu" {
			x, y, _ = at(d.Frame(), "Billing")
			settledClick(d, x, y)
			expect("Menu: a click on a menu item hands focus back without a ring", false)
		}
		hit(d, "shift+tab tab")
		expect(name+": shift+tab then tab back onto it", true)
		x, y, _ = at(d.Frame(), "plain")
		settledClick(d, x, y)
		expect(name+": a press on plain text after the keyboard", false)
	}
	x, y, _ := at(d.Frame(), "Password")
	d.Down(x, y)
	d.Advance(settleTime)
	x, y, _ = at(d.Frame(), "plain")
	d.Up(x, y)
	d.Advance(settleTime)
	expect("a tab trigger pressed and released elsewhere", false)
	for _, name := range []string{"Field", "Search"} {
		x, y, _ = at(d.Frame(), name)
		settledClick(d, x, y)
		expect("a click on the text field "+name+" rings its border, as browsers do", true)
		x, y, _ = at(d.Frame(), "plain")
		settledClick(d, x, y)
	}
}

func TestHideFocusRingsEverywhere(t *testing.T) {
	ring := zinc(t, theme.Light).Tokens[theme.Ring].RGBA
	d := ringApp(t, true)
	for i := range 12 {
		hit(d, "tab")
		if ringed(d.Frame(), ring) {
			t.Errorf("tab %d with rings hidden shows a ring:\n%s", i+1, d.Frame().Text())
		}
	}
	x, y, _ := at(d.Frame(), "Field")
	settledClick(d, x, y)
	if ringed(d.Frame(), ring) {
		t.Errorf("a text field click with rings hidden shows a ring:\n%s", d.Frame().Text())
	}
}
