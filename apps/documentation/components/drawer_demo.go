package components

import (
	"strconv"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func DrawerDemo(rt *twi.Runtime) func() twi.Node {
	drawer, goal := ui.NewDrawer(rt, ui.Bottom), 350
	step := func(label string, by int) twi.Node {
		return ui.Button(ui.Outline, ui.SizeIcon, twi.OnClick(func(*twi.Event) {
			goal = min(max(goal+by, 200), 500)
			rt.Invalidate()
		}), twi.Text(label))
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row"),
			drawer.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Open drawer")),
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
						drawer.Close(ui.Default, ui.SizeDefault, twi.Class("py-1"), twi.Text("Submit")),
						drawer.Close(ui.Outline, ui.SizeDefault, twi.Class("border shadow-none"), twi.Text("Cancel")),
					),
				),
			),
		)
	}
}
