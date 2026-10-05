package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func SheetDemo(rt *twi.Runtime) func() twi.Node {
	sheets := map[string]*ui.Dialog{"Top": ui.NewSheet(rt, ui.Top), "Right": ui.NewSheet(rt, ui.Right), "Bottom": ui.NewSheet(rt, ui.Bottom), "Left": ui.NewSheet(rt, ui.Left)}
	toaster := ui.NewToaster(rt)
	toaster.Avoid(sheets["Top"], sheets["Right"], sheets["Bottom"], sheets["Left"])
	name := ui.NewInput(rt)
	name.Insert("Pedro Duarte")
	sheet := func(side string) twi.Node {
		s := sheets[side]
		return twi.Element(twi.Class("flex flex-row"),
			s.Trigger(ui.Outline, ui.SizeDefault, twi.Text(side)),
			s.Content(
				s.Header(s.Title(twi.Text("Edit profile")), s.Description(twi.Text("Make changes to your profile here."))),
				twi.Element(twi.Class("px-2"), ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Name")), name.Node())),
				s.Footer(
					ui.Button(ui.Default, ui.SizeDefault, twi.OnClick(func(*twi.Event) { toaster.Success("Profile saved", "Your changes are live.", ui.ToastAction{}) }), twi.Text("Save changes")),
					s.Close(ui.Outline, ui.SizeDefault, twi.Text("Close")),
				),
			),
		)
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row gap-2"), sheet("Top"), sheet("Right"), sheet("Bottom"), sheet("Left"), toaster.Node())
	}
}
