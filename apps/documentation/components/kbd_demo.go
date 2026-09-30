package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func KbdDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row items-center gap-2"),
			ui.KbdGroup(ui.Kbd(twi.Text("Ctrl")), ui.Kbd(twi.Text("⇧")), ui.Kbd(twi.Text("K"))),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text("then")),
			ui.KbdGroup(ui.Kbd(twi.Text("Ctrl")), twi.Text("+"), ui.Kbd(twi.Text("B"))),
		)
	}
}
