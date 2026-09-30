# Toaster

Short messages that stack in the corner and leave on their own.

<Preview name="toaster-demo" />

## Usage

```go
toaster := ui.NewToaster(rt)
```

Render `toaster.Node()` once, anywhere in the tree, then add toasts from any handler:

```go
toaster.Success("Event has been created", "Sunday, December 03 at 9:00", ui.ToastAction{
	Label:   "Undo",
	OnClick: undo,
})
```

The newest toast shows in full with the older ones as edges behind it. The pointer over the stack spreads it out and holds every toast until it leaves; Escape dismisses the newest.

## API reference

<Props of="Toaster" />
