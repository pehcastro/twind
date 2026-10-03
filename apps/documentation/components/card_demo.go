package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func CardDemo(rt *twi.Runtime) func() twi.Node {
	email := ui.NewInput(rt)
	email.Placeholder = "m@example.com"
	return func() twi.Node {
		return ui.Card(twi.Class("w-full max-w-56 gap-0.5 py-0.5 border-[0.5px]"),
			ui.CardHeader(
				ui.CardTitle(twi.Text("Login to your account")),
				ui.CardDescription(twi.Text("Enter your email below to login")),
				ui.CardAction(ui.Button(ui.Link, ui.SizeXS, twi.Text("Sign up"))),
			),
			ui.CardContent(ui.Field(ui.Vertical, twi.Class("gap-0.5"), ui.FieldLabel(twi.Text("Email")), email.Node())),
			ui.CardFooter(twi.Class("flex-col gap-0.5"),
				ui.Button(ui.Default, ui.SizeDefault, twi.Class("w-full py-0.5"), twi.Text("Login")),
				ui.Button(ui.Outline, ui.SizeDefault, twi.Class("w-full border-[0.5px] shadow-none"), twi.Text("Login with Google")),
			),
		)
	}
}
