# Collapsible

A panel that a trigger shows and hides.

<Preview name="collapsible-demo" />

## Usage

```go
more := ui.NewCollapsible(rt)
```

```go
more.Node(
	more.Trigger(ui.ButtonGhost, ui.ButtonSizeIcon, twi.Text("⇅")),
	more.Content(twi.Text("Shown while open")),
)
```

`AsTrigger()` turns any element into the trigger, such as a whole heading row.

## API reference

<Props of="Collapsible" />
