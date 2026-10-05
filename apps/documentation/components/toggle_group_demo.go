package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func ToggleGroupDemo(rt *twi.Runtime) func() twi.Node {
	align, marks := ui.NewToggleGroup(rt), ui.NewToggleGroup(rt)
	align.Variant, align.Value = ui.ToggleOutline, []string{"left"}
	marks.Multiple, marks.Value = true, []string{"bold"}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row items-center gap-2"),
			align.Node(align.Item("left", twi.Text("Left")), align.Item("center", twi.Text("Center")), align.Item("right", twi.Text("Right"))),
			marks.Node(marks.Item("bold", twi.Text("B")), marks.Item("italic", twi.Text("I")), marks.Item("underline", twi.Text("U"))),
		)
	}
}
