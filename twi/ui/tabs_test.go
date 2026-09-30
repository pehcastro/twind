package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/theme"
)

func TestTabsKeysAndClicks(t *testing.T) {
	light := zinc(t, theme.Light)
	var (
		tabs, side    *Tabs
		changes       []string
		arrows, after int
		field         *Input
	)
	d := overlayDriver(t, 80, 16, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(light)
		tabs, side, field = NewTabs(rt), NewTabs(rt), NewInput(rt)
		tabs.OnChange = func(v string) { changes = append(changes, v) }
		side.Orientation, side.Value = Vertical, "missing"
		heard := twi.OnKeyDown(func(e *twi.Event) {
			if e.Key.Key == input.KeyArrowUp || e.Key.Key == input.KeyArrowDown {
				arrows++
			}
		})
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"), heard,
				tabs.Node(
					tabs.List(tabs.Trigger("account", twi.Text("Account")), tabs.Trigger("password", twi.Text("Password")), tabs.Trigger("team", twi.Text("Team"))),
					tabs.Content("account", twi.Element(twi.Class("flex flex-col gap-1"), twi.Text("Account panel"), field.Node())),
					tabs.Content("password", twi.Text("Password panel")),
					tabs.Content("team", twi.Text("Team panel")),
				),
				Button(Outline, SizeDefault, twi.OnClick(func(*twi.Event) { after++ }), twi.Text("After")),
				side.Node(
					side.List(side.Trigger("general", twi.Text("General")), side.Trigger("billing", twi.Text("Billing"))),
					side.Content("general", twi.Text("General panel")),
					side.Content("billing", twi.Text("Billing panel")),
				),
			)
		}
	})
	has := func(s string) bool { return strings.Contains(d.Frame().Text(), s) }
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: value %q, changes %v, arrows %d, after %d:\n%s", what, tabs.Value, changes, arrows, after, d.Frame().Text())
		}
	}
	ring := light.Tokens[theme.Ring].RGBA
	expect("the first trigger is selected when no value is set, only its panel is drawn", tabs.Value == "account" && has("Account panel") && !has("Password panel"))
	expect("a value no trigger has selects the first trigger", side.Value == "general" && has("General panel") && !has("Billing panel"))
	hit(d, "tab")
	expect("tab focuses the list and rings the selected trigger", tabs.focused && ringed(d.Frame(), ring))
	hit(d, "right")
	expect("right selects the next trigger and shows its panel", tabs.Value == "password" && has("Password panel") && !has("Account panel"))
	hit(d, "right right")
	expect("right wraps from the last trigger to the first", tabs.Value == "account")
	hit(d, "left")
	expect("left wraps from the first trigger to the last", tabs.Value == "team")
	hit(d, "home")
	expect("home selects the first", tabs.Value == "account")
	hit(d, "end")
	expect("end selects the last", tabs.Value == "team")
	hit(d, "up down")
	expect("up and down are not taken by a horizontal list", arrows == 2 && tabs.Value == "team")
	hit(d, "tab enter")
	expect("the list is one tab stop, the next tab reaches the button after it", after == 1)
	x, y, _ := at(d.Frame(), "Password")
	d.Click(x, y)
	expect("a click selects its trigger and focuses the list without a ring", tabs.Value == "password" && tabs.focused && !ringed(d.Frame(), ring))
	d.Click(x, y)
	expect("a click on the selected trigger changes nothing", slices.Equal(changes, []string{"password", "team", "account", "team", "account", "team", "password"}))
	hit(d, "left")
	expect("a key after the click brings the ring back", tabs.Value == "account" && ringed(d.Frame(), ring))
	hit(d, "tab tab tab")
	expect("the hidden panels hold no tab stop: from the account list, tab reaches its input, the button, then the side list", side.focused)
	hit(d, "down")
	expect("a vertical list moves with down", side.Value == "billing" && has("Billing panel") && arrows == 2)
	hit(d, "right")
	expect("right does nothing in a vertical list", side.Value == "billing")
	expect("triggers do not pile up across frames", len(tabs.items) == 3 && len(side.items) == 2)
}
