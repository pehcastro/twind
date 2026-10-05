package ui

import (
	"slices"
	"time"

	konst "github.com/pehcastro/twind/internal/konst/ui"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
)

type NavigationMenu struct {
	control
	Value        string
	OnSelect     func(string)
	items        []*NavigationMenuItem
	active, link int
	wait         *twi.Timer
}

type NavigationMenuItem struct {
	menu         *NavigationMenu
	value        string
	presence     presence
	links, built []string
}

func NewNavigationMenu(rt *twi.Runtime) *NavigationMenu {
	return &NavigationMenu{control: control{rt: rt}, link: -1}
}

func (n *NavigationMenu) Item(value string) *NavigationMenuItem {
	i := &NavigationMenuItem{menu: n, value: value}
	n.items = append(n.items, i)
	return i
}

func (n *NavigationMenu) Node(children ...twi.NodeOption) twi.Node {
	return part("relative flex flex-row w-fit", append([]twi.NodeOption{
		twi.OnPointerDownOutside(func() { n.show(nil) }),
		twi.OnPointerEnter(n.stop),
		twi.OnPointerLeave(func() { n.after(konst.HoverShut, nil) }),
	}, children...))
}

func (n *NavigationMenu) List(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center gap-1", append(n.behave(n.key), children...))
}

func (i *NavigationMenuItem) Node(children ...twi.NodeOption) twi.Node {
	return part("relative flex", children)
}

func (i *NavigationMenuItem) Trigger(children ...twi.NodeOption) twi.Node {
	n := i.menu
	at := slices.Index(n.items, i)
	open := n.Value == i.value
	return part("flex flex-row items-center gap-1 rounded-md bg-background px-2 font-medium select-none hover:bg-accent hover:text-accent-foreground data-[state=open]:bg-accent/50 data-[state=open]:text-accent-foreground "+n.ring("", onItem), append([]twi.NodeOption{
		openState(open), n.dataActive(at == n.active && n.link < 0),
		twi.OnPointerEnter(func() {
			if n.Value != "" {
				n.show(i)
				return
			}
			n.after(konst.HoverOpen, i)
		}),
		twi.OnPointerLeave(n.stop),
		n.click(func() {
			if open {
				n.show(nil)
				return
			}
			n.show(i)
		}),
	}, append(children, icon(map[bool]string{false: "⌄", true: "⌃"}[open], "text-muted-foreground"))...))
}

func (i *NavigationMenuItem) Content(children ...twi.NodeOption) twi.Node {
	i.links, i.built = i.built, nil
	at := i.presence.next(i.menu.rt, i.menu.Value == i.value)
	return part("absolute top-full left-0 z-50 flex pt-1", at.holding(func() twi.Node {
		return part("flex flex-col w-max shrink-0 rounded-md border bg-popover text-popover-foreground shadow "+popMotion, append([]twi.NodeOption{at.state()}, children...))
	}))
}

func (i *NavigationMenuItem) Link(href string, children ...twi.NodeOption) twi.Node {
	n := i.menu
	at := len(i.built)
	i.built = append(i.built, href)
	classes := "flex flex-col rounded-sm px-1 hover:bg-accent hover:text-accent-foreground"
	if n.Value == i.value && n.link == at {
		classes += " bg-accent text-accent-foreground"
	}
	return part(classes, append([]twi.NodeOption{
		twi.OnPointerEnter(func() {
			if n.link != at {
				n.link = at
				n.rt.Invalidate()
			}
		}),
		n.click(func() { n.choose(href) }),
	}, children...))
}

func (n *NavigationMenu) after(delay time.Duration, i *NavigationMenuItem) {
	n.stop()
	n.wait = n.rt.After(delay, func() { n.show(i) })
}

func (n *NavigationMenu) stop() {
	if n.wait != nil {
		n.wait.Stop()
		n.wait = nil
	}
}

func (n *NavigationMenu) show(i *NavigationMenuItem) {
	n.stop()
	value := ""
	if i != nil {
		value, n.active = i.value, slices.Index(n.items, i)
	}
	if n.Value != value {
		n.Value, n.link = value, -1
		n.rt.Invalidate()
	}
}

func (n *NavigationMenu) choose(href string) {
	n.show(nil)
	notify(n.OnSelect, href)
}

func (n *NavigationMenu) key(k input.KeyEvent) bool {
	count := len(n.items)
	if count == 0 {
		return false
	}
	open := slices.IndexFunc(n.items, func(i *NavigationMenuItem) bool { return i.value == n.Value })
	switch {
	case k.Key == input.KeyArrowLeft || k.Key == input.KeyArrowRight:
		n.active = (n.active + arrow(k) + count) % count
		if open >= 0 {
			n.show(n.items[n.active])
		}
	case k.Key == input.KeyHome:
		n.active = 0
	case k.Key == input.KeyEnd:
		n.active = count - 1
	case press(k) && open >= 0 && n.link >= 0:
		n.choose(n.items[open].links[n.link])
	case press(k) && open >= 0:
		n.show(nil)
	case press(k) || k.Key == input.KeyArrowDown && open < 0:
		n.show(n.items[n.active])
	case k.Key == input.KeyArrowDown:
		n.link = min(n.link+1, len(n.items[open].links)-1)
	case k.Key == input.KeyArrowUp && open >= 0:
		n.link = max(n.link-1, -1)
	case escape(k) && open >= 0:
		n.show(nil)
	case k.Key == input.KeyTab:
		n.show(nil)
		return false
	default:
		return false
	}
	return true
}
