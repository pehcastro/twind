package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func CommandDemo(rt *twi.Runtime) func() twi.Node {
	command, chosen := ui.NewCommand(rt), "nothing yet"
	command.OnSelect = func(value string) { chosen = value }
	shortcut := func(name, keys string) ui.CommandItem {
		return command.Item(name, twi.Text(name), ui.CommandShortcut(twi.Text(keys)))
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col items-center gap-1"),
			command.Node(twi.Class("w-48 rounded-lg border shadow-md"),
				command.Input("Type a command or search..."),
				command.List(
					command.Group("Suggestions", command.Item("Calendar"), command.Item("Search Emoji"), command.Item("Calculator")),
					command.Separator(),
					command.Group("Settings", shortcut("Profile", "⌘P"), shortcut("Billing", "⌘B"), shortcut("Settings", "⌘S")),
				),
			),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text("chosen: "+chosen)),
		)
	}
}
