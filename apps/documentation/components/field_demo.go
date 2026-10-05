package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func FieldDemo(rt *twi.Runtime) func() twi.Node {
	name, number, same := ui.NewInput(rt), ui.NewInput(rt), ui.NewCheckbox(rt)
	name.Placeholder = "Evil Rabbit"
	number.Insert("1234 5678")
	number.Invalid = true
	return func() twi.Node {
		return ui.FieldSet(twi.Class("w-full max-w-56"),
			ui.FieldLegend(twi.Text("Payment method")),
			ui.FieldGroup(
				ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Name on card")), name.Node(), ui.FieldDescription(twi.Text("As it is printed on the card"))),
				ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Card number")), number.Node(), ui.FieldError(twi.Text("Enter a valid card number"))),
				ui.FieldSeparator(twi.Text("or")),
				ui.Field(ui.Horizontal, same.Node(), ui.FieldLabel(twi.Text("Same as the shipping address"))),
			),
		)
	}
}
