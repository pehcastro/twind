package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func ContextMenuDemo(rt *twi.Runtime) func() twi.Node {
	menu, bookmarks, person := ui.NewContextMenu(rt), true, "Pedro Duarte"
	return func() twi.Node {
		return menu.Node(
			menu.Trigger(twi.Class("h-5 w-40 items-center justify-center rounded-md border border-dashed"), twi.Text("Right click here")),
			menu.Content(
				menu.Item("Back", ui.DropdownMenuShortcut(twi.Text("⌘["))),
				menu.Item("Forward", ui.DropdownMenuShortcut(twi.Text("⌘]"))),
				menu.Item("Reload"),
				ui.DropdownMenuSeparator(),
				menu.CheckboxItem("Show bookmarks", &bookmarks),
				ui.DropdownMenuSeparator(),
				ui.DropdownMenuLabel(twi.Text("People")),
				menu.RadioItem("Pedro Duarte", &person),
				menu.RadioItem("Colm Tuite", &person),
			),
		)
	}
}
