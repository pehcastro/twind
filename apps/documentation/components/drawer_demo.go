package components

import (
	"strconv"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func DrawerDemo(rt *twi.Runtime) func() twi.Node {
	drawer, goal := ui.NewDrawer(rt, ui.SideBottom), 350
	step := func(label string, by int) twi.Node {
		return ui.Button(ui.ButtonOutline, ui.ButtonSizeIcon, twi.OnClick(func(*twi.Event) {
			goal = min(max(goal+by, 200), 500)
			rt.Invalidate()
		}), twi.Text(label))
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row"),
			drawer.Trigger(ui.ButtonOutline, ui.ButtonSizeDefault, twi.Text("Open drawer")),
			drawer.Content(
				twi.Element(twi.Class("flex flex-col w-full max-w-48 self-center"),
					drawer.Header(drawer.Title(twi.Text("Move goal")), drawer.Description(twi.Text("Set your daily activity goal."))),
					twi.Element(twi.Class("flex flex-row items-center justify-center gap-4"),
						step("-", -10),
						twi.Element(twi.Class("flex flex-col items-center"),
							twi.Element(twi.Class("font-bold"), twi.Text(strconv.Itoa(goal))),
							twi.Element(twi.Class("text-muted-foreground"), twi.Text("calories a day")),
						),
						step("+", 10),
					),
					drawer.Footer(
						drawer.Close(ui.ButtonDefault, ui.ButtonSizeDefault, twi.Class("py-1"), twi.Text("Submit")),
						drawer.Close(ui.ButtonOutline, ui.ButtonSizeDefault, twi.Class("border shadow-none"), twi.Text("Cancel")),
					),
				),
			),
		)
	}
}
