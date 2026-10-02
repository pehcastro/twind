package ui

import (
	"image"
	"slices"

	konst "github.com/twind-dev/twind/internal/konst/ui"
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

const (
	fadeMotion = "data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:animate-in data-[state=open]:fade-in-0"
	popMotion  = fadeMotion + " data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95"
)

type phase uint8

const (
	gone phase = iota
	opened
	closing
)

func (p phase) state() twi.NodeOption {
	switch p {
	case opened:
		return twi.Data("state", "open")
	case closing:
		return twi.Data("state", "closed")
	case gone:
	}
	panic("ui: a gone overlay has no state")
}

func (p phase) trap(rt *twi.Runtime, keys func(input.KeyEvent) bool) []twi.NodeOption {
	if p == closing {
		return []twi.NodeOption{twi.Key("closing")}
	}
	return []twi.NodeOption{twi.FocusScope(), keyDown(rt, keys)}
}

func (p phase) holding(content func() twi.Node) []twi.NodeOption {
	if p == gone {
		return nil
	}
	return []twi.NodeOption{content()}
}

type presence struct {
	shown, closing bool
	closes         int
}

func (p *presence) next(rt *twi.Runtime, open bool) phase {
	switch {
	case open:
		p.shown, p.closing = true, false
		return opened
	case p.shown:
		p.shown, p.closing = false, true
		p.closes++
		this := p.closes
		rt.Dispatch(func() {
			if p.closing && p.closes == this {
				p.closing = false
				rt.Invalidate()
			}
		})
		return closing
	case p.closing:
		return closing
	}
	return gone
}

type overlay struct {
	control
	presence
	Key          string
	Open         bool
	OnOpenChange func(bool)
}

func (o *overlay) phase() phase {
	o.remember()
	return o.next(o.rt, o.Open)
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
	}, func() { o.set(!o.Open) }, children)
}

func (o *overlay) trigger(v Variant, s Size, keys func(input.KeyEvent) bool, click func(), children []twi.NodeOption) twi.Node {
	return part(button(v, s, o.ring(idleRing(v), onSelf)), slices.Concat(o.behave(keys), []twi.NodeOption{o.click(click)}, children))
}

func (o *overlay) dismissable(classes string, at phase, children []twi.NodeOption) twi.Node {
	return part(classes, slices.Concat([]twi.NodeOption{at.state()}, at.trap(o.rt, func(k input.KeyEvent) bool {
		if escape(k) {
			o.set(false)
		}
		return escape(k)
	}), children))
}

func escape(k input.KeyEvent) bool { return k.Key == input.KeyEscape }

func closed() twi.Node { return part("hidden", []twi.NodeOption{twi.Key("closed")}) }

type floating struct {
	anchor, box             *twi.Ref
	sideOffset, alignOffset int
	anchorWidth             bool
	at                      image.Point
}

func (f *floating) float(rt *twi.Runtime, from image.Rectangle, side Side, align Alignment, at phase, content func(placed []twi.NodeOption) twi.Node) twi.Node {
	positioner, holder := "fixed z-50 flex", "flex flex-col shrink-0"
	if f.anchorWidth {
		positioner, holder = "absolute z-50 flex min-w-full", holder+" min-w-full"
	}
	var children []twi.NodeOption
	if at != gone {
		f.at, side = f.spot(rt.Viewport(), from, side, align)
		if f.anchorWidth {
			f.at = f.at.Sub(f.anchor.Bounds().Min)
		}
		children = []twi.NodeOption{part(holder, []twi.NodeOption{twi.Measure(f.box), content([]twi.NodeOption{
			twi.Data("side", pick("side", side, map[Side]string{Bottom: "bottom", Top: "top", Right: "right", Left: "left"})),
			twi.Data("align", pick("align", align, map[Alignment]string{Start: "start", Center: "center", End: "end"})),
		})})}
	}
	return part(positioner, append([]twi.NodeOption{twi.At(f.at.X, f.at.Y)}, children...))
}

