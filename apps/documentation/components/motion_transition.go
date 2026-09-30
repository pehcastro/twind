package components

import "github.com/twind-dev/twind/twi"

func MotionTransition(*twi.Runtime) func() twi.Node {
	card := func(classes, label string) twi.Node {
		return twi.Element(twi.Class("flex flex-col flex-1 rounded-lg border px-2 py-1 "+classes),
			twi.Element(twi.Class("font-medium"), twi.Text(label)),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text("move the pointer over it")),
		)
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row w-full gap-2"),
			card("transition-colors duration-150 hover:bg-accent", "150 ms"),
			card("transition-colors duration-500 hover:bg-primary hover:text-primary-foreground", "500 ms"),
			card("hover:bg-accent", "no transition"),
		)
	}
}
