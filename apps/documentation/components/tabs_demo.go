package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func TabsDemo(rt *twi.Runtime) func() twi.Node {
	tabs, name, password := ui.NewTabs(rt), ui.NewInput(rt), ui.NewInput(rt)
	name.Insert("Pedro Duarte")
	password.Placeholder = "Current password"
	panel := func(title, about string, field twi.Node) twi.Node {
		return ui.Card(
			ui.CardHeader(ui.CardTitle(twi.Text(title)), ui.CardDescription(twi.Text(about))),
			ui.CardContent(field),
			ui.CardFooter(ui.Button(ui.Default, ui.SizeDefault, twi.Text("Save"))),
		)
	}
	return func() twi.Node {
		return tabs.Node(twi.Class("w-full max-w-48"),
			tabs.List(
				tabs.Trigger("account", twi.Text("Account")),
				tabs.Trigger("password", twi.Text("Password")),
			),
			tabs.Content("account", panel("Account", "Make changes to your account here.",
				ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Name")), name.Node()))),
			tabs.Content("password", panel("Password", "Change your password here.",
				ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Current password")), password.Node()))),
		)
	}
}
