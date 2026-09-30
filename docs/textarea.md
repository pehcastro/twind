# Textarea

A text field of several lines.

<Preview name="textarea-demo" />

## Usage

```go
message := ui.NewTextarea(rt)
message.Placeholder = "Type your message here."
```

```go
ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Your message")), message.Node())
```

Enter breaks a line. It grows with its text from a minimum of four rows; `message.Value()` returns the text.

## API reference

<Props of="Textarea" />
