package twi

import (
	"image"

	"github.com/pehcastro/twind/internal/events"
	"github.com/pehcastro/twind/internal/runtime"
	"github.com/pehcastro/twind/twi/style"
)

type Event = events.Event[*runtime.Elem]

type listener = events.Listener[*runtime.Elem]

func (n *node) behave() *runtime.Node {
	own := n.withHandlers()
	if own.events == nil {
		own.events = &runtime.Node{}
	}
	return own.events
}

func push[T any](list *[]T, item T) { *list = append(*list, item) }

type behaviour func(*runtime.Node)

func (b behaviour) apply(n *node) { b(n.behave()) }

func (n *node) listen(t events.Type, handler func(*Event)) {
	push(&n.behave().Listeners, listener{Type: t, Handle: handler})
}

type onKeyDown func(*Event)

func (h onKeyDown) apply(n *node) { n.listen(events.KeyDown, h) }

func OnKeyDown(handler func(*Event)) NodeOption { return onKeyDown(handler) }

type onFocus func(*Event)

func (h onFocus) apply(n *node) { n.listen(events.Focus, h) }

func OnFocus(handler func(*Event)) NodeOption { return onFocus(handler) }

type onBlur func(*Event)

func (h onBlur) apply(n *node) { n.listen(events.Blur, h) }

func OnBlur(handler func(*Event)) NodeOption { return onBlur(handler) }

type onClick func(*Event)

func (h onClick) apply(n *node) { n.listen(events.Click, h) }

func OnClick(handler func(*Event)) NodeOption { return onClick(handler) }

type onPointerDown func(*Event)

func (h onPointerDown) apply(n *node) { n.listen(events.PointerDown, h) }

func OnPointerDown(handler func(*Event)) NodeOption { return onPointerDown(handler) }

type onPointerMove func(*Event)

func (h onPointerMove) apply(n *node) { n.listen(events.PointerMove, h) }

func OnPointerMove(handler func(*Event)) NodeOption { return onPointerMove(handler) }

type onPointerUp func(*Event)

func (h onPointerUp) apply(n *node) { n.listen(events.PointerUp, h) }

func OnPointerUp(handler func(*Event)) NodeOption { return onPointerUp(handler) }

type onPointerEnter func(*Event)

func (h onPointerEnter) apply(n *node) { n.listen(events.PointerEnter, h) }

func OnPointerEnter(handler func(*Event)) NodeOption { return onPointerEnter(handler) }

type onPointerLeave func(*Event)

func (h onPointerLeave) apply(n *node) { n.listen(events.PointerLeave, h) }

func OnPointerLeave(handler func(*Event)) NodeOption { return onPointerLeave(handler) }

type onPointerDownOutside func(*Event)

func (h onPointerDownOutside) apply(n *node) {
	push(&n.behave().PointerDownOutside, (func(*Event))(h))
}

func OnPointerDownOutside(handler func(*Event)) NodeOption { return onPointerDownOutside(handler) }

type onFocusOutside func(*Event)

func (h onFocusOutside) apply(n *node) { push(&n.behave().FocusOutside, (func(*Event))(h)) }

func OnFocusOutside(handler func(*Event)) NodeOption { return onFocusOutside(handler) }

type onScroll func(image.Point)

func (h onScroll) apply(n *node) { push(&n.behave().Scroll, (func(image.Point))(h)) }

func OnScroll(handler func(offset image.Point)) NodeOption { return onScroll(handler) }

func OnPaste(handler func(text string)) NodeOption {
	return behaviour(func(n *runtime.Node) { n.Paste = handler })
}

func OnHotkey(handler func(*Event)) NodeOption {
	return behaviour(func(n *runtime.Node) { n.Hotkey = handler })
}

func OnWidth(handler func(contentWidth int)) NodeOption {
	return behaviour(func(n *runtime.Node) { n.Width = handler })
}

func Focusable() NodeOption { return behaviour(func(n *runtime.Node) { n.Focusable = true }) }

func AutoFocus() NodeOption { return behaviour(func(n *runtime.Node) { n.AutoFocus = true }) }

type topLayer struct{}

func (topLayer) apply(n *node) {
	n.behave().TopLayer = true
	n.tree.TopLayer = 1
}

func TopLayer() NodeOption { return topLayer{} }

type disabled struct{}

func (disabled) apply(n *node) {
	n.behave().Disabled = true
	n.state().States |= style.StateDisabled
}

func Disabled() NodeOption { return disabled{} }

func FocusScope() NodeOption {
	return behaviour(func(n *runtime.Node) { n.Scope = runtime.ModalScope })
}

func NonModalFocusScope() NodeOption {
	return behaviour(func(n *runtime.Node) { n.Scope = runtime.NonModalScope })
}

type keyed string

func (k keyed) apply(n *node) {
	n.tree.Key = string(k)
	n.behave().Key = string(k)
}

func Key(key string) NodeOption { return keyed(key) }

type Ref = runtime.Ref

func Measure(ref *Ref) NodeOption { return behaviour(func(n *runtime.Node) { n.Measure = ref }) }
