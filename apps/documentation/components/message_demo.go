package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func MessageDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return ui.MessageGroup(twi.Class("w-56"),
			ui.Message(ui.End,
				ui.MessageAvatar(ui.Avatar(ui.SizeSM, ui.AvatarFallback(twi.Text("PD")))),
				ui.MessageContent(
					ui.MessageHeader(twi.Text("Pedro")),
					ui.Bubble(ui.Default, ui.End, ui.BubbleContent(twi.Text("Is the release notes draft ready?"))),
					ui.MessageFooter(twi.Text("read 2:14 PM")),
				),
			),
			ui.Message(ui.Start,
				ui.MessageAvatar(ui.Avatar(ui.SizeSM, ui.AvatarFallback(twi.Text("AI")))),
				ui.MessageContent(
					ui.MessageHeader(twi.Text("Assistant")),
					ui.Bubble(ui.Muted, ui.Start, ui.BubbleContent(twi.Text("Almost. Two sections left: the driver and the themes."))),
					ui.MessageFooter(twi.Text("2:15 PM")),
				),
			),
		)
	}
}
