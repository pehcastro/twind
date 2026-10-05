package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func ToasterDemo(rt *twi.Runtime) func() twi.Node {
	toaster := ui.NewToaster(rt)
	button := func(label string, show func(title, description string, actions ...ui.ToastAction)) twi.Node {
		return ui.Button(ui.ButtonOutline, ui.ButtonSizeDefault, twi.OnClick(func(*twi.Event) {
			show("Event has been created", "Sunday, December 03 at 9:00", ui.ToastAction{Label: "Undo"})
		}), twi.Text(label))
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row gap-2"),
			button("Show toast", toaster.Show),
			button("Success", toaster.Success),
			button("Error", toaster.Error),
			toaster.Node(),
		)
	}
}
