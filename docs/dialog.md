# Dialog

A window overlaid on the page, rendering the content underneath inert.

<Preview name="dialog-demo" />

## Usage

```go
dialog := ui.NewDialog(rt)
```

```go
dialog.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Open")),
dialog.Content(
	dialog.Header(
		dialog.Title(twi.Text("Are you sure?")),
		dialog.Description(twi.Text("This cannot be undone.")),
	),
	dialog.Footer(dialog.Close(ui.Default, ui.SizeDefault, twi.Text("Confirm"))),
),
```

Create the dialog once, next to the runtime, and call its parts from the render function. The dialog keeps its own state: `Open` says whether it is showing, and `OnOpenChange` hears every change.

## Closing

Escape, a click outside the panel, the close button in its corner and any `dialog.Close` button close it. Tab keeps focus inside the dialog while it is open. It fades and zooms in over 200 ms, and plays the same motion back when it closes.

## API reference

<Props of="Dialog" />