func (f *floating) spot(view, anchor image.Rectangle, side Side, align Alignment) (image.Point, Side) {
	size := f.box.Bounds().Size()
	room := image.Rect(view.Min.X+konst.CollisionPadX, view.Min.Y+konst.CollisionPadY, view.Max.X-konst.CollisionPadX, view.Max.Y-konst.CollisionPadY)
	gap := func(s Side) int {
		if s == Right || s == Left {
			return f.sideOffset + 1
		}
		return f.sideOffset
	}
	spare := func(s Side) int {
		switch s {
		case Bottom:
			return room.Max.Y - anchor.Max.Y - gap(s) - size.Y
		case Top:
			return anchor.Min.Y - gap(s) - size.Y - room.Min.Y
		case Right:
			return room.Max.X - anchor.Max.X - gap(s) - size.X
		case Left:
		}
		return anchor.Min.X - gap(s) - size.X - room.Min.X
	}
	if opposite := map[Side]Side{Bottom: Top, Top: Bottom, Right: Left, Left: Right}[side]; spare(side) < 0 && spare(opposite) > spare(side) {
		side = opposite
	}
	across := func(start, length, size int) int {
		return f.alignOffset + start + map[Alignment]int{Start: 0, Center: (length - size) / 2, End: length - size}[align]
	}
	var at image.Point
	switch side {
	case Bottom:
		at = image.Pt(across(anchor.Min.X, anchor.Dx(), size.X), anchor.Max.Y+gap(side))
	case Top:
		at = image.Pt(across(anchor.Min.X, anchor.Dx(), size.X), anchor.Min.Y-gap(side)-size.Y)
	case Right:
		at = image.Pt(anchor.Max.X+gap(side), across(anchor.Min.Y, anchor.Dy(), size.Y))
	case Left:
		at = image.Pt(anchor.Min.X-gap(side)-size.X, across(anchor.Min.Y, anchor.Dy(), size.Y))
	}
	return image.Pt(max(min(at.X, room.Max.X-size.X), room.Min.X), max(min(at.Y, room.Max.Y-size.Y), room.Min.Y)), side
}

type anchored struct {
	overlay
	floating
	Side  Side
	Align Alignment
}

func newAnchored(rt *twi.Runtime, side Side, align Alignment) anchored {
	return anchored{overlay: overlay{control: control{rt: rt}}, floating: floating{anchor: twi.NewRef(rt), box: twi.NewRef(rt)}, Side: side, Align: align}
}

func (a *anchored) Node(children ...twi.NodeOption) twi.Node {
	return part("relative flex w-fit h-fit", append([]twi.NodeOption{twi.Measure(a.anchor), twi.OnPointerDownOutside(func() { a.set(false) })}, children...))
}

func (a *anchored) place(at phase, content func(placed []twi.NodeOption) twi.Node) twi.Node {
	return a.float(a.rt, a.anchor.Bounds(), a.Side, a.Align, at, content)
}

type Popover struct{ anchored }

func NewPopover(rt *twi.Runtime) *Popover {
	return &Popover{newAnchored(rt, Bottom, Center)}
}

func (p *Popover) Content(children ...twi.NodeOption) twi.Node {
	at := p.phase()
	return p.place(at, func(placed []twi.NodeOption) twi.Node {
		return p.dismissable("flex flex-col w-36 shrink-0 rounded-md border bg-popover px-2 py-1 text-popover-foreground shadow-md "+popMotion, at, append(placed, children...))
	})
}

type hint struct{ anchored }

func (h *hint) Node(children ...twi.NodeOption) twi.Node {
	return h.anchored.Node(append([]twi.NodeOption{twi.OnPointerEnter(func() { h.set(true) }), twi.OnPointerLeave(func() { h.set(false) })}, children...)...)
}

func (h *hint) Trigger(v Variant, s Size, children ...twi.NodeOption) twi.Node {
	show := func(on bool) func() {
		return func() {
			h.focused = on
			h.set(on)
			h.rt.Invalidate()
		}
	}
	return part(button(v, s, h.ring(idleRing(v), onSelf)), append([]twi.NodeOption{
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
	at := h.phase()
	return h.place(at, func(placed []twi.NodeOption) twi.Node {
		return part("shrink-0 "+classes, slices.Concat([]twi.NodeOption{at.state()}, placed, children))
	})
}

type Tooltip struct{ hint }

func NewTooltip(rt *twi.Runtime) *Tooltip {
	return &Tooltip{hint{newAnchored(rt, Top, Center)}}
}

func (t *Tooltip) Content(children ...twi.NodeOption) twi.Node {
	return t.content("w-fit animate-in rounded-md bg-foreground px-2 text-background fade-in-0 zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95", children)
}

type HoverCard struct{ hint }

func NewHoverCard(rt *twi.Runtime) *HoverCard {
	return &HoverCard{hint{newAnchored(rt, Bottom, Center)}}
}

func (c *HoverCard) Content(children ...twi.NodeOption) twi.Node {
	return c.content("flex flex-col w-32 rounded-md border bg-popover px-2 py-1 text-popover-foreground shadow-md "+popMotion, children)
}
