# Drawer

A panel that slides up from the bottom edge, with a handle that drags it closed.

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

It is a [Dialog](dialog.md) that slides in from an edge and takes at most 80% of the height. Dragging the handle down moves the panel with the pointer; a release past a quarter of its height closes it, anything less puts it back. `ui.Top`, `ui.Right` and `ui.Left` slide it from the other edges.

## API reference

<Props of="Drawer" />
