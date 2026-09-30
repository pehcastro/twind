# Select

A list of options to pick from, opened by a button.

<Preview name="select-demo" />

## Usage

```go
fruit := ui.NewSelect(rt)
fruit.Placeholder = "Select a fruit"
```

```go
fruit.Node(
	fruit.Trigger(twi.Class("w-30")),
	fruit.Content(
		ui.SelectLabel(twi.Text("Fruits")),
		fruit.Item("apple", "Apple"),
		fruit.Item("banana", "Banana"),
	),
)
```

Enter, Space or the arrows open it. Up and Down move, typing a letter jumps to the next item that starts with it, Enter picks and Escape closes. On the closed button, a letter picks without opening.

## API reference

<Props of="Select" />
