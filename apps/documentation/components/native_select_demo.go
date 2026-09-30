package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func NativeSelectDemo(rt *twi.Runtime) func() twi.Node {
	status := ui.NewNativeSelect(rt)
	status.Options, status.Value = []string{"Todo", "In Progress", "Done", "Cancelled"}, "Todo"
	return func() twi.Node {
		return status.Node(twi.Class("w-30"))
	}
}
