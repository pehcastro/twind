# Checkbox

A control that toggles between checked and not checked.

<Preview name="checkbox-demo" />

## Usage

```go
terms := ui.NewCheckbox(rt)
terms.OnChange = func(checked bool) { accepted = checked }
```

```go
ui.Field(ui.Horizontal, terms.Node(), ui.FieldLabel(twi.Text("Accept terms and conditions")))
```

A click or Space flips it. `Invalid` draws a destructive ring, and `Disabled` greys it out.

## API reference

<Props of="Checkbox" />
