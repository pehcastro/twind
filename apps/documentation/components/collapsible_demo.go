package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func CollapsibleDemo(rt *twi.Runtime) func() twi.Node {
	collapsible := ui.NewCollapsible(rt)
	repo := func(name string) twi.Node {
		return twi.Element(twi.Class("rounded-md border px-2"), twi.Text(name))
	}
	return func() twi.Node {
		return collapsible.Node(twi.Class("w-48 gap-1"),
			twi.Element(twi.Class("flex flex-row items-center justify-between"),
				twi.Element(twi.Class("font-semibold"), twi.Text("@peduarte starred 3 repositories")),
				collapsible.Trigger(ui.ButtonGhost, ui.ButtonSizeIcon, twi.Text("⇅")),
			),
			repo("@radix-ui/primitives"),
			collapsible.Content(twi.Class("gap-1"), repo("@radix-ui/colors"), repo("@stitches/react")),
		)
	}
}
