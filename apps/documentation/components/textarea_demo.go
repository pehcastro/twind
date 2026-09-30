package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func TextareaDemo(rt *twi.Runtime) func() twi.Node {
	message := ui.NewTextarea(rt)
	message.Placeholder = "Type your message here."
	return func() twi.Node {
		return ui.Field(ui.Vertical, twi.Class("w-48"),
			ui.FieldLabel(twi.Text("Your message")),
			message.Node(),
			ui.FieldDescription(twi.Text("Your message is sent to the support team.")),
		)
	}
}
