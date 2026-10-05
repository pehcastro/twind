package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func ButtonVariants(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row flex-wrap items-center gap-2"),
			ui.Button(ui.ButtonDefault, ui.ButtonSizeDefault, twi.Text("Default")),
			ui.Button(ui.ButtonSecondary, ui.ButtonSizeDefault, twi.Text("Secondary")),
			ui.Button(ui.ButtonDestructive, ui.ButtonSizeDefault, twi.Text("Destructive")),
			ui.Button(ui.ButtonOutline, ui.ButtonSizeDefault, twi.Text("Outline")),
			ui.Button(ui.ButtonGhost, ui.ButtonSizeDefault, twi.Text("Ghost")),
			ui.Button(ui.ButtonLink, ui.ButtonSizeDefault, twi.Text("Link")),
		)
	}
}
