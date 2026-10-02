package ui

import (
	"math"
	"slices"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

type Menubar struct {
	control
	menus  []*MenubarMenu
	active int
	within bool
}

func NewMenubar(rt *twi.Runtime) *Menubar { return &Menubar{control: control{rt: rt}} }

type MenubarMenu struct{ DropdownMenu }

func (b *Menubar) Menu() *MenubarMenu {
	m := &MenubarMenu{DropdownMenu{anchored: newAnchored(b.rt, Bottom, Start), bar: b}}
	m.root, m.sideOffset, m.alignOffset = &m.DropdownMenu, 1, -1
	b.menus = append(b.menus, m)
	return m
}

func (b *Menubar) Node(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center gap-1 rounded-md border bg-background px-1 shadow-xs", append([]twi.NodeOption{keyDown(b.rt, b.key)}, children...))
}

func (b *Menubar) open() *MenubarMenu {
	if i := slices.IndexFunc(b.menus, func(m *MenubarMenu) bool { return m.Open }); i >= 0 {
		return b.menus[i]
	}
	return nil
}

func (b *Menubar) key(k input.KeyEvent) bool {
	n := len(b.menus)
	if n == 0 {
		return false
	}
	m := b.open()
	if m == nil {
		switch {
		case k.Key == input.KeyArrowLeft || k.Key == input.KeyArrowRight:
			b.active = (b.active + arrow(k) + n) % n
		case k.Key == input.KeyHome:
			b.active = 0
		case k.Key == input.KeyEnd:
			b.active = n - 1
		case press(k) || k.Key == input.KeyArrowDown:
			b.menus[b.active].show(0)
		case k.Key == input.KeyArrowUp:
			b.menus[b.active].show(math.MaxInt)
		default:
			return false
		}
		return true
	}
	level, sub := m.deepest()
	opensSub := level.active < len(level.items) && level.items[level.active].sub != nil
	switch {
	case escape(k):
		m.dismiss()
	case k.Key == input.KeyTab:
		m.dismiss()
		return false
	case k.Key == input.KeyArrowLeft && sub != nil:
		sub.Open = false
		m.close(level)
	case k.Key == input.KeyArrowLeft || k.Key == input.KeyArrowRight && !opensSub:
		m.dismiss()
		b.active = (b.active + arrow(k) + n) % n
		b.menus[b.active].show(0)
	default:
		return level.key(k)
	}
	return true
}

func (m *MenubarMenu) Trigger(children ...twi.NodeOption) twi.Node {
	b := m.bar
	at := slices.Index(b.menus, m)
	classes := "flex flex-row items-center rounded-sm px-2 font-medium select-none"
	if m.Open || at == b.active && b.within {
		classes += " bg-accent text-accent-foreground"
	}
	focus := func(on bool) func() {
		return func() {
			b.within = on
			if on {
				b.active = at
			}
			b.rt.Invalidate()
		}
	}
	options := []twi.NodeOption{openState(m.Open), twi.OnFocus(focus(true)), twi.OnBlur(focus(false)),
		twi.OnPointerEnter(func() {
			if open := b.open(); open != nil && open != m {
				open.dismiss()
				b.active = at
				m.show(0)
			}
		}),
		b.click(func() {
			b.active = at
			if m.Open {
				m.dismiss()
				return
			}
			m.show(0)
		}),
	}
	if at == b.active || !b.within {
		options = append(options, twi.Focusable())
	}
	return part(classes, append(options, children...))
}

func (m *MenubarMenu) Content(children ...twi.NodeOption) twi.Node {
	return m.menu(m.anchor.Bounds(), "flex flex-col min-w-24 shrink-0 rounded-md border bg-popover text-popover-foreground shadow-md data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95", children)
}
