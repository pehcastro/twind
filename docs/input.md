# Input

A one-line text field.

<Preview name="input-demo" />

## Usage

```go
email := ui.NewInput(rt)
email.Placeholder = "m@example.com"
```

```go
ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Email")), email.Node())
```

Read what it holds with `email.Value()`. It edits like a browser field: the arrows, Home and End move, Ctrl with them moves by word, Shift selects, Ctrl+A selects all, and Ctrl+Z and Ctrl+Y undo and redo. For several lines use a [Textarea](textarea.md); for addons inside the border, an [Input Group](input-group.md).

## API reference

<Props of="Input" />
