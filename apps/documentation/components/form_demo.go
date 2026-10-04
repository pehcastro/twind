package components

import (
	"strings"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func FormDemo(rt *twi.Runtime) func() twi.Node {
	form, name, email, terms := ui.NewForm(rt), ui.NewInput(rt), ui.NewInput(rt), ui.NewCheckbox(rt)
	name.Placeholder, email.Placeholder = "shadcn", "m@example.com"
	joined := "Fill in the form and press Enter."
	form.Input("username", name, func(v string) string {
		if strings.TrimSpace(v) == "" {
			return "Username is required."
		}
		return ""
	})
	form.Input("email", email, func(v string) string {
		if !strings.Contains(v, "@") {
			return "Enter a valid email."
		}
		return ""
	})
	form.Checkbox("terms", terms, func(on bool) string {
		if !on {
			return "Accept the terms first."
		}
		return ""
	})
	form.OnSubmit = func(v ui.FormValues) { joined = "Welcome, " + v.Text["username"] + "." }
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col w-full max-w-56 gap-1"),
			form.Item("username", form.Label("username", twi.Text("Username")), form.Control("username"), ui.FieldDescription(twi.Text("This is your public display name."))),
			form.Item("email", form.Label("email", twi.Text("Email")), form.Control("email")),
			form.Item("terms", ui.Field(ui.Horizontal, form.Control("terms"), form.Label("terms", twi.Text("Accept the terms")))),
			twi.Element(twi.Class("flex flex-row items-center gap-2"),
				ui.Button(ui.Default, ui.SizeDefault, twi.Focusable(), twi.OnClick(func(*twi.Event) { form.Submit() }), twi.Text("Submit")),
				ui.Button(ui.Outline, ui.SizeDefault, twi.Focusable(), twi.OnClick(func(*twi.Event) { form.Reset() }), twi.Text("Reset")),
			),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text(joined)),
		)
	}
}
