package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func AccordionDemo(rt *twi.Runtime) func() twi.Node {
	accordion := ui.NewAccordion(rt)
	accordion.Collapsible, accordion.Value = true, []string{"shipping"}
	item := func(value, question, answer string) twi.Node {
		return accordion.Item(value,
			accordion.Trigger(value, twi.Text(question)),
			accordion.Content(value, twi.Text(answer)),
		)
	}
	return func() twi.Node {
		return accordion.Node(twi.Class("w-full max-w-56"),
			item("shipping", "What are your shipping options?", "Standard in 5 to 7 days, express in 2 to 3, or overnight."),
			item("returns", "What is your return policy?", "Returns within 30 days, in the original packaging."),
			item("support", "How can I reach support?", "By email, chat or phone, around the clock."),
		)
	}
}
