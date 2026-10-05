package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func SelectDemo(rt *twi.Runtime) func() twi.Node {
	fruit := ui.NewSelect(rt)
	fruit.Placeholder = "Select a fruit"
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col h-12"),
			fruit.Node(twi.Class("w-30"),
				fruit.Trigger(twi.Class("w-30")),
				fruit.Content(
					ui.SelectLabel(twi.Text("Fruits")),
					fruit.Item("apple", "Apple"), fruit.Item("banana", "Banana"), fruit.Item("blueberry", "Blueberry"),
					ui.SelectSeparator(),
					ui.SelectLabel(twi.Text("Vegetables")),
					fruit.Item("carrot", "Carrot"), fruit.Item("leek", "Leek"),
				),
			),
		)
	}
}
