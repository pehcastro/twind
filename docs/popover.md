# Popover

Rich content in a panel anchored to its trigger.

<Preview name="popover-demo" />

## Usage

```go
popover := ui.NewPopover(rt)
```

```go
popover.Node(
	popover.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Open popover")),
	popover.Content(twi.Text("Place content for the popover here.")),
)
```

It opens under the trigger, fades and zooms in, and keeps Tab inside while open. Escape or a press outside closes it. `Side` and `Align` place it.

## API reference

<Props of="Popover" />
