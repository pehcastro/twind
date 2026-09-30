package components

import "github.com/twind-dev/twind/twi"

func LayoutFlex(*twi.Runtime) func() twi.Node {
	box := func(classes, label string) twi.Node {
		return twi.Element(twi.Class("rounded-md bg-secondary px-1 text-secondary-foreground "+classes), twi.Text(label))
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col w-full gap-1"),
			twi.Element(twi.Class("flex flex-row gap-1"),
				box("w-10", "w-10"),
				box("grow bg-primary text-primary-foreground", "grow"),
				box("grow-2 bg-accent", "grow-2"),
				box("w-10", "w-10"),
			),
			twi.Element(twi.Class("flex flex-row justify-between rounded-md border px-1"),
				twi.Text("justify-between"), twi.Text("puts"), twi.Text("the space"), twi.Text("between"),
			),
			twi.Element(twi.Class("flex flex-row h-3 items-center justify-center rounded-md border"),
				box("", "items-center justify-center"),
			),
		)
	}
}
