package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func ButtonSizes(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row flex-wrap items-center gap-2"),
			ui.Button(ui.ButtonOutline, ui.ButtonSizeXS, twi.Text("Extra small")),
			ui.Button(ui.ButtonOutline, ui.ButtonSizeSM, twi.Text("Small")),
			ui.Button(ui.ButtonOutline, ui.ButtonSizeDefault, twi.Text("Default")),
			ui.Button(ui.ButtonOutline, ui.ButtonSizeLG, twi.Text("Large")),
			ui.Button(ui.ButtonOutline, ui.ButtonSizeIcon, twi.Text("+")),
		)
	}
}
