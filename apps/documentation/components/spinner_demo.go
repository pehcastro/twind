package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func SpinnerDemo(rt *twi.Runtime) func() twi.Node {
	alone, inButton, inBadge := ui.NewSpinner(rt), ui.NewSpinner(rt), ui.NewSpinner(rt)
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row items-center gap-2"),
			alone.Node(),
			ui.Button(ui.ButtonSecondary, ui.ButtonSizeSM, twi.Disabled(), inButton.Node(), twi.Text("Please wait")),
			ui.Badge(ui.BadgeOutline, inBadge.Node(), twi.Text("Syncing")),
		)
	}
}
