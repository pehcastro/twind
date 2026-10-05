package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func AlertDialogDemo(rt *twi.Runtime) func() twi.Node {
	dialog := ui.NewAlertDialog(rt)
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row"),
			dialog.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Show dialog")),
			dialog.Content(
				dialog.Header(
					dialog.Title(twi.Text("Are you absolutely sure?")),
					dialog.Description(twi.Text("This cannot be undone. It deletes your account and its data.")),
				),
				dialog.Footer(
					dialog.Close(ui.Outline, ui.SizeDefault, twi.Text("Cancel")),
					dialog.Close(ui.Default, ui.SizeDefault, twi.Text("Continue")),
				),
			),
		)
	}
}
