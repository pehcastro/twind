package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func SwitchDemo(rt *twi.Runtime) func() twi.Node {
	airplane := ui.NewSwitch(rt)
	return func() twi.Node {
		state := "off"
		if airplane.Checked {
			state = "on"
		}
		return twi.Element(twi.Class("flex flex-row items-center gap-2"),
			airplane.Node(),
			ui.Label(twi.Text("Airplane mode")),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text(state)),
		)
	}
}
