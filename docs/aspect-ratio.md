# Aspect Ratio

Displays content within a desired ratio.

<Preview name="aspect-ratio-demo" />

## Usage

```go
ui.AspectRatio(twi.Class("aspect-video rounded-lg bg-muted"))
```

The height follows the width. A terminal cell is taller than it is wide, so Twind computes the ratio in the terminal's own cell pixels: a 16:9 box looks 16:9 on screen, not in character counts.

## API reference

<Props of="AspectRatio" />
