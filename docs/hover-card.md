# Hover Card

A preview of what is behind a link, shown while the pointer or the focus is on it.

<Preview name="hover-card-demo" />

## Usage

```go
card := ui.NewHoverCard(rt)
```

```go
card.Node(
	card.Trigger(ui.ButtonLink, ui.ButtonSizeDefault, twi.Text("@nextjs")),
	card.Content(twi.Text("The React Framework.")),
)
```

It opens under the trigger; `Side` and `Align` move it.

## API reference

<Props of="HoverCard" />
