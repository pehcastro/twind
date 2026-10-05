package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func PopoverDemo(rt *twi.Runtime) func() twi.Node {
	popover, width, height := ui.NewPopover(rt), ui.NewInput(rt), ui.NewInput(rt)
	popover.Align = ui.Start
	width.Insert("100%")
	height.Insert("25px")
	size := func(label string, in *ui.Input) twi.Node {
		return twi.Element(twi.Class("flex flex-row items-center gap-2"),
			twi.Element(twi.Class("w-7"), ui.Label(twi.Text(label))),
			in.Node(),
		)
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col h-9"),
			popover.Node(
				popover.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Open popover")),
				popover.Content(
					twi.Element(twi.Class("font-medium"), twi.Text("Dimensions")),
					twi.Element(twi.Class("text-muted-foreground"), twi.Text("Set the size of the layer.")),
					size("Width", width),
					size("Height", height),
				),
			),
		)
	}
}
