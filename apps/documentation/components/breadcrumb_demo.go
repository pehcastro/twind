package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func BreadcrumbDemo(rt *twi.Runtime) func() twi.Node {
	opened := "Components"
	link := func(name string) twi.Node {
		return ui.BreadcrumbItem(ui.BreadcrumbLink(twi.OnClick(func(*twi.Event) {
			opened = name
			rt.Invalidate()
		}), twi.Text(name)))
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col items-center gap-1"),
			ui.Breadcrumb(ui.BreadcrumbList(
				link("Home"), ui.BreadcrumbSeparator(),
				ui.BreadcrumbItem(ui.BreadcrumbEllipsis()), ui.BreadcrumbSeparator(),
				link("Components"), ui.BreadcrumbSeparator(),
				ui.BreadcrumbItem(ui.BreadcrumbPage(twi.Text("Breadcrumb"))),
			)),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text("last clicked: "+opened)),
		)
	}
}
