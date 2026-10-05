# Attachment

A file in a message or a composer: its media, name, size and state.

<Preview name="attachment-demo" />

## Usage

```go
ui.Attachment(ui.UploadFailed, ui.Horizontal,
	ui.AttachmentMedia(ui.AttachmentMediaIcon, twi.Text("▣")),
	ui.AttachmentContent(
		ui.AttachmentTitle(twi.Text("video.mp4")),
		ui.AttachmentDescription(twi.Text("too large")),
	),
	ui.AttachmentActions(ui.AttachmentAction(twi.Text("✕"))),
)
```

`Failed` turns the border, the media and the description destructive; `Idle` draws a dashed border for a drop target.

## API reference

<Props of="Attachment" />
