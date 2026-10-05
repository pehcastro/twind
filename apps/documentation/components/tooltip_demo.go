package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func TooltipDemo(rt *twi.Runtime) func() twi.Node {
	tip := ui.NewTooltip(rt)
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col h-5 justify-end"),
			tip.Node(
				tip.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Hover")),
				tip.Content(twi.Text("Add to library")),
			),
		)
	}
}
