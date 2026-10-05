package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func ResizableDemo(rt *twi.Runtime) func() twi.Node {
	panes, stack := ui.NewResizable(rt), ui.NewResizable(rt)
	stack.Orientation = ui.Vertical
	panes.WithHandle, stack.WithHandle = true, true
	center := func(s string) twi.Node {
		return twi.Element(twi.Class("flex flex-1 items-center justify-center font-semibold"), twi.Text(s))
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex h-12 w-full max-w-56 rounded-lg border"),
			panes.Node(
				panes.Panel(center("One")),
				panes.Handle(),
				panes.Panel(stack.Node(stack.Panel(center("Two")), stack.Handle(), stack.Panel(center("Three")))),
			),
		)
	}
}
