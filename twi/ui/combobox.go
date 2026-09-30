package ui

import (
	"slices"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

type comboItem struct {
	selectItem
	shown bool
}

type Combobox struct {
	anchored
	Value, Placeholder, Empty string
	OnChange                  func(string)
	field                     *Input
	typed, filter, filled     string
	active                    int
	items, built              []comboItem
}

func NewCombobox(rt *twi.Runtime) *Combobox {
	return &Combobox{anchored: anchored{overlay: overlay{control: control{rt: rt}}, Align: Start}, Empty: "No items found.", field: NewInput(rt)}
}

func (c *Combobox) Input(options ...twi.NodeOption) twi.Node {
	if typed := c.field.Value(); typed != c.typed {
		c.typed, c.filter, c.active = typed, typed, 0
		if c.field.focused {
			c.set(true)
		}
	}
	c.field.Placeholder = c.Placeholder
	chevron := part("flex rounded-sm px-1 text-muted-foreground hover:bg-accent hover:text-accent-foreground", []twi.NodeOption{
		c.click(func() {
			if c.Open {
				c.set(false)
				return
			}
			c.show()
		}),
		icon("⌄", ""),
	})
	field := c.field.field("grow", []twi.NodeOption{twi.OnBlur(func() { c.set(false) })})
	return group(&c.field.control, field, []Addon{InputGroupAddon(InlineEnd, chevron)}, append(options, keyDown(c.rt, c.key))...)
}

func (c *Combobox) Content(children ...twi.NodeOption) twi.Node {
	c.items, c.built = c.built, nil
	c.active = min(c.active, max(shown(c.items)-1, 0))
	if c.Value != c.filled {
		c.fill()
		c.rt.Invalidate()
	}
	at := c.phase()
	return c.place(at, func() twi.Node {
		if shown(c.items) == 0 {
			children = append(children, part("py-1 text-center text-muted-foreground", []twi.NodeOption{twi.Text(c.Empty)}))
		}
		return part("flex flex-col min-w-full shrink-0 gap-1 mt-1 rounded-md border bg-popover px-1 text-popover-foreground shadow-md "+popMotion, append([]twi.NodeOption{at.state()}, children...))
	})
}

func (c *Combobox) Item(value, label string, children ...twi.NodeOption) twi.Node {
	at := shown(c.built)
	c.built = append(c.built, comboItem{selectItem{value, label}, score(label, c.filter) > 0})
	if !c.built[len(c.built)-1].shown {
		return closed()
	}
	return c.option(label, at, &c.active, value == c.Value, func() { c.choose(selectItem{value, label}) }, children)
}

func shown(items []comboItem) int {
	n := 0
	for _, it := range items {
		if it.shown {
			n++
		}
	}
	return n
}

func (c *Combobox) chosen() int {
	return slices.IndexFunc(c.items, func(it comboItem) bool { return it.value == c.Value })
}

func (c *Combobox) show() {
	c.filter, c.active = "", max(c.chosen(), 0)
	c.set(true)
}

func (c *Combobox) fill() {
	c.filled, c.filter = c.Value, ""
	label := ""
	if i := c.chosen(); i >= 0 {
		label = c.items[i].label
	}
	c.field.Apply(input.KeyEvent{Key: input.KeyRune, Rune: 'a', Modifiers: input.ModCtrl})
	c.field.Insert(label)
	c.typed = c.field.Value()
}

func (c *Combobox) key(k input.KeyEvent) bool {
	switch {
	case k.Key == input.KeyTab:
		c.set(false)
		return false
	case !c.Open && (k.Key == input.KeyArrowDown || k.Key == input.KeyArrowUp):
		c.show()
	case !c.Open:
		return false
	case escape(k):
		c.set(false)
		c.fill()
	case k.Key == input.KeyArrowDown:
		c.active = min(c.active+1, max(shown(c.items)-1, 0))
	case k.Key == input.KeyArrowUp:
		c.active = max(c.active-1, 0)
	case k.Key == input.KeyEnter && k.Modifiers == 0:
		if shown(c.items) > 0 {
			c.choose(c.visible(c.active))
		}
	default:
		return false
	}
	return true
}

func (c *Combobox) visible(n int) selectItem {
	for _, it := range c.items {
		if it.shown {
			if n == 0 {
				return it.selectItem
			}
			n--
		}
	}
	panic("ui: combobox has no visible item there")
}

func (c *Combobox) choose(it selectItem) {
	if c.Value != it.value {
		c.Value = it.value
		notify(c.OnChange, it.value)
	}
	c.fill()
	c.set(false)
}
