package ui

import (
	"slices"
	"strings"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

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
}

func NewDropdownMenu(rt *twi.Runtime) *DropdownMenu {
	m := &DropdownMenu{anchored: anchored{overlay: overlay{control: control{rt: rt}}}}
	m.root = m
	return m
}

func (m *DropdownMenu) Trigger(v Variant, s Size, children ...twi.NodeOption) twi.Node {
	open := func() {
		m.active = 0
		m.close(&m.menuLevel)
		m.set(true)
	}
	return m.trigger(v, s, func(k input.KeyEvent) bool {
		opens := press(k) || k.Key == input.KeyArrowDown
		if opens {
			open()
		}
		return opens
	}, func() {
		if m.Open {
			m.dismiss()
			return
		}
		open()
	}, children)
}

func (m *DropdownMenu) Content(children ...twi.NodeOption) twi.Node {
	m.settle()
	if !m.Open {
		return closed()
	}
	return m.place(m.content("flex flex-col min-w-16 shrink-0 rounded-md border bg-popover px-1 text-popover-foreground shadow-md", func(k input.KeyEvent) bool {
		if escape(k) {
			m.dismiss()
			return true
		}
		return m.key(k)
	}, children))
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

type DropdownMenuSub struct {
	menuLevel
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
	return s.parent.add(menuItem{text: text, sub: s}, "", append(children, part("pl-2 text-muted-foreground", []twi.NodeOption{twi.Text("›")})))
}

func (s *DropdownMenuSub) Content(children ...twi.NodeOption) twi.Node {
	s.settle()
	if !s.Open {
		return closed()
	}
	return s.content("absolute left-full -top-1 ml-1 z-50 flex flex-col min-w-16 rounded-md border bg-popover px-1 text-popover-foreground shadow-lg", func(k input.KeyEvent) bool {
		if k.Key == input.KeyArrowLeft {
			s.Open = false
			s.root.close(&s.menuLevel)
			return true
		}
		return s.key(k)
	}, children)
}

func (l *menuLevel) settle() {
	l.items, l.built = l.built, nil
	l.active = min(l.active, max(len(l.items)-1, 0))
	if i := slices.IndexFunc(l.items, func(it menuItem) bool { return it.sub != nil && it.sub.Open }); i >= 0 {
		l.active = i
	}
}

func (l *menuLevel) content(classes string, keys func(input.KeyEvent) bool, children []twi.NodeOption) twi.Node {
	return part(classes, append([]twi.NodeOption{twi.FocusScope(), twi.Focusable(), keyDown(l.root.rt, keys)}, children...))
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
	return part("relative flex flex-row items-center rounded-sm px-2 select-none "+classes, append([]twi.NodeOption{highlight, choose, part("grow", []twi.NodeOption{twi.Text(it.text)})}, children...))
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
		children = []twi.NodeOption{twi.Text(mark)}
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
	case k.Key == input.KeyRune && k.Modifiers&^input.ModShift == 0:
		for i := range last + 1 {
			if next := (l.active + 1 + i) % (last + 1); strings.HasPrefix(strings.ToLower(l.items[next].text), strings.ToLower(string(k.Rune))) {
				l.active = next
				break
			}
		}
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
