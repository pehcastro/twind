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
	return &Select{anchored: anchored{overlay: overlay{control: control{rt: rt}}, Align: Start}}
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
			s.change(s.items[typeahead(len(s.items), s.index(), k.Rune, func(i int) string { return s.items[i].label })].value)
		default:
			return false
		}
		return true
	})
	toggle := s.click(func() {
		if s.Open {
			s.set(false)
			return
		}
		s.open()
	})
	return part(fade+"flex flex-row h-1 items-center justify-between gap-2 rounded-md px-1 whitespace-nowrap dark:bg-input/30 dark:hover:bg-input/50 [&_svg]:text-muted-foreground [&_svg]:shrink-0 [&_svg]:pointer-events-none "+s.ring(inputRing, onSelf),
		slices.Concat(keys, []twi.NodeOption{toggle, value, icon("⌄", "opacity-50")}, children))
}

func (s *Select) Content(children ...twi.NodeOption) twi.Node {
	s.items, s.built = s.built, nil
	s.active = min(s.active, max(len(s.items)-1, 0))
	if s.label() != s.shown {
		s.rt.Invalidate()
	}
	if !s.Open {
		return closed()
	}
	return s.place(part("flex flex-col min-w-full shrink-0 rounded-md border bg-popover px-1 text-popover-foreground shadow-md", append([]twi.NodeOption{
		twi.FocusScope(), twi.Focusable(), keyDown(s.rt, s.key),
	}, children...)))
}

func (s *Select) Item(value, label string, children ...twi.NodeOption) twi.Node {
	at := len(s.built)
	s.built = append(s.built, selectItem{value, label})
	classes := "relative flex flex-row items-center rounded-sm pr-4 pl-1 select-none"
	if at == s.active {
		classes += " bg-accent text-accent-foreground"
	}
	var mark []twi.NodeOption
	if value == s.Value {
		mark = []twi.NodeOption{icon("✓", "")}
	}
	highlight := twi.OnPointerEnter(func() {
		if s.active != at {
			s.active = at
			s.rt.Invalidate()
		}
	})
	return part(classes, append([]twi.NodeOption{highlight, s.click(func() { s.choose(value) }), part("grow", []twi.NodeOption{twi.Text(label)}), part("absolute right-1 flex", mark)}, children...))
}

func SelectLabel(children ...twi.NodeOption) twi.Node {
	return part("px-1 text-muted-foreground", children)
}

func SelectSeparator() twi.Node {
	return DropdownMenuSeparator()
}

func (s *Select) key(k input.KeyEvent) bool {
	last := len(s.items) - 1
	switch {
	case escape(k):
		s.set(false)
	case k.Key == input.KeyTab:
	case last < 0:
		return press(k) || typed(k) || arrow(k) != 0
	case k.Key == input.KeyArrowDown:
		s.active = min(s.active+1, last)
	case k.Key == input.KeyArrowUp:
		s.active = max(s.active-1, 0)
	case k.Key == input.KeyHome:
		s.active = 0
	case k.Key == input.KeyEnd:
		s.active = last
	case press(k):
		s.choose(s.items[s.active].value)
	case typed(k):
		s.active = typeahead(last+1, s.active, k.Rune, func(i int) string { return s.items[i].label })
	default:
		return false
	}
	return true
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
