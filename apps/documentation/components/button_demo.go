package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func ButtonDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row items-center gap-2"),
			ui.Button(ui.Outline, ui.SizeDefault, twi.Text("Button")),
			ui.Button(ui.Outline, ui.SizeIcon, twi.Text("↑")),
		)
	}
}
