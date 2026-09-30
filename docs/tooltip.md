# Tooltip

A short label shown while the pointer or the focus is on an element.

<Preview name="tooltip-demo" />

## Usage

```go
tip := ui.NewTooltip(rt)
```

```go
tip.Node(
	tip.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Hover")),
	tip.Content(twi.Text("Add to library")),
)
```

It shows above the trigger by default and goes away on leave, blur or Escape.

## API reference

<Props of="Tooltip" />
