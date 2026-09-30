# Drawer

A panel that slides up from the bottom edge, with a handle.

<Preview name="drawer-demo" />

## Usage

```go
drawer := ui.NewDrawer(rt, ui.Bottom)
```

```go
drawer.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Open drawer")),
drawer.Content(
	drawer.Header(drawer.Title(twi.Text("Move goal"))),
	drawer.Footer(drawer.Close(ui.Default, ui.SizeDefault, twi.Text("Submit"))),
),
```

It is a [Dialog](dialog.md) that slides in from an edge and takes at most 80% of the height. `ui.Top`, `ui.Right` and `ui.Left` slide it from the other edges.

## API reference

<Props of="Drawer" />
