package ui

import (
	"slices"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
)

type AccordionType uint8

const (
	Single AccordionType = iota
	Multiple
)

type Accordion struct {
	control
	Type         AccordionType
	Collapsible  bool
	Value        []string
	OnChange     func([]string)
	active       int
	items, built []string
}

func NewAccordion(rt *twi.Runtime) *Accordion { return &Accordion{control: control{rt: rt}} }

func (a *Accordion) Node(children ...twi.NodeOption) twi.Node {
	if len(a.built) != len(a.items) {
		a.rt.Invalidate()
	}
	a.items, a.built = a.built, nil
	a.active = min(a.active, max(len(a.items)-1, 0))
	own := twi.OnKeyDown(func(e *twi.Event) {
		if e.Target() == e.Current() && !e.Key.Release && a.key(e.Key) {
			e.PreventDefault()
			e.StopPropagation()
			a.rt.Invalidate()
		}
	})
	return part(fade+"flex flex-col", slices.Concat(a.behave(nil), []twi.NodeOption{own}, children))
}

func (a *Accordion) Item(value string, children ...twi.NodeOption) twi.Node {
	border := "border-b"
	if len(a.items) > 0 && a.items[len(a.items)-1] == value {
		border = ""
	}
	return part("flex flex-col "+border, append([]twi.NodeOption{openState(a.open(value))}, children...))
}

func (a *Accordion) Trigger(value string, children ...twi.NodeOption) twi.Node {
	at := len(a.built)
	a.built = append(a.built, value)
	chevron := "⌄"
	if a.open(value) {
		chevron = "⌃"
	}
	return part("flex flex-row flex-1 items-start justify-between gap-4 rounded-md font-medium hover:underline"+a.ring("", onItem), slices.Concat(
		[]twi.NodeOption{openState(a.open(value)), a.dataActive(at == a.active), a.click(func() {
			a.active = at
			a.toggle(value)
		})},
		children,
		[]twi.NodeOption{icon(chevron, "shrink-0 text-muted-foreground")},
	))
}

func (a *Accordion) Content(value string, children ...twi.NodeOption) twi.Node {
	if !a.open(value) {
		return closed()
	}
	return part("flex flex-col overflow-hidden py-1", append([]twi.NodeOption{openState(true)}, children...))
}

func (a *Accordion) open(value string) bool { return slices.Contains(a.Value, value) }

func (a *Accordion) key(k input.KeyEvent) bool {
	n := len(a.items)
	switch {
	case n == 0:
		return false
	case k.Key == input.KeyArrowDown:
		a.active = (a.active + 1) % n
	case k.Key == input.KeyArrowUp:
		a.active = (a.active + n - 1) % n
	case k.Key == input.KeyHome:
		a.active = 0
	case k.Key == input.KeyEnd:
		a.active = n - 1
	case press(k):
		a.toggle(a.items[a.active])
	default:
		return false
	}
	return true
}

func (a *Accordion) toggle(value string) {
	open := a.open(value)
	switch a.Type {
	case Single:
		if !open {
			a.change([]string{value})
		} else if a.Collapsible {
			a.change(nil)
		}
	case Multiple:
		if open {
			a.change(slices.DeleteFunc(slices.Clone(a.Value), func(v string) bool { return v == value }))
		} else {
			a.change(append(slices.Clone(a.Value), value))
		}
	default:
		panic("ui: unknown accordion type")
	}
}

func (a *Accordion) change(value []string) {
	a.Value = value
	notify(a.OnChange, value)
}

func openState(open bool) twi.NodeOption {
	if open {
		return twi.Data("state", "open")
	}
	return twi.Data("state", "closed")
}

type Collapsible struct{ overlay }

func NewCollapsible(rt *twi.Runtime) *Collapsible {
	return &Collapsible{overlay{control: control{rt: rt}}}
}

func (c *Collapsible) Node(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col", append([]twi.NodeOption{openState(c.Open)}, children...))
}

func (c *Collapsible) Trigger(v Variant, s Size, children ...twi.NodeOption) twi.Node {
	toggle := func() { c.set(!c.Open) }
	return c.trigger(v, s, func(k input.KeyEvent) bool {
		if press(k) {
			toggle()
		}
		return press(k)
	}, toggle, append([]twi.NodeOption{openState(c.Open)}, children...))
}

func (c *Collapsible) AsTrigger() twi.NodeOption {
	return c.click(func() { c.set(!c.Open) })
}

func (c *Collapsible) Content(children ...twi.NodeOption) twi.Node {
	if !c.Open {
		return closed()
	}
	return part("flex flex-col", append([]twi.NodeOption{openState(true)}, children...))
}
