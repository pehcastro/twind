package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func ResizableDemo(rt *twi.Runtime) func() twi.Node {
	panes, stack := ui.NewResizable(rt), ui.NewResizable(rt)
	stack.Orientation = ui.Vertical
	center := func(s string) twi.Node {
		return twi.Element(twi.Class("flex flex-1 items-center justify-center font-semibold"), twi.Text(s))
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex h-12 w-full max-w-56 rounded-lg border"),
			panes.Node(
				panes.Panel(center("One")),
				panes.Handle(true),
				panes.Panel(stack.Node(stack.Panel(center("Two")), stack.Handle(true), stack.Panel(center("Three")))),
			),
		)
	}
}
