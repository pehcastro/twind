package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func HoverCardDemo(rt *twi.Runtime) func() twi.Node {
	card := ui.NewHoverCard(rt)
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col h-9"),
			card.Node(
				card.Trigger(ui.Link, ui.SizeDefault, twi.Text("@nextjs")),
				card.Content(
					twi.Element(twi.Class("font-semibold"), twi.Text("@nextjs")),
					twi.Text("The React Framework, created and maintained by @vercel."),
					twi.Element(twi.Class("text-muted-foreground"), twi.Text("Joined December 2021")),
				),
			),
		)
	}
}
