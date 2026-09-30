# Label

The name of a control.

<Preview name="label-demo" />

## Usage

```go
ui.Label(twi.OnClick(flipTerms), twi.Text("Accept terms and conditions"))
```

A label does nothing on its own. Give it a `twi.OnClick` that acts on its control, as the demo does to flip the checkbox, or use a [Field](field.md).

## API reference

<Props of="Label" />
