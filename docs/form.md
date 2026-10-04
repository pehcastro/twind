# Form

Binds fields to rules, shows each field's first failing message, and submits only when every rule passes.

<Preview name="form-demo" />

## Usage

```go
form, email, terms := ui.NewForm(rt), ui.NewInput(rt), ui.NewCheckbox(rt)
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
form.OnSubmit = func(v ui.FormValues) { save(v.Text["email"], v.Checked["terms"]) }

form.Item("email", form.Label("email", twi.Text("Email")), form.Control("email"))
form.Item("terms", ui.Field(ui.Horizontal, form.Control("terms"), form.Label("terms", twi.Text("Accept the terms"))))
```

Register each control once with its key and rules (`Input`, `Textarea`, `Checkbox`, `Select`); a rule returns the message to show, or an empty string, and the first failing rule wins. A field shows its message when it loses focus and when the form is submitted. `Submit` moves focus to the first invalid field and calls `OnSubmit` only when every rule passes; Enter in a registered Input submits too. `Item`, `Label` and `Control` render a field; add `FieldDescription` as usual. `Reset` restores every field to the value it had when registered and clears every message. No reflection and no struct tags: values arrive as `FormValues.Text` and `FormValues.Checked`, keyed by the names you registered.

## API reference

<Props of="Form" />
