package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func MarkerDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col w-56 gap-1"),
			ui.Marker(ui.Default, ui.MarkerIcon(twi.Text("•")), ui.MarkerContent(twi.Text("Pedro joined the conversation"))),
			ui.Marker(ui.Ruled, ui.MarkerContent(twi.Text("Yesterday"))),
			ui.Marker(ui.Bordered, ui.MarkerContent(twi.Text("Earlier messages"))),
		)
	}
}
