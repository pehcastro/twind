package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func BubbleDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return ui.BubbleGroup(twi.Class("w-48"),
			ui.Bubble(ui.Default, ui.Start, ui.BubbleContent(twi.Text("Default"))),
			ui.Bubble(ui.Secondary, ui.End, ui.BubbleContent(twi.Text("Secondary"))),
			ui.Bubble(ui.Tinted, ui.Start, ui.BubbleContent(twi.Text("Tinted"))),
			ui.Bubble(ui.Outline, ui.End, ui.BubbleContent(twi.Text("Outline"))),
			ui.Bubble(ui.Destructive, ui.Start, ui.BubbleContent(twi.Text("Destructive"))),
			ui.Bubble(ui.Muted, ui.End, ui.BubbleContent(twi.Text("With reactions")), ui.BubbleReactions(ui.Bottom, ui.End, twi.Text("✓ 2"))),
		)
	}
}
