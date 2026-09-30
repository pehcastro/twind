package twi

import (
	"github.com/twind-dev/twind/twi/events"
	"github.com/twind-dev/twind/twi/runtime"
	"github.com/twind-dev/twind/twi/style"
)

type Event = events.Event[*runtime.Elem]

type behaviour func(*runtime.Node)

func (b behaviour) apply(n *Node) {
	b(&n.events)
	n.behaves = true
}

func OnKeyDown(handler func(*Event)) NodeOption {
	return behaviour(func(n *runtime.Node) {
		n.KeyDown = append(n.KeyDown, events.Listener[*runtime.Elem]{Handle: handler})
	})
}

func OnFocus(handler func()) NodeOption {
	return behaviour(func(n *runtime.Node) {
		n.Focus = append(n.Focus, events.Listener[*runtime.Elem]{Handle: func(*Event) { handler() }})
	})
}

func OnBlur(handler func()) NodeOption {
	return behaviour(func(n *runtime.Node) {
		n.Blur = append(n.Blur, events.Listener[*runtime.Elem]{Handle: func(*Event) { handler() }})
	})
}

func OnClick(handler func(*Event)) NodeOption {
	return behaviour(func(n *runtime.Node) {
		n.Click = append(n.Click, events.Listener[*runtime.Elem]{Handle: handler})
	})
}

func OnPointerEnter(handler func()) NodeOption {
	return behaviour(func(n *runtime.Node) {
		n.Enter = append(n.Enter, events.Listener[*runtime.Elem]{Handle: func(*Event) { handler() }})
	})
}

func OnPointerLeave(handler func()) NodeOption {
	return behaviour(func(n *runtime.Node) {
		n.Leave = append(n.Leave, events.Listener[*runtime.Elem]{Handle: func(*Event) { handler() }})
	})
}

func OnPointerDownOutside(handler func()) NodeOption {
	return behaviour(func(n *runtime.Node) { n.PointerDownOutside = append(n.PointerDownOutside, handler) })
}

func Focusable() NodeOption { return behaviour(func(n *runtime.Node) { n.Focusable = true }) }

func AutoFocus() NodeOption { return behaviour(func(n *runtime.Node) { n.AutoFocus = true }) }

type disabled struct{}

func (disabled) apply(n *Node) {
	behaviour(func(n *runtime.Node) { n.Disabled = true }).apply(n)
	n.state().States |= style.StateDisabled
}

func Disabled() NodeOption { return disabled{} }

func FocusScope() NodeOption {
	return behaviour(func(n *runtime.Node) { n.Scope = runtime.ModalScope })
}

func NonModalFocusScope() NodeOption {
	return behaviour(func(n *runtime.Node) { n.Scope = runtime.NonModalScope })
}

func OnFocusOutside(handler func()) NodeOption {
	return behaviour(func(n *runtime.Node) { n.FocusOutside = append(n.FocusOutside, handler) })
}

func Key(key string) NodeOption { return behaviour(func(n *runtime.Node) { n.Key = key }) }
