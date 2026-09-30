package ui

import (
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/theme"
)

func ringed(f drive.Frame, ring color.RGBA) bool {
	cells := f.Cells()
	for y := range cells.Height() {
		for x := range cells.Width() {
			if c := cells.At(x, y); strings.ContainsAny(c.Grapheme, "─│▄▌▀▐") && c.Fg.RGBA == ring {
				return true
			}
		}
	}
	return false
}

func TestButtonKeys(t *testing.T) {
	light := zinc(t, theme.Light)
	var one, two, link, enters int
	var heard []rune
	d := overlayDriver(t, 80, 12, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(light)
		count := func(n *int) twi.NodeOption { return twi.OnClick(func(*twi.Event) { *n++; rt.Invalidate() }) }
		bubbled := twi.OnKeyDown(func(e *twi.Event) {
			if e.Key.Key == input.KeyEnter {
				enters++
			}
		})
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-row items-start gap-2 p-1 h-full bg-background text-foreground"), listen(&heard), bubbled,
				Button(Default, SizeDefault, count(&one), twi.Text("One")),
				Button(Outline, SizeDefault, twi.Disabled(), count(&two), twi.Text("Two")),
				Button(Ghost, SizeDefault, twi.Text("Three")),
				PaginationLink(false, count(&link), twi.Text("4")),
			)
		}
	})
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: one %d, two %d, link %d, enters %d, heard %q:\n%s", what, one, two, link, enters, string(heard), d.Frame().Text())
		}
	}
	ring := light.Tokens[theme.Ring].RGBA
	expect("no ring before any key", !ringed(d.Frame(), ring))
	x, y, _ := at(d.Frame(), "One")
	d.Click(x, y)
	expect("a click presses One once and focuses it without the ring", one == 1 && !ringed(d.Frame(), ring))
	hit(d, "tab")
	expect("tab skips the disabled Two to Three and shows the ring", ringed(d.Frame(), ring))
	hit(d, "enter")
	expect("enter on Three, which has no press, bubbles to the page", one == 1 && two == 0 && enters == 1)
	hit(d, "shift+tab enter space")
	expect("shift+tab back to One, enter and space press it once each, and the keys bubble as in a browser", one == 3 && enters == 2 && string(heard) == " ")
	hit(d, "ctrl+enter x")
	expect("ctrl+enter and a letter do not press it, both reach the page", one == 3 && enters == 3 && string(heard) == " x")
	hit(d, "tab tab enter")
	expect("the pagination link takes focus and fires once on enter", link == 1 && enters == 4)
	x, y, _ = at(d.Frame(), "Two")
	d.Click(x, y)
	expect("a click on the disabled Two does nothing", two == 0)
}
