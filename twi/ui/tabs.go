package ui

import (
	"slices"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
)

type Tabs struct {
	control
	Orientation  Orientation
	Value        string
	OnChange     func(string)
	items, built []string
}

func NewTabs(rt *twi.Runtime) *Tabs { return &Tabs{control: control{rt: rt}} }

func (t *Tabs) Node(children ...twi.NodeOption) twi.Node {
	t.items, t.built = t.built, nil
	if len(t.items) > 0 && !slices.Contains(t.items, t.Value) {
		t.Value = t.items[0]
		t.rt.Invalidate()
	}
	return part(pick("tabs", t.Orientation, map[Orientation]string{
		Horizontal: "flex flex-col gap-1",
		Vertical:   "flex flex-row gap-2",
	}), children)
}

func (t *Tabs) List(children ...twi.NodeOption) twi.Node {
	return part(fade+"flex w-fit items-center justify-center rounded-lg bg-input dark:bg-muted text-muted-foreground "+pick("tabs list", t.Orientation, map[Orientation]string{
		Horizontal: "flex-row",
		Vertical:   "flex-col h-fit",
	}), append(t.behave(t.key), children...))
}

func (t *Tabs) Trigger(value string, children ...twi.NodeOption) twi.Node {
	if t.Value == "" {
		t.Value = value
	}
	t.built = append(t.built, value)
	classes := "text-foreground/60 hover:text-foreground dark:text-muted-foreground dark:hover:text-foreground"
	if value == t.Value {
		classes = "bg-background text-foreground shadow-sm dark:bg-foreground/10 dark:text-foreground"
	}
	return part("flex flex-row flex-1 items-center gap-1 rounded-md px-2 font-medium whitespace-nowrap [&_svg]:shrink-0 [&_svg]:pointer-events-none "+classes+t.ring("", onItem)+" "+pick("tabs trigger", t.Orientation, map[Orientation]string{
		Horizontal: "justify-center",
		Vertical:   "w-full justify-start",
	}), append([]twi.NodeOption{state(value == t.Value), t.dataActive(value == t.Value), t.click(func() { t.choose(value) })}, children...))
}

func (t *Tabs) Content(value string, children ...twi.NodeOption) twi.Node {
	if value != t.Value {
		return closed()
	}
	return part("flex flex-col flex-1", append([]twi.NodeOption{state(true)}, children...))
}

func state(active bool) twi.NodeOption {
	if active {
		return twi.Data("state", "active")
	}
	return twi.Data("state", "inactive")
}

func (t *Tabs) key(k input.KeyEvent) bool {
	n := len(t.items)
	i := max(slices.Index(t.items, t.Value), 0)
	next, previous := input.KeyArrowRight, input.KeyArrowLeft
	if t.Orientation == Vertical {
		next, previous = input.KeyArrowDown, input.KeyArrowUp
	}
	switch {
	case n == 0:
		return false
	case k.Key == next:
		i = (i + 1) % n
	case k.Key == previous:
		i = (i + n - 1) % n
	case k.Key == input.KeyHome:
		i = 0
	case k.Key == input.KeyEnd:
		i = n - 1
	default:
		return false
	}
	t.choose(t.items[i])
	return true
}

func (t *Tabs) choose(value string) {
	if t.Value != value {
		t.Value = value
		notify(t.OnChange, value)
	}
}
