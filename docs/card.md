# Card

Displays a card with a header, content and a footer.

<Preview name="card-demo" />

## Usage

```go
ui.Card(
	ui.CardHeader(
		ui.CardTitle(twi.Text("Card Title")),
		ui.CardDescription(twi.Text("Card Description")),
		ui.CardAction(ui.Button(ui.ButtonLink, ui.ButtonSizeXS, twi.Text("Card Action"))),
	),
	ui.CardContent(twi.Text("Card Content")),
	ui.CardFooter(twi.Text("Card Footer")),
)
```

## API reference

<Props of="Card" />
