# Menubar

A row of menus, as at the top of a desktop app.

<Preview name="menubar-demo" />

## Usage

```go
bar := ui.NewMenubar(rt)
file, edit := bar.Menu(), bar.Menu()
```

```go
bar.Node(
	file.Node(file.Trigger(twi.Text("File")), file.Content(file.Item("New Tab"), file.Item("Print..."))),
	edit.Node(edit.Trigger(twi.Text("Edit")), edit.Content(edit.Item("Undo"), edit.Item("Redo"))),
)
```

Left and Right move between menus and carry an open menu along; once one is open, the pointer opens the menu it moves over. The items are those of a [Dropdown Menu](dropdown-menu.md).

## API reference

<Props of="Menubar" />
