package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func MessageDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return ui.MessageGroup(twi.Class("w-56"),
			ui.Message(ui.AlignEnd,
				ui.MessageAvatar(ui.Avatar(ui.AvatarSizeSM, ui.AvatarFallback(twi.Text("PD")))),
				ui.MessageContent(
					ui.MessageHeader(twi.Text("Pedro")),
					ui.Bubble(ui.BubbleDefault, ui.AlignEnd, ui.BubbleContent(twi.Text("Is the release notes draft ready?"))),
					ui.MessageFooter(twi.Text("read 2:14 PM")),
				),
			),
			ui.Message(ui.AlignStart,
				ui.MessageAvatar(ui.Avatar(ui.AvatarSizeSM, ui.AvatarFallback(twi.Text("AI")))),
				ui.MessageContent(
					ui.MessageHeader(twi.Text("Assistant")),
					ui.Bubble(ui.BubbleMuted, ui.AlignStart, ui.BubbleContent(twi.Text("Almost. Two sections left: the driver and the themes."))),
					ui.MessageFooter(twi.Text("2:15 PM")),
				),
			),
		)
	}
}
