# Combobox

A text field with a list of suggestions that filters as you type.

<Preview name="combobox-demo" />

## Usage

```go
framework := ui.NewCombobox(rt)
framework.Placeholder = "Select framework..."
```

```go
framework.Node(
	framework.Input(),
	framework.Content(
		framework.Item("next", "Next.js"),
		framework.Item("svelte", "SvelteKit"),
	),
)
```

Typing opens the list and keeps the items that match. Up and Down move, Enter picks, Escape closes and puts back the chosen label. The chevron opens the whole list.

## API reference

<Props of="Combobox" />
