package ui

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/theme"
)

const visibleStep = 0.04

func oklabDistance(x, y color.RGBA) float64 {
	lab := func(c color.RGBA) [3]float64 {
		lin := func(v uint8) float64 {
			s := float64(v) / 255
			if s <= 0.04045 {
				return s / 12.92
			}
			return math.Pow((s+0.055)/1.055, 2.4)
		}
		r, g, b := lin(c.R), lin(c.G), lin(c.B)
		l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
		m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
		s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
		return [3]float64{
			0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
			1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
			0.0259040371*l + 0.7827717662*m - 0.8086757660*s,
		}
	}
	a, b := lab(x), lab(y)
	return math.Sqrt((a[0]-b[0])*(a[0]-b[0]) + (a[1]-b[1])*(a[1]-b[1]) + (a[2]-b[2])*(a[2]-b[2]))
}

func TestTabsActiveStandsOutInEveryTheme(t *testing.T) {
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	hex := func(c color.RGBA) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }
	table := "theme, scheme, page, list, active, inactive text, active text, step\n"
	for _, th := range theme.Builtin() {
		d := drive.New(func(rt *twi.Runtime) func() twi.Node {
			rt.SetTheme(th)
			tabs := NewTabs(rt)
			return func() twi.Node {
				return twi.Element(twi.Class("flex flex-col p-1 h-full bg-background text-foreground"),
					tabs.Node(tabs.List(tabs.Trigger("preview", twi.Text("Preview")), tabs.Trigger("code", twi.Text("Code")))))
			}
		}, drive.Size(30, 4), drive.With(twi.Styles(sheet)))
		d.Advance(settleTime)
		f := d.Frame()
		ax, ay, _ := at(f, "Preview")
		ix, iy, _ := at(f, "Code")
		active, list, page := f.Cells().At(ax, ay), f.Cells().At(ix, iy), f.Cells().At(0, 0)
		step := oklabDistance(active.Bg.RGBA, list.Bg.RGBA)
		table += fmt.Sprintf("%s, %d, %s, %s, %s, %s, %s, %.3f\n", th.Name, th.Scheme, hex(page.Bg.RGBA), hex(list.Bg.RGBA), hex(active.Bg.RGBA), hex(list.Fg.RGBA), hex(active.Fg.RGBA), step)
		if step < visibleStep {
			t.Errorf("%s %d: active trigger %s on list %s, step %.3f, want at least %.3f", th.Name, th.Scheme, hex(active.Bg.RGBA), hex(list.Bg.RGBA), step, visibleStep)
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}
	t.Log("\n" + table)
}

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
