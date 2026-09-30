package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func CommandDialogDemo(rt *twi.Runtime) func() twi.Node {
	palette, chosen := ui.NewCommandDialog(rt), "nothing yet"
	palette.Hotkey = 'j'
	palette.OnSelect = func(value string) { chosen = value }
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col items-center gap-1"),
			twi.Element(twi.Class("flex flex-row items-center gap-2"),
				ui.KbdGroup(ui.Kbd(twi.Text("Ctrl")), ui.Kbd(twi.Text("J"))),
				twi.Element(twi.Class("text-muted-foreground"), twi.Text("or")),
				palette.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Open the palette")),
			),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text("chosen: "+chosen)),
			palette.Node(
				palette.Input("Type a command or search..."),
				palette.List(
					palette.Group("Suggestions", palette.Item("Calendar"), palette.Item("Search Emoji"), palette.Item("Calculator")),
					palette.Group("Settings", palette.Item("Profile"), palette.Item("Billing"), palette.Item("Settings")),
				),
			),
		)
	}
}
