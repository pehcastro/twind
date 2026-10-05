package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func AlertDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col w-full max-w-56 gap-1"),
			ui.Alert(ui.Default,
				ui.AlertTitle(twi.Text("✓ Your changes have been saved")),
				ui.AlertDescription(twi.Text("An alert with an icon, a title and a description.")),
			),
			ui.Alert(ui.Destructive,
				ui.AlertTitle(twi.Text("⊗ Unable to process your payment")),
				ui.AlertDescription(twi.Text("Check your billing information and try again.")),
			),
		)
	}
}
