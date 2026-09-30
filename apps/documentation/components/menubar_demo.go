package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func MenubarDemo(rt *twi.Runtime) func() twi.Node {
	bar, bookmarks := ui.NewMenubar(rt), true
	file, edit, view := bar.Menu(), bar.Menu(), bar.Menu()
	shortcut := func(keys string) twi.Node { return ui.DropdownMenuShortcut(twi.Text(keys)) }
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col h-12"),
			bar.Node(
				file.Node(file.Trigger(twi.Text("File")), file.Content(
					file.Item("New Tab", shortcut("⌘T")), file.Item("New Window", shortcut("⌘N")),
					ui.DropdownMenuSeparator(), file.Item("Print...", shortcut("⌘P")),
				)),
				edit.Node(edit.Trigger(twi.Text("Edit")), edit.Content(
					edit.Item("Undo", shortcut("⌘Z")), edit.Item("Redo", shortcut("⇧⌘Z")),
					ui.DropdownMenuSeparator(), edit.Item("Cut"), edit.Item("Copy"), edit.Item("Paste"),
				)),
				view.Node(view.Trigger(twi.Text("View")), view.Content(
					view.CheckboxItem("Always show bookmarks", &bookmarks),
					ui.DropdownMenuSeparator(), view.Item("Reload"), view.Item("Full screen"),
				)),
			),
		)
	}
}
