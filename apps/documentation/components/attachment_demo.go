package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func AttachmentDemo(*twi.Runtime) func() twi.Node {
	file := func(u ui.Upload, glyph, name, about string) twi.Node {
		return ui.Attachment(u, ui.Horizontal,
			ui.AttachmentMedia(ui.Icon, twi.Text(glyph)),
			ui.AttachmentContent(ui.AttachmentTitle(twi.Text(name)), ui.AttachmentDescription(twi.Text(about))),
			ui.AttachmentActions(ui.AttachmentAction(twi.Text("✕"))),
		)
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col gap-1"),
			file(ui.Done, "▤", "report.pdf", "1.2 MB"),
			file(ui.Uploading, "▣", "photo.png", "uploading 40%"),
			file(ui.Failed, "▣", "video.mp4", "too large"),
		)
	}
}
