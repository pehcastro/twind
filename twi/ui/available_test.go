package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
)

func longMenu(rt *twi.Runtime, page string, count int) func() twi.Node {
	m := NewDropdownMenu(rt)
	m.Align = AlignStart
	return func() twi.Node {
		items := make([]twi.NodeOption, count)
		for i := range items {
			items[i] = m.Item(fmt.Sprintf("Item %02d", i+1))
		}
		return twi.Element(twi.Class(page+" flex flex-col p-1 h-full bg-background text-foreground"),
			m.Node(m.Trigger(ButtonOutline, ButtonSizeDefault, twi.Text("Open")), m.Content(items...)))
	}
}

func TestArrowsRevealTheHighlightInAnAvailableHeightList(t *testing.T) {
	names := make([]string, 25)
	for i := range names {
		names[i] = fmt.Sprintf("Item %02d", i+1)
	}
	page := func(child twi.Node) twi.Node {
		return twi.Element(twi.Class("flex flex-col items-start p-1 h-full bg-background text-foreground"), child)
	}
	for _, c := range []struct {
		name, trigger string
		app           drive.App
	}{
		{"menu", "Open", func(rt *twi.Runtime) func() twi.Node { return longMenu(rt, "", 25) }},
		{"select", "Pick", func(rt *twi.Runtime) func() twi.Node {
			s := NewSelect(rt)
			s.Placeholder = "Pick"
			return func() twi.Node {
				items := make([]twi.NodeOption, len(names))
				for i, n := range names {
					items[i] = s.Item(n, n)
				}
				return page(s.Node(s.Trigger(twi.Class("w-20")), s.Content(items...)))
			}
		}},
		{"native select", "Item 01", func(rt *twi.Runtime) func() twi.Node {
			s := NewNativeSelect(rt)
			s.Options, s.Value = names, names[0]
			return func() twi.Node { return page(s.Node(twi.Class("w-20"))) }
		}},
		{"combobox", "Pick", func(rt *twi.Runtime) func() twi.Node {
			c := NewCombobox(rt)
			c.Placeholder = "Pick"
			return func() twi.Node {
				items := make([]twi.NodeOption, len(names))
				for i, n := range names {
					items[i] = c.Item(n, n)
				}
				return page(c.Node(twi.Class("w-24"), c.Input(), c.Content(items...)))
			}
		}},
	} {
		d := overlayDriver(t, 40, 20, c.app)
		x, y, _ := at(d.Frame(), c.trigger)
		settledClick(d, x+1, y)
		if rowOf(d.Frame(), "Item 21") >= 0 {
			t.Fatalf("%s: the list is not capped, item 21 shows on opening:\n%s", c.name, d.Frame().Text())
		}
		hit(d, strings.Repeat("down ", 20))
		if rowOf(d.Frame(), "Item 21") < 0 {
			t.Errorf("%s: twenty downs leave the highlight on item 21 below the fold:\n%s", c.name, d.Frame().Text())
		} else {
			t.Logf("%s after twenty downs:\n%s", c.name, d.Frame().Text())
		}
		hit(d, strings.Repeat("up ", 20))
		if rowOf(d.Frame(), "Item 01") < 0 {
			t.Errorf("%s: twenty ups leave the highlight on item 01 above the fold:\n%s", c.name, d.Frame().Text())
		}
	}
}

func rowOf(f drive.Frame, s string) int {
	_, y, ok := at(f, s)
	if !ok {
		return -1
	}
	return y
}

func TestMenuTakesTheAvailableRoomBelowItsTrigger(t *testing.T) {
	d := overlayDriver(t, 40, 20, func(rt *twi.Runtime) func() twi.Node { return longMenu(rt, "", 25) })
	x, trigger, _ := at(d.Frame(), "Open")
	settledClick(d, x+1, trigger)
	opened := d.Frame()
	first, last := rowOf(opened, "Item 01"), rowOf(opened, "Item 25")
	switch {
	case rowOf(opened, "Open") != trigger:
		t.Errorf("the menu covers its trigger on row %d:\n%s", trigger, opened.Text())
	case first <= trigger:
		t.Errorf("the first item is on row %d, not below the trigger on row %d:\n%s", first, trigger, opened.Text())
	case rowOf(opened, "╰") < first:
		t.Errorf("the menu has no bottom border on screen:\n%s", opened.Text())
	case last >= 0:
		t.Errorf("all 25 items fit a 20-row screen:\n%s", opened.Text())
	}
	t.Logf("opened:\n%s", opened.Text())
	d.Wheel(x+2, first, 30)
	d.Advance(settleTime)
	scrolled := d.Frame()
	if last := rowOf(scrolled, "Item 25"); last <= trigger || rowOf(scrolled, "Open") != trigger {
		t.Errorf("the wheel does not bring the last item into the room below the trigger, row %d:\n%s", last, scrolled.Text())
	}
	t.Logf("scrolled to the end:\n%s", scrolled.Text())
	d.Advance(frameStep)
	if again := d.Frame().Text(); again != scrolled.Text() {
		t.Errorf("the menu moves on a frame with no input:\n%s", again)
	}
}

func TestMenuTakesTheAvailableRoomAboveWhenItIsLarger(t *testing.T) {
	d := overlayDriver(t, 40, 20, func(rt *twi.Runtime) func() twi.Node { return longMenu(rt, "justify-end pb-3", 12) })
	x, trigger, _ := at(d.Frame(), "Open")
	settledClick(d, x+1, trigger)
	f := d.Frame()
	if first, last := rowOf(f, "Item 01"), rowOf(f, "Item 12"); first < 0 || last < 0 || last >= trigger {
		t.Errorf("the menu does not open whole above its trigger on row %d (first %d, last %d):\n%s", trigger, first, last, f.Text())
	}
	t.Logf("flipped above:\n%s", f.Text())
	d.Advance(frameStep)
	if again := d.Frame().Text(); again != f.Text() {
		t.Errorf("the menu moves on a frame with no input:\n%s", again)
	}
}

func TestSelectListAnchorWidthInAClippingCard(t *testing.T) {
	d := overlayDriver(t, 40, 16, func(rt *twi.Runtime) func() twi.Node {
		s := NewSelect(rt)
		s.Placeholder = "Fruit"
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-row p-1 h-full bg-background text-foreground"),
				twi.Element(twi.Class("flex flex-row h-5 items-center overflow-hidden rounded-lg border px-1"),
					s.Node(s.Trigger(twi.Class("w-24")), s.Content(s.Item("fig", "Fig"), s.Item("kiwi", "Kiwi")))))
		}
	})
	x, y, _ := at(d.Frame(), "Fruit")
	settledClick(d, x+1, y)
	f := d.Frame()
	lx, ly, ok := at(f, "Kiwi")
	if !ok || ly < 6 {
		t.Fatalf("the list is clipped by its card:\n%s", f.Text())
	}
	line := []rune(strings.Split(f.Text(), "\n")[ly])
	left, right := lx, lx
	for left > 0 && line[left] != '│' {
		left--
	}
	for right < len(line)-1 && line[right] != '│' {
		right++
	}
	if width := right - left + 1; width != 24 {
		t.Errorf("the list is %d wide, not its trigger's 24:\n%s", width, f.Text())
	}
	t.Logf("the list at its trigger's width:\n%s", f.Text())
}
