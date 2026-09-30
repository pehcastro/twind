# Item

A row with media, a title, a description and actions.

<Preview name="item-demo" />

## Usage

```go
ui.Item(ui.Outline, ui.SizeDefault,
	ui.ItemContent(
		ui.ItemTitle(twi.Text("Basic item")),
		ui.ItemDescription(twi.Text("A title and a description.")),
	),
	ui.ItemActions(ui.Button(ui.Outline, ui.SizeSM, twi.Text("Action"))),
)
```

## API reference

<Props of="Item" />
