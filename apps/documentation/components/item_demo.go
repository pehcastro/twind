package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func ItemDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return ui.ItemGroup(twi.Class("w-56 gap-1"),
			ui.Item(ui.Outline, ui.SizeDefault,
				ui.ItemContent(ui.ItemTitle(twi.Text("Basic item")), ui.ItemDescription(twi.Text("A title and a description."))),
				ui.ItemActions(ui.Button(ui.Outline, ui.SizeSM, twi.Text("Action"))),
			),
			ui.Item(ui.Muted, ui.SizeSM,
				ui.ItemMedia(ui.Icon, twi.Text("✓")),
				ui.ItemContent(ui.ItemTitle(twi.Text("Your profile has been verified."))),
				ui.ItemActions(twi.Text("›")),
			),
			ui.ItemSeparator(),
			ui.Item(ui.Default, ui.SizeDefault,
				ui.ItemMedia(ui.Default, ui.Avatar(ui.SizeSM, ui.AvatarFallback(twi.Text("ER")))),
				ui.ItemContent(ui.ItemTitle(twi.Text("evilrabbit")), ui.ItemDescription(twi.Text("Last seen 5 months ago"))),
			),
		)
	}
}
