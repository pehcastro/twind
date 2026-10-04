# Direction

Sets the text direction of everything inside it, for right-to-left languages such as Hebrew and Arabic.

<Preview name="direction-demo" />

## Usage

```go
ui.Direction(text.DirRTL,
	ui.Card(ui.CardTitle(twi.Text("שלום"))),
)
```

Any element can take the same option directly: `twi.Element(twi.Dir(text.DirRTL), ...)`. The direction is inherited, as CSS `direction` is, until a descendant sets its own.

Inside a right-to-left element, text starts at the right edge and mixed text is laid out in visual order. A flex row starts at the right too, so `justify-start` packs to the right and `justify-end` to the left. `text-center` and `text-right` keep their meaning. `text-left` and `text-start` are one value in the Style IR, as are `text-right` and `text-end`, so for now `text-left` follows the direction and `text-end` does not.

## API reference

<Props of="Direction" />
