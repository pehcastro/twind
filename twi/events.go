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

func Focusable() NodeOption { return behaviour(func(n *runtime.Node) { n.Focusable = true }) }

func AutoFocus() NodeOption { return behaviour(func(n *runtime.Node) { n.AutoFocus = true }) }

type disabled struct{}

func (disabled) apply(n *Node) {
	behaviour(func(n *runtime.Node) { n.Disabled = true }).apply(n)
	n.state().States |= style.StateDisabled
}

func Disabled() NodeOption { return disabled{} }

func FocusScope() NodeOption { return behaviour(func(n *runtime.Node) { n.Scope = true }) }

func Key(key string) NodeOption { return behaviour(func(n *runtime.Node) { n.Key = key }) }
