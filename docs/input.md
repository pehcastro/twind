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

Read what it holds with `email.Value()`. It edits with the same keys as a [Textarea](textarea.md), on one line that scrolls sideways: Ctrl+A and Ctrl+E go to the start and end, Ctrl+Left and Ctrl+Right move by word, Ctrl+W and Ctrl+U delete back, Ctrl+K deletes to the end, a paste is one undo step, Ctrl+G selects all, Shift selects, and Ctrl+Z and Ctrl+Y undo and redo. With `Submit` set, Enter sends the trimmed text and clears the field, and Up and Down walk what was sent; without it, Enter, Up and Down go on to the page. For addons inside the border, an [Input Group](input-group.md).

## API reference

<Props of="Input" />
