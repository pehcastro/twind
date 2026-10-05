package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func CheckboxDemo(rt *twi.Runtime) func() twi.Node {
	terms, notify, disabled := ui.NewCheckbox(rt), ui.NewCheckbox(rt), ui.NewCheckbox(rt)
	notify.Checked, disabled.Disabled = true, true
	row := func(box *ui.Checkbox, label string) twi.Node {
		return twi.Element(twi.Class("flex flex-row items-center gap-2"), box.Node(), ui.Label(twi.Text(label)))
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col gap-1"),
			row(terms, "Accept terms and conditions"),
			row(notify, "Enable notifications"),
			row(disabled, "Disabled"),
		)
	}
}
