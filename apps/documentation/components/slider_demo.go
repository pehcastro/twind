package components

import (
	"strconv"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func SliderDemo(rt *twi.Runtime) func() twi.Node {
	volume := ui.NewSlider(rt)
	volume.Value = 33
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row w-48 items-center gap-2"),
			volume.Node(),
			twi.Element(twi.Class("w-3 shrink-0 text-right text-muted-foreground"), twi.Text(strconv.Itoa(volume.Value))),
		)
	}
}
