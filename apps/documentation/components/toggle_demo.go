package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func ToggleDemo(rt *twi.Runtime) func() twi.Node {
	bold, italic, underline := ui.NewToggle(rt), ui.NewToggle(rt), ui.NewToggle(rt)
	italic.Variant, italic.Size, italic.Pressed = ui.ToggleOutline, ui.ToggleSizeSM, true
	underline.Size = ui.ToggleSizeLG
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row items-center gap-2"),
			bold.Node(twi.Text("B")),
			italic.Node(twi.Text("I")),
			underline.Node(twi.Text("U")),
		)
	}
}
