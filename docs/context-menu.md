# Context Menu

Displays a menu of actions where you right click.

<Preview name="context-menu-demo" />

## Usage

```go
menu := ui.NewContextMenu(rt)
menu.OnSelect = func(item string) { run(item) }
```

```go
menu.Node(
	menu.Trigger(twi.Class("h-5 w-40 border border-dashed"), twi.Text("Right click here")),
	menu.Content(menu.Item("Back"), menu.Item("Forward"), menu.Item("Reload")),
)
```

A right click opens the menu at the pointer; Shift+F10 opens it from the keyboard at the corner of the area. The items are the same as a [Dropdown Menu](dropdown-menu.md)'s.

## API reference

<Props of="ContextMenu" />
