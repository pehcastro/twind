package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func BubbleDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return ui.BubbleGroup(twi.Class("w-48"),
			ui.Bubble(ui.BubbleDefault, ui.AlignStart, ui.BubbleContent(twi.Text("Default"))),
			ui.Bubble(ui.BubbleSecondary, ui.AlignEnd, ui.BubbleContent(twi.Text("Secondary"))),
			ui.Bubble(ui.BubbleTinted, ui.AlignStart, ui.BubbleContent(twi.Text("Tinted"))),
			ui.Bubble(ui.BubbleOutline, ui.AlignEnd, ui.BubbleContent(twi.Text("Outline"))),
			ui.Bubble(ui.BubbleDestructive, ui.AlignStart, ui.BubbleContent(twi.Text("Destructive"))),
			ui.Bubble(ui.BubbleMuted, ui.AlignEnd, ui.BubbleContent(twi.Text("With reactions")), ui.BubbleReactions(ui.SideBottom, ui.AlignEnd, twi.Text("✓ 2"))),
		)
	}
}
