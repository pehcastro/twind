package ui

import (
	"reflect"
	"slices"
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/input"
)

func attrs(n twi.Node, name string) []string {
	var found []string
	var walk func(reflect.Value)
	walk = func(v reflect.Value) {
		if own := v.FieldByName("State"); !own.IsNil() {
			list := own.Elem().FieldByName("Attrs")
			for i := range list.Len() {
				if list.Index(i).FieldByName("Name").String() == name {
					found = append(found, list.Index(i).FieldByName("Value").String())
				}
			}
		}
		children := v.FieldByName("Children")
		for i := range children.Len() {
			walk(children.Index(i))
		}
	}
	walk(rendered(n))
	return found
}

func TestOverlayPlacesWhereItFits(t *testing.T) {
	const width, height = 40, 10
	type placed struct {
		frame       drive.Frame
		side, align []string
	}
	run := func(page string, app func(*twi.Runtime) func() twi.Node) placed {
		var last twi.Node
		d := overlayDriver(t, width, height, func(rt *twi.Runtime) func() twi.Node {
			build := app(rt)
			return func() twi.Node {
				last = twi.Element(twi.Class(page+" h-full bg-background text-foreground"), build())
				return last
			}
		})
		p := placed{frame: d.Frame(), side: attrs(last, "data-side"), align: attrs(last, "data-align")}
		d.Advance(settleTime)
		if again := d.Frame().Text(); again != p.frame.Text() {
			t.Errorf("%s: the placement moves on an idle frame:\n%s\nthen\n%s", page, p.frame.Text(), again)
		}
		t.Logf("%s, data-side %v, data-align %v\n%s", page, p.side, p.align, p.frame.Text())
		return p
	}
	tooltip := func(page, hint string) placed {
		return run(page, func(rt *twi.Runtime) func() twi.Node {
			tip := NewTooltip(rt)
			tip.Open = true
			return func() twi.Node {
				return tip.Node(tip.Trigger(Outline, SizeDefault, twi.Text("Hover")), tip.Content(twi.Text(hint)))
			}
		})
	}

	top := tooltip("flex flex-row", "Add to library")
	_, by, _ := at(top.frame, "Hover")
	_, ty, shown := at(top.frame, "Add to library")
	if !shown || ty <= by || !slices.Equal(top.side, []string{"bottom"}) {
		t.Errorf("a tooltip on a button at the top edge opens below: button row %d, tooltip row %d (shown %v), data-side %v", by, ty, shown, top.side)
	}

	const wide = "A hint much wider than it"
	edge := tooltip("flex flex-col items-end justify-center", wide)
	_, by, _ = at(edge.frame, "Hover")
	tx, ty, shown := at(edge.frame, wide)
	if end := tx + len(wide); !shown || ty >= by || end > width-1 || !slices.Equal(edge.side, []string{"top"}) || !slices.Equal(edge.align, []string{"center"}) {
		t.Errorf("a tooltip at the right edge stays on top and shifts left inside the padding: button row %d, tooltip row %d, ends at column %d (shown %v), data-side %v, data-align %v", by, ty, end, shown, edge.side, edge.align)
	}

	bottom := run("flex flex-col justify-end", func(rt *twi.Runtime) func() twi.Node {
		m := NewDropdownMenu(rt)
		m.Align, m.Open = Start, true
		return func() twi.Node {
			return m.Node(m.Trigger(Outline, SizeDefault, twi.Text("Open")), m.Content(m.Item("Profile"), m.Item("Billing"), m.Item("Team"), m.Item("Log out")))
		}
	})
	_, oy, _ := at(bottom.frame, "Open")
	_, py, shown := at(bottom.frame, "Profile")
	_, ly, _ := at(bottom.frame, "Log out")
	if !shown || ly >= oy || py >= ly || !slices.Equal(bottom.side, []string{"top"}) || !slices.Equal(bottom.align, []string{"start"}) {
		t.Errorf("a dropdown near the bottom opens upward: trigger row %d, items rows %d to %d (shown %v), data-side %v, data-align %v", oy, py, ly, shown, bottom.side, bottom.align)
	}

	right := run("flex flex-row justify-end", func(rt *twi.Runtime) func() twi.Node {
		m := NewDropdownMenu(rt)
		m.Align, m.Open = End, true
		invite := m.Sub()
		invite.Open = true
		return func() twi.Node {
			return m.Node(m.Trigger(Outline, SizeDefault, twi.Text("Open")), m.Content(
				m.Item("Profile"),
				invite.Node(invite.Trigger("Invite"), invite.Content(invite.Item("Email"), invite.Item("Message"))),
				m.Item("Log out"),
			))
		}
	})
	ix, iy, _ := at(right.frame, "Invite")
	ex, ey, shown := at(right.frame, "Email")
	if !shown || ex+len("Message") >= ix || ey != iy || !slices.Equal(right.side, []string{"bottom", "left"}) || !slices.Equal(right.align, []string{"end", "start"}) {
		t.Errorf("a sub menu near the right edge opens left beside its item: item at column %d row %d, sub item at column %d row %d (shown %v), data-side %v, data-align %v", ix, iy, ex, ey, shown, right.side, right.align)
	}

	d := overlayDriver(t, width, height, func(rt *twi.Runtime) func() twi.Node {
		c := NewContextMenu(rt)
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col h-full bg-background text-foreground"),
				c.Node(twi.Class("grow"), c.Trigger(twi.Class("grow"), twi.Text("area")), c.Content(c.Item("Back"), c.Item("Forward"), c.Item("Reload"))))
		}
	})
	d.ClickWith(input.MouseRight, width-2, height-2)
	d.Advance(settleTime)
	fx, fy, shown := at(d.Frame(), "Forward")
	_, ry, _ := at(d.Frame(), "Reload")
	if !shown || fx+len("Forward") > width-1 || ry >= height-1 || ry <= fy {
		t.Errorf("a context menu opened in the bottom right corner stays on the screen: Forward at column %d row %d, Reload row %d (shown %v)\n%s", fx, fy, ry, shown, d.Frame().Text())
	}
	t.Logf("context menu from the bottom right corner:\n%s", d.Frame().Text())
}
