package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func DropdownMenuDemo(rt *twi.Runtime) func() twi.Node {
	menu, statusBar, panel := ui.NewDropdownMenu(rt), true, "Bottom"
	menu.Align = ui.AlignStart
	invite := menu.Sub()
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col h-16"),
			menu.Node(
				menu.Trigger(ui.ButtonOutline, ui.ButtonSizeDefault, twi.Text("Open menu ⌄")),
				menu.Content(
					ui.DropdownMenuLabel(twi.Text("My Account")),
					menu.Item("Profile", ui.DropdownMenuShortcut(twi.Text("⇧⌘P"))),
					menu.Item("Billing", ui.DropdownMenuShortcut(twi.Text("⌘B"))),
					invite.Node(
						invite.Trigger("Invite users"),
						invite.Content(invite.Item("Email"), invite.Item("Message"), ui.DropdownMenuSeparator(), invite.Item("More...")),
					),
					ui.DropdownMenuSeparator(),
					menu.CheckboxItem("Status Bar", &statusBar),
					ui.DropdownMenuSeparator(),
					ui.DropdownMenuLabel(twi.Text("Panel Position")),
					menu.RadioItem("Top", &panel),
					menu.RadioItem("Bottom", &panel),
				),
			),
		)
	}
}
