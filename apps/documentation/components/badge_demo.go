package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func BadgeDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row flex-wrap items-center gap-2"),
			ui.Badge(ui.Default, twi.Text("Badge")),
			ui.Badge(ui.Secondary, twi.Text("Secondary")),
			ui.Badge(ui.Destructive, twi.Text("Destructive")),
			ui.Badge(ui.Outline, twi.Text("Outline")),
			ui.Badge(ui.Secondary, twi.Class("bg-blue-500 text-white dark:bg-blue-600"), twi.Text("✓ Verified")),
		)
	}
}
