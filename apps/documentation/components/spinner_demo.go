package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func SpinnerDemo(rt *twi.Runtime) func() twi.Node {
	alone, inButton, inBadge := ui.NewSpinner(rt), ui.NewSpinner(rt), ui.NewSpinner(rt)
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row items-center gap-2"),
			alone.Node(),
			ui.Button(ui.Secondary, ui.SizeSM, twi.Disabled(), inButton.Node(), twi.Text("Please wait")),
			ui.Badge(ui.Outline, inBadge.Node(), twi.Text("Syncing")),
		)
	}
}
