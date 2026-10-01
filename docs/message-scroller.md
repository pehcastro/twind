# Message scroller

A chat column that sticks to the latest message while new ones arrive, and stops following when the reader scrolls up.

<Preview name="message-scroller-demo" />

## Usage

```go
scroller := ui.NewMessageScroller(rt)
```

```go
scroller.Node(twi.Class("h-12"),
	scroller.Viewport(
		scroller.Item(ui.Message(ui.Start, ui.MessageContent(twi.Text("Hello")))),
	),
	scroller.Button(),
)
```

It opens on the latest message. The wheel, PageUp or the arrows on the focused viewport scroll up and stop following; the button that appears jumps back and follows again, and so does scrolling back to the end. Following costs no frames while nothing changes.

## API reference

<Props of="MessageScroller" />
