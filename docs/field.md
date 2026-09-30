# Field

Combines a label, a control and help text into a form field, and fields into groups.

<Preview name="field-demo" />

## Usage

```go
ui.FieldSet(
	ui.FieldLegend(twi.Text("Payment method")),
	ui.FieldGroup(
		ui.Field(ui.Vertical,
			ui.FieldLabel(twi.Text("Name on card")),
			name.Node(),
			ui.FieldDescription(twi.Text("As it is printed on the card")),
		),
		ui.Field(ui.Horizontal, same.Node(), ui.FieldLabel(twi.Text("Same as shipping"))),
	),
)
```

`ui.Vertical` puts the label above the control, `ui.Horizontal` beside it. `FieldError` shows under a control whose `Invalid` is set.

## API reference

<Props of="Field" />
