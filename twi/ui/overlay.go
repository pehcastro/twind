package ui

import (
	"slices"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

type Side uint8

const (
	Bottom Side = iota
	Top
	Right
	Left
)

type Alignment uint8

const (
	Center Alignment = iota
	Start
	End
)

type overlay struct {
	control
	Open         bool
	OnOpenChange func(bool)
}

func (o *overlay) set(open bool) {
	if o.Open != open {
		o.Open = open
		notify(o.OnOpenChange, open)
		o.rt.Invalidate()
	}
}

func (o *overlay) Trigger(v Variant, s Size, children ...twi.NodeOption) twi.Node {
	return o.trigger(v, s, func(k input.KeyEvent) bool {
		if press(k) {
			o.set(true)
		}
		return press(k)
	}, children)
}

func (o *overlay) trigger(v Variant, s Size, keys func(input.KeyEvent) bool, children []twi.NodeOption) twi.Node {
	return part(button(v, s, o.ring(idleRing(v), true)), slices.Concat(o.behave(keys), children))
}

func (o *overlay) dismissable(classes string, children []twi.NodeOption) twi.Node {
	return part(classes, append([]twi.NodeOption{twi.FocusScope(), keyDown(o.rt, func(k input.KeyEvent) bool {
		if escape(k) {
			o.set(false)
		}
		return escape(k)
	})}, children...))
}

func escape(k input.KeyEvent) bool { return k.Key == input.KeyEscape }

func closed() twi.Node { return part("hidden", []twi.NodeOption{twi.Key("closed")}) }

type anchored struct {
	overlay
	Side  Side
	Align Alignment
}

func (a *anchored) Node(children ...twi.NodeOption) twi.Node {
	return part("relative flex w-fit h-fit", children)
}

func (a *anchored) place(content twi.Node) twi.Node {
	across := map[Alignment]string{Start: "justify-start", Center: "justify-center", End: "justify-end"}
	if a.Side == Right || a.Side == Left {
		across = map[Alignment]string{Start: "items-start", Center: "items-center", End: "items-end"}
	}
	return part("absolute z-50 flex flex-row "+pick("side", a.Side, map[Side]string{
		Bottom: "top-full inset-x-0",
		Top:    "bottom-full inset-x-0",
		Right:  "left-full inset-y-0 ml-1",
		Left:   "right-full inset-y-0 mr-1 justify-end",
	})+" "+pick("align", a.Align, across), []twi.NodeOption{content})
}

type Popover struct{ anchored }

func NewPopover(rt *twi.Runtime) *Popover {
	return &Popover{anchored{overlay: overlay{control: control{rt: rt}}}}
}

func (p *Popover) Content(children ...twi.NodeOption) twi.Node {
	if !p.Open {
		return closed()
	}
	return p.place(p.dismissable("flex flex-col w-36 shrink-0 rounded-md border bg-popover px-2 py-1 text-popover-foreground shadow-md", children))
}

type hint struct{ anchored }

func (h *hint) Trigger(v Variant, s Size, children ...twi.NodeOption) twi.Node {
	show := func(on bool) func() {
		return func() {
			h.focused = on
			h.set(on)
			h.rt.Invalidate()
		}
	}
	return part(button(v, s, h.ring(idleRing(v), true)), append([]twi.NodeOption{
		twi.Focusable(), twi.OnFocus(show(true)), twi.OnBlur(show(false)),
		keyDown(h.rt, func(k input.KeyEvent) bool {
			hide := escape(k) && h.Open
			if hide {
				h.set(false)
			}
			return hide
		}),
	}, children...))
}

func (h *hint) content(classes string, children []twi.NodeOption) twi.Node {
	if !h.Open {
		return closed()
	}
	return h.place(part("shrink-0 "+classes, children))
}

type Tooltip struct{ hint }

func NewTooltip(rt *twi.Runtime) *Tooltip {
	return &Tooltip{hint{anchored{overlay: overlay{control: control{rt: rt}}, Side: Top}}}
}

func (t *Tooltip) Content(children ...twi.NodeOption) twi.Node {
	return t.content("w-fit rounded-md bg-foreground px-2 text-background", children)
}

type HoverCard struct{ hint }

func NewHoverCard(rt *twi.Runtime) *HoverCard {
	return &HoverCard{hint{anchored{overlay: overlay{control: control{rt: rt}}}}}
}

func (c *HoverCard) Content(children ...twi.NodeOption) twi.Node {
	return c.content("flex flex-col w-32 rounded-md border bg-popover px-2 py-1 text-popover-foreground shadow-md", children)
}
