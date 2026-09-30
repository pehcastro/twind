package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func NavigationMenuDemo(rt *twi.Runtime) func() twi.Node {
	nav, followed := ui.NewNavigationMenu(rt), "nothing yet"
	started, parts := nav.Item("started"), nav.Item("components")
	nav.OnSelect = func(href string) { followed = href }
	link := func(item *ui.NavigationMenuItem, href, title, about string) twi.Node {
		return item.Link(href,
			twi.Element(twi.Class("font-medium"), twi.Text(title)),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text(about)),
		)
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col h-12 gap-1"),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text("followed: "+followed)),
			nav.Node(nav.List(
				started.Node(started.Trigger(twi.Text("Getting started")), started.Content(twi.Class("w-40"),
					link(started, "/docs", "Introduction", "What Twind is and how it works."),
					link(started, "/docs/installation", "Installation", "Add Twind to a Go module."),
				)),
				parts.Node(parts.Trigger(twi.Text("Components")), parts.Content(twi.Class("w-40"),
					link(parts, "/docs/alert-dialog", "Alert Dialog", "A modal that asks before it acts."),
					link(parts, "/docs/hover-card", "Hover Card", "A preview behind a link."),
				)),
			)),
		)
	}
}
