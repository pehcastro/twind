package components

import (
	"strconv"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func ProgressDemo(rt *twi.Runtime) func() twi.Node {
	done := 60
	step := func(label string, by int) twi.Node {
		return ui.Button(ui.Outline, ui.SizeSM, twi.OnClick(func(*twi.Event) {
			done = min(max(done+by, 0), 100)
			rt.Invalidate()
		}), twi.Text(label))
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col w-48 items-center gap-1"),
			ui.Progress(done),
			twi.Element(twi.Class("flex flex-row items-center gap-2"),
				step("- 10", -10),
				twi.Element(twi.Class("w-5 text-center"), twi.Text(strconv.Itoa(done)+"%")),
				step("+ 10", 10),
			),
		)
	}
}
