package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func RadioGroupDemo(rt *twi.Runtime) func() twi.Node {
	density := ui.NewRadioGroup(rt)
	density.Value = "comfortable"
	return func() twi.Node {
		return density.Node(
			density.Item("default", ui.Label(twi.Text("Default"))),
			density.Item("comfortable", ui.Label(twi.Text("Comfortable"))),
			density.Item("compact", ui.Label(twi.Text("Compact"))),
		)
	}
}
