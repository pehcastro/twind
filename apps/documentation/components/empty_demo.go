package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func EmptyDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return ui.Empty(twi.Class("w-56 border border-dashed"),
			ui.EmptyHeader(
				ui.EmptyMedia(ui.Icon, twi.Text("▣")),
				ui.EmptyTitle(twi.Text("No projects yet")),
				ui.EmptyDescription(twi.Text("You have not created a project yet. Start by creating your first one.")),
			),
			ui.EmptyContent(twi.Element(twi.Class("flex flex-row gap-2"),
				ui.Button(ui.Default, ui.SizeDefault, twi.Text("Create project")),
				ui.Button(ui.Outline, ui.SizeDefault, twi.Text("Import project")),
			)),
		)
	}
}
