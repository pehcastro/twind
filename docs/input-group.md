# Input Group

Adds icons, text and buttons inside an input's border.

<Preview name="input-group-demo" />

## Usage

```go
url := ui.NewInput(rt)
```

```go
url.Group(
	ui.InputGroupAddon(ui.InlineStart, ui.InputGroupText(twi.Text("https://"))),
	ui.InputGroupAddon(ui.InlineEnd, ui.InputGroupButton(twi.OnClick(copyURL), twi.Text("Copy"))),
)
```

`InlineStart` and `InlineEnd` sit before and after the text; `BlockStart` and `BlockEnd` take a row above or below it, as the message box above does. A [Textarea](textarea.md) has the same `Group`. Tab reaches the buttons after the field.

## API reference

<Props of="InputGroup" />
