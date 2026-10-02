package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func InputDemo(rt *twi.Runtime) func() twi.Node {
	email, disabled, invalid := ui.NewInput(rt), ui.NewInput(rt), ui.NewInput(rt)
	email.Key, email.Placeholder, disabled.Placeholder = "email", "m@example.com", "Disabled"
	disabled.Disabled, invalid.Invalid = true, true
	invalid.Insert("not-an-email")
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col w-48 gap-1"),
			ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Email")), email.Node()),
			ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Disabled")), disabled.Node()),
			ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Invalid")), invalid.Node()),
		)
	}
}
