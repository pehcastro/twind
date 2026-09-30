# Toggle Group

A row of toggles where one, or several, can be on.

<Preview name="toggle-group-demo" />

## Usage

```go
align := ui.NewToggleGroup(rt)
align.Value = []string{"left"}
```

```go
align.Node(
	align.Item("left", twi.Text("Left")),
	align.Item("center", twi.Text("Center")),
	align.Item("right", twi.Text("Right")),
)
```

One item is on at a time unless `Multiple` is set. The group is one stop in the Tab order; the arrows move between items, Space or Enter flips one.

## API reference

<Props of="ToggleGroup" />
