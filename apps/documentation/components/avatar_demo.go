package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func AvatarDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row items-center gap-2"),
			ui.Avatar(ui.AvatarSizeSM, ui.AvatarFallback(twi.Text("CN"))),
			ui.Avatar(ui.AvatarSizeDefault, ui.AvatarFallback(twi.Text("CN"))),
			ui.Avatar(ui.AvatarSizeLG, ui.AvatarFallback(twi.Text("ER"))),
		)
	}
}
