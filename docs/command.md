# Command

A command menu for search and quick actions.

<Preview name="command-demo" />

## Usage

```go
command := ui.NewCommand(rt)
command.OnSelect = func(value string) { run(value) }
```

```go
command.Node(
	command.Input("Type a command or search..."),
	command.List(
		command.Group("Suggestions", command.Item("Calendar"), command.Item("Calculator")),
		command.Separator(),
		command.Group("Settings", command.Item("Profile", twi.Text("Profile"), ui.CommandShortcut(twi.Text("⌘P")))),
	),
)
```

The search matches letters in order, so `clc` finds Calculator, and ranks an unbroken match nearer the start first. Up and Down move, Enter or a click chooses.

## Command dialog

The same menu in a dialog, opened by Ctrl and a key. The search box at the top of this site is one. This demo listens for Ctrl+J, because Ctrl+K already opens the site's own.

<Preview name="command-dialog-demo" />

```go
palette := ui.NewCommandDialog(rt)
```

```go
palette.Node(palette.Input("Search..."), palette.List(groups...))
```

## API reference

<Props of="Command" />

<Props of="CommandDialog" />
