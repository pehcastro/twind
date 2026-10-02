package twi

import (
	"image"

	"github.com/twind-dev/twind/twi/events"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/runtime"
	"github.com/twind-dev/twind/twi/style"
)

type Event = events.Event[*runtime.Elem]

type listener = events.Listener[*runtime.Elem]

func (n *node) behave() *runtime.Node {
	own := n.withHandlers()
	own.behaves = true
	return &own.events
}

func push[T any](list *[]T, item T) { *list = append(*list, item) }

func (n *node) listen(t events.Type, handler func(*Event)) {
	push(&n.behave().Listeners, listener{Type: t, Handle: handler})
}

func always(handler func()) func(*Event) { return func(*Event) { handler() } }

type behaviour func(*runtime.Node)

func (b behaviour) apply(n *node) { b(n.behave()) }

type onKeyDown func(*Event)

func (h onKeyDown) apply(n *node) { n.listen(events.KeyDown, h) }

func OnKeyDown(handler func(*Event)) NodeOption { return onKeyDown(handler) }

type onFocus func()

func (h onFocus) apply(n *node) { n.listen(events.Focus, always(h)) }

func OnFocus(handler func()) NodeOption { return onFocus(handler) }

type onBlur func()

func (h onBlur) apply(n *node) { n.listen(events.Blur, always(h)) }

func OnBlur(handler func()) NodeOption { return onBlur(handler) }

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

type onPointerEnter func()

func (h onPointerEnter) apply(n *node) { n.listen(events.PointerEnter, always(h)) }

func OnPointerEnter(handler func()) NodeOption { return onPointerEnter(handler) }

type onPointerLeave func()

func (h onPointerLeave) apply(n *node) { n.listen(events.PointerLeave, always(h)) }

func OnPointerLeave(handler func()) NodeOption { return onPointerLeave(handler) }

type onPointerDownOutside func()

func (h onPointerDownOutside) apply(n *node) { push(&n.behave().PointerDownOutside, (func())(h)) }

func OnPointerDownOutside(handler func()) NodeOption { return onPointerDownOutside(handler) }

type onFocusOutside func()

func (h onFocusOutside) apply(n *node) { push(&n.behave().FocusOutside, (func())(h)) }

func OnFocusOutside(handler func()) NodeOption { return onFocusOutside(handler) }

type onScroll func(image.Point)

func (h onScroll) apply(n *node) { push(&n.behave().Scroll, (func(image.Point))(h)) }

func OnScroll(handler func(offset image.Point)) NodeOption { return onScroll(handler) }

func OnPaste(handler func(text string)) NodeOption {
	return behaviour(func(n *runtime.Node) { n.Paste = handler })
}

func OnHotkey(handler func(input.KeyEvent) bool) NodeOption {
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

func NewRef(*Runtime) *Ref { return &Ref{} }

func Measure(ref *Ref) NodeOption { return behaviour(func(n *runtime.Node) { n.Measure = ref }) }
