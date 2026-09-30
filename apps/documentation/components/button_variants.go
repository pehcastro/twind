package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func ButtonVariants(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row flex-wrap items-center gap-2"),
			ui.Button(ui.Default, ui.SizeDefault, twi.Text("Default")),
			ui.Button(ui.Secondary, ui.SizeDefault, twi.Text("Secondary")),
			ui.Button(ui.Destructive, ui.SizeDefault, twi.Text("Destructive")),
			ui.Button(ui.Outline, ui.SizeDefault, twi.Text("Outline")),
			ui.Button(ui.Ghost, ui.SizeDefault, twi.Text("Ghost")),
			ui.Button(ui.Link, ui.SizeDefault, twi.Text("Link")),
		)
	}
}
