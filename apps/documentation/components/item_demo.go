package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func ItemDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return ui.ItemGroup(twi.Class("w-full max-w-56 gap-1"),
			ui.Item(ui.ItemOutline, ui.ItemSizeDefault,
				ui.ItemContent(ui.ItemTitle(twi.Text("Basic item")), ui.ItemDescription(twi.Text("A title and a description."))),
				ui.ItemActions(ui.Button(ui.ButtonOutline, ui.ButtonSizeSM, twi.Text("Action"))),
			),
			ui.Item(ui.ItemMuted, ui.ItemSizeSM,
				ui.ItemMedia(ui.ItemMediaIcon, twi.Text("✓")),
				ui.ItemContent(ui.ItemTitle(twi.Text("Your profile has been verified."))),
				ui.ItemActions(twi.Text("›")),
			),
			ui.ItemSeparator(),
			ui.Item(ui.ItemDefault, ui.ItemSizeDefault,
				ui.ItemMedia(ui.ItemMediaDefault, ui.Avatar(ui.AvatarSizeSM, ui.AvatarFallback(twi.Text("ER")))),
				ui.ItemContent(ui.ItemTitle(twi.Text("evilrabbit")), ui.ItemDescription(twi.Text("Last seen 5 months ago"))),
			),
		)
	}
}
