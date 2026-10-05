package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func LabelDemo(rt *twi.Runtime) func() twi.Node {
	terms := ui.NewCheckbox(rt)
	flip := twi.OnClick(func(*twi.Event) {
		terms.Checked = !terms.Checked
		rt.Invalidate()
	})
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row items-center gap-2"),
			terms.Node(),
			ui.Label(flip, twi.Text("Accept terms and conditions")),
		)
	}
}
