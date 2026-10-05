package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func BadgeDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row flex-wrap items-center gap-2"),
			ui.Badge(ui.BadgeDefault, twi.Text("Badge")),
			ui.Badge(ui.BadgeSecondary, twi.Text("Secondary")),
			ui.Badge(ui.BadgeDestructive, twi.Text("Destructive")),
			ui.Badge(ui.BadgeOutline, twi.Text("Outline")),
			ui.Badge(ui.BadgeSecondary, twi.Class("bg-blue-500 text-white dark:bg-blue-600"), twi.Text("✓ Verified")),
		)
	}
}
