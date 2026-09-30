package components

import (
	"strconv"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func ScrollAreaDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		tags := []twi.NodeOption{twi.Class("h-10 w-30 rounded-md border"), twi.Element(twi.Class("px-1 font-medium"), twi.Text("Tags"))}
		for i := range 30 {
			tags = append(tags, twi.Element(twi.Class("px-1"), twi.Text("v1.2.0-beta."+strconv.Itoa(30-i))))
		}
		return ui.ScrollArea(tags...)
	}
}
