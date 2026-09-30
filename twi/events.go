package twi

import (
	"github.com/twind-dev/twind/twi/events"
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

func always(handler func()) listener { return listener{Handle: func(*Event) { handler() }} }

type behaviour func(*runtime.Node)

func (b behaviour) apply(n *node) { b(n.behave()) }

type onKeyDown func(*Event)

func (h onKeyDown) apply(n *node) { push(&n.behave().KeyDown, listener{Handle: h}) }

func OnKeyDown(handler func(*Event)) NodeOption { return onKeyDown(handler) }

type onFocus func()

func (h onFocus) apply(n *node) { push(&n.behave().Focus, always(h)) }

func OnFocus(handler func()) NodeOption { return onFocus(handler) }

type onBlur func()

func (h onBlur) apply(n *node) { push(&n.behave().Blur, always(h)) }

func OnBlur(handler func()) NodeOption { return onBlur(handler) }

type onClick func(*Event)

func (h onClick) apply(n *node) { push(&n.behave().Click, listener{Handle: h}) }

func OnClick(handler func(*Event)) NodeOption { return onClick(handler) }

type onPointerDown func(*Event)

func (h onPointerDown) apply(n *node) { push(&n.behave().PointerDown, listener{Handle: h}) }

func OnPointerDown(handler func(*Event)) NodeOption { return onPointerDown(handler) }

type onPointerEnter func()

func (h onPointerEnter) apply(n *node) { push(&n.behave().Enter, always(h)) }

func OnPointerEnter(handler func()) NodeOption { return onPointerEnter(handler) }

type onPointerLeave func()

func (h onPointerLeave) apply(n *node) { push(&n.behave().Leave, always(h)) }

func OnPointerLeave(handler func()) NodeOption { return onPointerLeave(handler) }

type onPointerDownOutside func()

func (h onPointerDownOutside) apply(n *node) { push(&n.behave().PointerDownOutside, (func())(h)) }

func OnPointerDownOutside(handler func()) NodeOption { return onPointerDownOutside(handler) }

type onFocusOutside func()

func (h onFocusOutside) apply(n *node) { push(&n.behave().FocusOutside, (func())(h)) }

func OnFocusOutside(handler func()) NodeOption { return onFocusOutside(handler) }

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

func Key(key string) NodeOption { return behaviour(func(n *runtime.Node) { n.Key = key }) }
