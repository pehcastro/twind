# Item

A row with media, a title, a description and actions.

<Preview name="item-demo" />

## Usage

```go
ui.Item(ui.ItemOutline, ui.ItemSizeDefault,
	ui.ItemContent(
		ui.ItemTitle(twi.Text("Basic item")),
		ui.ItemDescription(twi.Text("A title and a description.")),
	),
	ui.ItemActions(ui.Button(ui.ButtonOutline, ui.ButtonSizeSM, twi.Text("Action"))),
)
```

## API reference

<Props of="Item" />
