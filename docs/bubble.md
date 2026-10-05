# Bubble

The coloured box around a message, in seven variants, with optional reactions.

<Preview name="bubble-demo" />

## Usage

```go
ui.Bubble(ui.BubbleSecondary, ui.AlignStart,
	ui.BubbleContent(twi.Text("See you at three.")),
	ui.BubbleReactions(ui.SideBottom, ui.AlignEnd, twi.Text("✓ 2")),
)
```

A bubble is never wider than four fifths of its message; longer text wraps inside it.

## API reference

<Props of="Bubble" />
