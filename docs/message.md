# Message

One turn in a conversation: an avatar, a header, bubbles and a footer.

<Preview name="message-demo" />

## Usage

```go
ui.Message(ui.End,
	ui.MessageAvatar(ui.Avatar(ui.SizeSM, ui.AvatarFallback(twi.Text("PD")))),
	ui.MessageContent(
		ui.MessageHeader(twi.Text("Pedro")),
		ui.Bubble(ui.Default, ui.End, ui.BubbleContent(twi.Text("Is the draft ready?"))),
		ui.MessageFooter(twi.Text("read 2:14 PM")),
	),
)
```

`End` is for the sender: the avatar moves to the right and the content aligns right. Message text is data; it is sanitised before it reaches the terminal.

## API reference

<Props of="Message" />
