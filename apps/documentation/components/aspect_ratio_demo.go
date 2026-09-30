package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func AspectRatioDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row items-end gap-2"),
			twi.Element(twi.Class("flex flex-col w-30 gap-1"),
				ui.AspectRatio(twi.Class("aspect-video rounded-lg bg-linear-to-br from-primary to-chart-2")),
				twi.Element(twi.Class("text-muted-foreground"), twi.Text("aspect-video")),
			),
			twi.Element(twi.Class("flex flex-col w-12 gap-1"),
				ui.AspectRatio(twi.Class("items-center justify-center rounded-lg bg-muted"), twi.Text("1:1")),
				twi.Element(twi.Class("text-muted-foreground"), twi.Text("aspect-square")),
			),
		)
	}
}
