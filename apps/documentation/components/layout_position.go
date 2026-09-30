package components

import "github.com/twind-dev/twind/twi"

func LayoutPosition(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row w-full gap-2"),
			twi.Element(twi.Class("relative flex flex-col flex-1 h-5 items-center justify-center rounded-lg border bg-card"),
				twi.Element(twi.Class("font-medium"), twi.Text("relative")),
				twi.Element(twi.Class("absolute -top-1 right-1 rounded-full bg-primary px-1 text-primary-foreground"), twi.Text("absolute -top-1 right-1")),
			),
			twi.Element(twi.Class("flex flex-col flex-1 h-5 overflow-hidden rounded-lg border bg-card px-1"),
				twi.Element(twi.Class("font-medium"), twi.Text("overflow-hidden")),
				twi.Element(twi.Class("w-60"), twi.Text("This line is longer than its box, so the box cuts it at the border.")),
			),
		)
	}
}
