package blocks

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func LoginForm(rt *twi.Runtime) func() twi.Node {
	email, password, remember := ui.NewInput(rt), ui.NewInput(rt), ui.NewCheckbox(rt)
	email.Placeholder, password.Placeholder = "m@example.com", "Password"
	status := ""
	login := twi.OnClick(func(*twi.Event) {
		status = "Signed in as " + email.Value()
		if email.Value() == "" {
			status = "Enter your email first"
		}
		rt.Invalidate()
	})
	return func() twi.Node {
		return ui.Card(twi.Class("w-56 gap-2 py-1"),
			ui.CardHeader(
				ui.CardTitle(twi.Text("Login to your account")),
				ui.CardDescription(twi.Text("Enter your email below to login")),
				ui.CardAction(ui.Button(ui.Link, ui.SizeXS, twi.Text("Sign up"))),
			),
			ui.CardContent(ui.FieldGroup(
				ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Email")), email.Node()),
				ui.Field(ui.Vertical,
					twi.Element(twi.Class("flex flex-row justify-between"),
						ui.FieldLabel(twi.Text("Password")),
						ui.Button(ui.Link, ui.SizeXS, twi.Text("Forgot it?")),
					),
					password.Node(),
				),
				ui.Field(ui.Horizontal, remember.Node(), ui.FieldLabel(twi.Text("Remember me"))),
			)),
			ui.CardFooter(twi.Class("flex-col gap-1"),
				ui.Button(ui.Default, ui.SizeDefault, twi.Class("w-full"), login, twi.Text("Login")),
				ui.Button(ui.Outline, ui.SizeDefault, twi.Class("w-full"), twi.Text("Login with Google")),
				twi.Element(twi.Class("text-muted-foreground"), twi.Text(status)),
			),
		)
	}
}
