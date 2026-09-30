# Sheet

A dialog that slides in from an edge of the screen, for content that goes with the page.

<Preview name="sheet-demo" />

## Usage

```go
sheet := ui.NewSheet(rt, ui.Right)
```

```go
sheet.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Open")),
sheet.Content(
	sheet.Header(sheet.Title(twi.Text("Edit profile"))),
	sheet.Footer(sheet.Close(ui.Default, ui.SizeDefault, twi.Text("Save changes"))),
),
```

It slides in over 500 ms and out over 300 ms. It closes like a [Dialog](dialog.md): Escape, a click outside, the close button in its corner or any `Close` button.

## API reference

<Props of="Sheet" />
