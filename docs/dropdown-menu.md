# Dropdown Menu

Displays a menu of actions or options, opened by a button.

<Preview name="dropdown-menu-demo" />

## Usage

```go
menu, statusBar := ui.NewDropdownMenu(rt), true
menu.OnSelect = func(item string) { run(item) }
```

```go
menu.Node(
	menu.Trigger(ui.ButtonOutline, ui.ButtonSizeDefault, twi.Text("Open")),
	menu.Content(
		ui.DropdownMenuLabel(twi.Text("My Account")),
		menu.Item("Profile", ui.DropdownMenuShortcut(twi.Text("⇧⌘P"))),
		ui.DropdownMenuSeparator(),
		menu.CheckboxItem("Status Bar", &statusBar),
	),
)
```

Enter, Space or Down on the button opens it. Up and Down move, typing a letter jumps to the next item that starts with it, Enter chooses and Escape closes. A checkbox item flips the `bool` it points to; radio items share a `string`.

## Submenus

`menu.Sub()` makes a submenu, opened by Right, Enter or the pointer, closed by Left.

```go
invite := menu.Sub()
```

```go
invite.Node(invite.Trigger("Invite users"), invite.Content(invite.Item("Email"), invite.Item("Message")))
```

## API reference

<Props of="DropdownMenu" />
