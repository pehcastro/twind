# Button Group

A container that joins related buttons.

<Preview name="button-group-demo" />

## Usage

```go
ui.ButtonGroup(ui.Horizontal,
	ui.Button(ui.Outline, ui.SizeDefault, twi.Text("Archive")),
	ui.Button(ui.Outline, ui.SizeDefault, twi.Text("Report")),
)
```

`ui.Vertical` stacks the buttons. Outline buttons join without a separator. `ui.ButtonGroupSeparator` is a line between two filled buttons; it runs across the group, so a horizontal group takes vertical separators.

## API reference

<Props of="ButtonGroup" />
