package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func ButtonSizes(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row flex-wrap items-center gap-2"),
			ui.Button(ui.Outline, ui.SizeXS, twi.Text("Extra small")),
			ui.Button(ui.Outline, ui.SizeSM, twi.Text("Small")),
			ui.Button(ui.Outline, ui.SizeDefault, twi.Text("Default")),
			ui.Button(ui.Outline, ui.SizeLG, twi.Text("Large")),
			ui.Button(ui.Outline, ui.SizeIcon, twi.Text("+")),
		)
	}
}
