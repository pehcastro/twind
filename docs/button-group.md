# Button Group

A container that joins related buttons.

<Preview name="button-group-demo" />

## Usage

```go
ui.ButtonGroup(ui.Horizontal,
	ui.Button(ui.Outline, ui.SizeDefault, twi.Text("Archive")),
	ui.ButtonGroupSeparator(ui.Vertical),
	ui.Button(ui.Outline, ui.SizeDefault, twi.Text("Report")),
)
```

`ui.Vertical` stacks the buttons. A separator runs across the group, so a horizontal group takes vertical separators.

## API reference

<Props of="ButtonGroup" />
