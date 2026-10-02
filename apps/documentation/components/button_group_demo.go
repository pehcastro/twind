package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func ButtonGroupDemo(*twi.Runtime) func() twi.Node {
	outline := func(label string) twi.Node { return ui.Button(ui.Outline, ui.SizeDefault, twi.Text(label)) }
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col items-center gap-1"),
			twi.Element(twi.Class("flex flex-row items-center gap-2"),
				ui.ButtonGroup(ui.Horizontal, outline("Archive"), outline("Report"), outline("Snooze")),
				ui.ButtonGroup(ui.Horizontal, ui.ButtonGroupText(twi.Text("https://")), outline("twind.dev")),
			),
			ui.ButtonGroup(ui.Vertical, outline("+"), outline("-")),
		)
	}
}
