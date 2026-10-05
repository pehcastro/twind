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

Actions are optional: `toaster.Show(title, description)` adds a toast with only its close button, and each action passed adds a button.

The newest toast shows in full with the older ones as edges behind it. The pointer over the stack spreads it out and holds every toast until it leaves; Escape dismisses the newest.

## Beside a sheet or drawer

A toast sits in the bottom right corner, where an open sheet or drawer has its buttons. Give the toaster those panels and the stack moves out of their way while they are open: to the left of a right sheet, clear of a left one, and above a bottom drawer or sheet.

```go
toaster.Avoid(cart, settings)
```

`Avoid` takes sheets and drawers only, from `ui.NewSheet` and `ui.NewDrawer`, and panics on any other dialog. The [Sheet](sheet.md) demo saves through a toast that stays clear of the open sheet.

## API reference

<Props of="Toaster" />
