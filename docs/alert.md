# Alert

Displays a callout for the user's attention.

<Preview name="alert-demo" />

## Usage

```go
ui.Alert(ui.Default,
	ui.AlertTitle(twi.Text("Heads up!")),
	ui.AlertDescription(twi.Text("You can add components to your app with the CLI.")),
)
```

`ui.Destructive` colours the text for an error.

## API reference

<Props of="Alert" />
