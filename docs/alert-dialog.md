# Alert Dialog

A modal dialog that interrupts with important content and expects a response.

<Preview name="alert-dialog-demo" />

## Usage

```go
confirm := ui.NewAlertDialog(rt)
```

```go
confirm.Trigger(ui.ButtonOutline, ui.ButtonSizeDefault, twi.Text("Delete account")),
confirm.Content(
	confirm.Header(confirm.Title(twi.Text("Are you absolutely sure?"))),
	confirm.Footer(
		confirm.Close(ui.ButtonOutline, ui.ButtonSizeDefault, twi.Text("Cancel")),
		confirm.Close(ui.ButtonDestructive, ui.ButtonSizeDefault, twi.OnClick(deleteAccount), twi.Text("Delete")),
	),
),
```

Unlike a [Dialog](dialog.md), it has no close button in its corner and a click outside does not close it: only its buttons and Escape do. Put the action in the `OnClick` of a `Close` button.

## API reference

<Props of="AlertDialog" />
