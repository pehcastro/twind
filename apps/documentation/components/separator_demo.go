package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func SeparatorDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col w-48 gap-1"),
			twi.Element(twi.Class("font-medium"), twi.Text("Twind")),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text("A UI runtime for terminal programs.")),
			ui.Separator(ui.Horizontal),
			twi.Element(twi.Class("flex flex-row h-1 gap-2"),
				twi.Text("Blog"), ui.Separator(ui.Vertical),
				twi.Text("Docs"), ui.Separator(ui.Vertical),
				twi.Text("Source"),
			),
		)
	}
}
