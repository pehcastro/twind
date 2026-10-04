package ui

import (
	"slices"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

type selectItem struct{ value, label string }

type Select struct {
	anchored
	Value, Placeholder string
	OnChange           func(string)
	active             int
	shown              string
	items, built       []selectItem
}

func NewSelect(rt *twi.Runtime) *Select {
	s := &Select{anchored: newAnchored(rt, Bottom, Start)}
	s.anchorWidth = true
	return s
}

func (s *Select) Trigger(children ...twi.NodeOption) twi.Node {
	s.shown = s.label()
	value := part("", []twi.NodeOption{twi.Text(s.shown)})
	if s.Value == "" {
		value = part("text-muted-foreground", []twi.NodeOption{twi.Text(s.Placeholder)})
	}
	keys := s.behave(func(k input.KeyEvent) bool {
		switch {
		case press(k) || k.Key == input.KeyArrowDown || k.Key == input.KeyArrowUp:
			s.open()
		case typed(k) && len(s.items) > 0:
			s.change(s.items[typeahead(len(s.items), s.index(), k.Rune, s.itemLabel)].value)
		default:
			return false
		}
		return true
	})
	return part(fade+"flex flex-row h-1 items-center justify-between gap-2 rounded-md px-1 whitespace-nowrap dark:bg-input/30 dark:hover:bg-input/50 [&_svg]:text-muted-foreground [&_svg]:shrink-0 [&_svg]:pointer-events-none "+s.ring(inputRing, onSelf),
		slices.Concat(keys, []twi.NodeOption{s.toggle(s.open), value, icon("⌄", "opacity-50")}, children))
}

func (s *Select) Content(children ...twi.NodeOption) twi.Node {
	s.items, s.built = s.built, nil
	s.active = min(s.active, max(len(s.items)-1, 0))
	if s.label() != s.shown {
		s.rt.Invalidate()
	}
	return s.list(func(k input.KeyEvent) bool {
		return s.listKey(k, &s.active, len(s.items), s.itemLabel, func(i int) { s.choose(s.items[i].value) })
	}, children)
}

func (s *Select) itemLabel(i int) string { return s.items[i].label }

func (a *anchored) list(keys func(input.KeyEvent) bool, children []twi.NodeOption) twi.Node {
	at := a.phase()
	return a.place(at, func(placed []twi.NodeOption) twi.Node {
		return part("flex flex-col shrink-0 rounded-md border bg-popover text-popover-foreground shadow-md "+popMotion,
			slices.Concat([]twi.NodeOption{at.state(), twi.Focusable()}, at.trap(a.rt, keys), placed, children))
	})
}

func (a *anchored) listKey(k input.KeyEvent, active *int, n int, label func(int) string, choose func(int)) bool {
	last := n - 1
	switch {
	case escape(k):
		a.set(false)
	case k.Key == input.KeyTab:
	case last < 0:
		return press(k) || typed(k) || arrow(k) != 0
	case k.Key == input.KeyArrowDown:
		*active = min(*active+1, last)
	case k.Key == input.KeyArrowUp:
		*active = max(*active-1, 0)
	case k.Key == input.KeyHome:
		*active = 0
	case k.Key == input.KeyEnd:
		*active = last
	case press(k):
		choose(*active)
	case typed(k):
		*active = typeahead(n, *active, k.Rune, label)
	default:
		return false
	}
	return true
}

func (s *Select) Item(value, label string, children ...twi.NodeOption) twi.Node {
	s.built = append(s.built, selectItem{value, label})
	return s.option(label, len(s.built)-1, &s.active, value == s.Value, func() { s.choose(value) }, children)
}

func (c *control) option(label string, at int, active *int, checked bool, choose func(), children []twi.NodeOption) twi.Node {
	classes := "relative flex flex-row items-center gap-1 rounded-sm pr-3 pl-1 select-none"
	if at == *active {
		classes += " bg-accent text-accent-foreground"
	}
	var mark []twi.NodeOption
	if checked {
		mark = []twi.NodeOption{icon("✓", "")}
	}
	highlight := twi.OnPointerEnter(func() {
		if *active != at {
			*active = at
			c.rt.Invalidate()
		}
	})
	return part(classes, slotted([]twi.NodeOption{highlight, c.click(choose), part("absolute right-1 flex", mark)}, children, part("grow", []twi.NodeOption{twi.Text(label)})))
}

func SelectLabel(children ...twi.NodeOption) twi.Node {
	return part("px-1 text-muted-foreground", children)
}

func SelectSeparator() twi.Node {
	return DropdownMenuSeparator()
}

func (s *Select) open() {
	s.active = max(s.index(), 0)
	s.set(true)
}

func (s *Select) choose(value string) {
	s.change(value)
	s.set(false)
}

func (s *Select) change(value string) {
	if s.Value != value {
		s.Value = value
		notify(s.OnChange, value)
	}
}

func (s *Select) index() int {
	return slices.IndexFunc(s.items, func(it selectItem) bool { return it.value == s.Value })
}

func (s *Select) label() string {
	if i := s.index(); i >= 0 {
		return s.items[i].label
	}
	return s.Value
}
