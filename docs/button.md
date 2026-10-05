# Button

Displays a button or a component that looks like a button.

<Preview name="button-demo" />

## Usage

```go
import "github.com/pehcastro/twind/twi/ui"
```

```go
ui.Button(ui.ButtonOutline, ui.ButtonSizeDefault, twi.Text("Button"))
```

A button is focusable. Enter, Space and a click all run its `twi.OnClick` handler.

## Variants

The first argument picks the look: `ui.ButtonDefault`, `ui.ButtonSecondary`, `ui.ButtonDestructive`, `ui.ButtonOutline`, `ui.ButtonGhost` or `ui.ButtonLink`.

<Preview name="button-variants" />

## Sizes

The second argument picks the size: `ui.ButtonSizeDefault`, `ui.ButtonSizeXS`, `ui.ButtonSizeSM`, `ui.ButtonSizeLG` or `ui.ButtonSizeIcon`.

<Preview name="button-sizes" />

## API reference

<Props of="Button" />
