package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func CardDemo(rt *twi.Runtime) func() twi.Node {
	email := ui.NewInput(rt)
	email.Placeholder = "m@example.com"
	return func() twi.Node {
		return ui.Card(twi.Class("w-56 gap-2 py-1"),
			ui.CardHeader(
				ui.CardTitle(twi.Text("Login to your account")),
				ui.CardDescription(twi.Text("Enter your email below to login")),
				ui.CardAction(ui.Button(ui.Link, ui.SizeXS, twi.Text("Sign up"))),
			),
			ui.CardContent(ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Email")), email.Node())),
			ui.CardFooter(twi.Class("gap-2"),
				ui.Button(ui.Default, ui.SizeDefault, twi.Text("Login")),
				ui.Button(ui.Outline, ui.SizeDefault, twi.Text("Login with Google")),
			),
		)
	}
}
