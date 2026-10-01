# Resizable

Panels that share a space, with handles to move the line between them.

<Preview name="resizable-demo" />

## Usage

```go
panes := ui.NewResizable(rt)
panes.Sizes = []int{40, 60}
```

```go
panes.Node(
	panes.Panel(twi.Text("One")),
	panes.Handle(true),
	panes.Panel(twi.Text("Two")),
)
```

Sizes are percentages, so the panels keep their share when the terminal is resized. Drag a handle, or Tab to it and use the arrows to step by 5%, Home and End for the smallest and largest; no panel goes under 10%. A drag keeps following the pointer outside the handle until the button is released.

## API reference

<Props of="Resizable" />
