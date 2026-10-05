# Button

Displays a button or a component that looks like a button.

<Preview name="button-demo" />

## Usage

```go
import "github.com/pehcastro/twind/twi/ui"
```

```go
ui.Button(ui.Outline, ui.SizeDefault, twi.Text("Button"))
```

A button is focusable. Enter, Space and a click all run its `twi.OnClick` handler.

## Variants

The first argument picks the look: `ui.Default`, `ui.Secondary`, `ui.Destructive`, `ui.Outline`, `ui.Ghost` or `ui.Link`.

<Preview name="button-variants" />

## Sizes

The second argument picks the size: `ui.SizeDefault`, `ui.SizeXS`, `ui.SizeSM`, `ui.SizeLG` or `ui.SizeIcon`.

<Preview name="button-sizes" />

## API reference

<Props of="Button" />
