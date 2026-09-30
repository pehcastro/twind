package components

import "github.com/twind-dev/twind/twi"

func LayoutGrid(*twi.Runtime) func() twi.Node {
	cell := func(classes, label string) twi.Node {
		return twi.Element(twi.Class("rounded-md bg-secondary px-1 text-secondary-foreground "+classes), twi.Text(label))
	}
	return func() twi.Node {
		return twi.Element(twi.Class("grid w-full grid-cols-3 gap-1"),
			cell("col-span-2 bg-primary text-primary-foreground", "col-span-2"),
			cell("", "one"),
			cell("", "two"),
			cell("", "three"),
			cell("", "four"),
		)
	}
}
