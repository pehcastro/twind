package ui

import (
	"image"
	"slices"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

const menuContent = "flex flex-col min-w-16 shrink-0 rounded-md border bg-popover px-1 text-popover-foreground shadow-md " + popMotion

type menuItem struct {
	text    string
	checked *bool
	group   *string
	sub     *DropdownMenuSub
}

type menuLevel struct {
	root         *DropdownMenu
	active       int
	items, built []menuItem
}

type DropdownMenu struct {
	anchored
	menuLevel
	OnSelect func(string)
	subs     []*DropdownMenuSub
	bar      *Menubar
}

func NewDropdownMenu(rt *twi.Runtime) *DropdownMenu {
	m := &DropdownMenu{anchored: anchored{overlay: overlay{control: control{rt: rt}}}}
	m.root = m
	return m
}

func (m *DropdownMenu) Node(children ...twi.NodeOption) twi.Node {
	return part("relative flex w-fit h-fit", append([]twi.NodeOption{twi.OnPointerDownOutside(m.dismiss)}, children...))
}

func (m *DropdownMenu) Trigger(v Variant, s Size, children ...twi.NodeOption) twi.Node {
	return m.trigger(v, s, func(k input.KeyEvent) bool {
		opens := press(k) || k.Key == input.KeyArrowDown
		if opens {
			m.show(0)
		}
		return opens
	}, func() {
		if m.Open {
			m.dismiss()
			return
		}
		m.show(0)
	}, children)
}

func (m *DropdownMenu) Content(children ...twi.NodeOption) twi.Node {
	return m.menu(m.place, menuContent, children)
}

func (m *DropdownMenu) menu(place func(phase, func() twi.Node) twi.Node, classes string, children []twi.NodeOption) twi.Node {
	m.settle()
	at := m.phase()
	return place(at, func() twi.Node {
		return m.content(classes, at, func(k input.KeyEvent) bool {
			if escape(k) {
				m.dismiss()
				return true
			}
			return m.key(k)
		}, children)
	})
}

func (m *DropdownMenu) show(item int) {
	m.active = item
	m.close(&m.menuLevel)
	m.set(true)
}

func (m *DropdownMenu) dismiss() {
	m.set(false)
	m.close(&m.menuLevel)
}

func (m *DropdownMenu) close(l *menuLevel) {
	for _, s := range m.subs {
		if s.parent == l {
			s.Open = false
			m.close(&s.menuLevel)
		}
	}
}

func (m *DropdownMenu) deepest() (*menuLevel, *DropdownMenuSub) {
	l, sub := &m.menuLevel, (*DropdownMenuSub)(nil)
	for {
		i := slices.IndexFunc(m.subs, func(s *DropdownMenuSub) bool { return s.parent == l && s.Open })
		if i < 0 {
			return l, sub
		}
		sub = m.subs[i]
		l = &sub.menuLevel
	}
}

type ContextMenu struct {
	DropdownMenu
	pointer *image.Point
}

func NewContextMenu(rt *twi.Runtime) *ContextMenu {
	c := &ContextMenu{}
	c.rt, c.root = rt, &c.DropdownMenu
	return c
}

func (c *ContextMenu) Node(children ...twi.NodeOption) twi.Node {
	return part("relative flex flex-col", append([]twi.NodeOption{twi.OnPointerDownOutside(c.dismiss)}, children...))
}

func (c *ContextMenu) Trigger(children ...twi.NodeOption) twi.Node {
	keys := c.behave(func(k input.KeyEvent) bool {
		opens := k.Key == input.KeyF10 && k.Modifiers == input.ModShift
		if opens {
			c.pointer = nil
			c.show(0)
		}
		return opens
	})
	right := twi.OnPointerDown(func(e *twi.Event) {
		if e.Mouse.Button == input.MouseRight && !c.Disabled {
			c.pointer = &image.Point{X: e.Mouse.X, Y: e.Mouse.Y}
			c.show(0)
			c.rt.Invalidate()
		}
	})
	return part("flex flex-col "+focusRing, slices.Concat(keys, []twi.NodeOption{right}, children))
}

func (c *ContextMenu) Content(children ...twi.NodeOption) twi.Node {
	return c.menu(func(at phase, content func() twi.Node) twi.Node {
		if c.pointer == nil {
			return part("absolute top-0 left-0 z-50 flex", at.holding(content))
		}
		return part("fixed z-50 flex", append([]twi.NodeOption{twi.At(c.pointer.X, c.pointer.Y)}, at.holding(content)...))
	}, menuContent, children)
}

type DropdownMenuSub struct {
	menuLevel
	presence
	Open   bool
	parent *menuLevel
}

func (l *menuLevel) Sub() *DropdownMenuSub {
	s := &DropdownMenuSub{menuLevel: menuLevel{root: l.root}, parent: l}
	l.root.subs = append(l.root.subs, s)
	return s
}

func (s *DropdownMenuSub) Node(children ...twi.NodeOption) twi.Node {
	return part("relative flex flex-col", children)
}

func (s *DropdownMenuSub) Trigger(text string, children ...twi.NodeOption) twi.Node {
	return s.parent.add(menuItem{text: text, sub: s}, "", append(children, icon("›", "pl-2 text-muted-foreground")))
}

func (s *DropdownMenuSub) Content(children ...twi.NodeOption) twi.Node {
	s.settle()
	at := s.next(s.root.rt, s.Open)
	return part("absolute left-full -top-1 ml-1 z-50 flex", at.holding(func() twi.Node {
		return s.content("flex flex-col min-w-16 rounded-md whitespace-nowrap border bg-popover px-1 text-popover-foreground shadow-lg "+popMotion, at, func(k input.KeyEvent) bool {
			if k.Key == input.KeyArrowLeft {
				s.Open = false
				s.root.close(&s.menuLevel)
				return true
			}
			return s.key(k)
		}, children)
	}))
}

func (l *menuLevel) settle() {
	l.items, l.built = l.built, nil
	l.active = min(l.active, max(len(l.items)-1, 0))
	if i := slices.IndexFunc(l.items, func(it menuItem) bool { return it.sub != nil && it.sub.Open }); i >= 0 {
		l.active = i
	}
}

func (l *menuLevel) content(classes string, at phase, keys func(input.KeyEvent) bool, children []twi.NodeOption) twi.Node {
	options := []twi.NodeOption{at.state()}
	if l.root.bar == nil {
		options = append(append(options, twi.Focusable()), at.trap(l.root.rt, keys)...)
	}
	return part(classes, append(options, children...))
}

func (l *menuLevel) opened() bool {
	return slices.ContainsFunc(l.root.subs, func(s *DropdownMenuSub) bool { return s.parent == l && s.Open })
}

func (l *menuLevel) add(it menuItem, classes string, children []twi.NodeOption) twi.Node {
	at := len(l.built)
	if it.sub != nil && it.sub.Open || at == l.active && !l.opened() {
		classes += " bg-accent text-accent-foreground"
	}
	l.built = append(l.built, it)
	m := l.root
	highlight := twi.OnPointerEnter(func() {
		if l.active == at && l.opened() == (it.sub != nil) {
			return
		}
		l.active = at
		m.close(l)
		if it.sub != nil {
			it.sub.Open, it.sub.active = true, 0
		}
		m.rt.Invalidate()
	})
	choose := m.click(func() { l.choose(it) })
	return part("relative flex flex-row items-center rounded-sm px-2 select-none [&_svg]:shrink-0 [&_svg]:pointer-events-none "+classes, append([]twi.NodeOption{highlight, choose, part("grow", []twi.NodeOption{twi.Text(it.text)})}, children...))
}

func (l *menuLevel) Item(text string, children ...twi.NodeOption) twi.Node {
	return l.add(menuItem{text: text}, "", children)
}

func (l *menuLevel) CheckboxItem(text string, checked *bool, children ...twi.NodeOption) twi.Node {
	return l.add(menuItem{text: text, checked: checked}, "pl-4", append(children, indicator(*checked, "✓")))
}

func (l *menuLevel) RadioItem(text string, value *string, children ...twi.NodeOption) twi.Node {
	return l.add(menuItem{text: text, group: value}, "pl-4", append(children, indicator(*value == text, "•")))
}

func indicator(on bool, mark string) twi.Node {
	var children []twi.NodeOption
	if on {
		children = []twi.NodeOption{icon(mark, "")}
	}
	return part("absolute left-1 flex", children)
}

func (l *menuLevel) key(k input.KeyEvent) bool {
	last := len(l.items) - 1
	switch {
	case k.Key == input.KeyArrowDown:
		l.active = max(min(l.active+1, last), 0)
	case k.Key == input.KeyArrowUp:
		l.active = max(l.active-1, 0)
	case k.Key == input.KeyHome:
		l.active = 0
	case k.Key == input.KeyEnd:
		l.active = max(last, 0)
	case k.Key == input.KeyTab || k.Key == input.KeyArrowLeft:
	case last < 0:
		return k.Key == input.KeyRune || k.Key == input.KeyEnter || k.Key == input.KeyArrowRight
	case press(k) || k.Key == input.KeyArrowRight && l.items[l.active].sub != nil:
		l.choose(l.items[l.active])
	case k.Key == input.KeyArrowRight:
	case typed(k):
		l.active = typeahead(last+1, l.active, k.Rune, func(i int) string { return l.items[i].text })
	default:
		return false
	}
	return true
}

func (l *menuLevel) choose(it menuItem) {
	m := l.root
	switch {
	case it.sub != nil:
		m.close(l)
		it.sub.Open, it.sub.active = true, 0
		return
	case it.checked != nil:
		*it.checked = !*it.checked
	case it.group != nil:
		*it.group = it.text
	}
	m.dismiss()
	notify(m.OnSelect, it.text)
}

func DropdownMenuLabel(children ...twi.NodeOption) twi.Node {
	return part("px-2 font-medium", children)
}

func DropdownMenuSeparator() twi.Node {
	return part("-mx-1 shrink-0 border-t", nil)
}

func DropdownMenuShortcut(children ...twi.NodeOption) twi.Node {
	return part("pl-4 text-muted-foreground", children)
}
