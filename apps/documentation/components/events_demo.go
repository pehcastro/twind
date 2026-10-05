package components

import (
	"strconv"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
)

func EventsDemo(rt *twi.Runtime) func() twi.Node {
	focused, count, last := "", 0, "nothing yet"
	box := func(name string) twi.Node {
		classes := "flex flex-col flex-1 items-center rounded-md border px-1"
		if focused == name {
			classes += " bg-accent text-accent-foreground"
		}
		return twi.Element(twi.Class(classes), twi.Key(name), twi.Focusable(),
			twi.OnFocus(func(*twi.Event) {
				focused = name
				rt.Invalidate()
			}),
			twi.OnClick(func(*twi.Event) {
				last = "a click on " + name
				rt.Invalidate()
			}),
			twi.OnKeyDown(func(e *twi.Event) {
				if e.Key.Key == input.KeyRune && e.Key.Rune == '+' {
					count++
					last = "+ on " + name
					e.StopPropagation()
					rt.Invalidate()
				}
			}),
			twi.Text(name),
		)
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col w-full gap-1"),
			twi.Element(twi.Class("flex flex-row gap-2"), box("one"), box("two"), box("three")),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text("count "+strconv.Itoa(count)+", last: "+last)),
		)
	}
}
