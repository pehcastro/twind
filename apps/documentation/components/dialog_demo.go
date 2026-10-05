package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func DialogDemo(rt *twi.Runtime) func() twi.Node {
	dialog, name, username := ui.NewDialog(rt), ui.NewInput(rt), ui.NewInput(rt)
	dialog.Key, name.Key, username.Key = "profile", "name", "username"
	name.Insert("Pedro Duarte")
	username.Insert("@peduarte")
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row"),
			dialog.Trigger(ui.ButtonOutline, ui.ButtonSizeDefault, twi.Text("Edit profile")),
			dialog.Content(
				dialog.Header(
					dialog.Title(twi.Text("Edit profile")),
					dialog.Description(twi.Text("Make changes to your profile here. Click save when you are done.")),
				),
				ui.FieldGroup(
					ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Name")), name.Node()),
					ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Username")), username.Node()),
				),
				dialog.Footer(
					dialog.Close(ui.ButtonOutline, ui.ButtonSizeDefault, twi.Text("Cancel")),
					dialog.Close(ui.ButtonDefault, ui.ButtonSizeDefault, twi.Text("Save changes")),
				),
			),
		)
	}
}
