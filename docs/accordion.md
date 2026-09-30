# Accordion

A vertically stacked set of headings that each reveal a section of content.

<Preview name="accordion-demo" />

## Usage

```go
accordion := ui.NewAccordion(rt)
```

```go
accordion.Node(
	accordion.Item("shipping",
		accordion.Trigger("shipping", twi.Text("What are your shipping options?")),
		accordion.Content("shipping", twi.Text("Standard, express or overnight.")),
	),
)
```

A click on a heading opens its section. With the accordion focused, Up and Down move between headings, Home and End jump to the ends, and Enter or Space opens or closes.

## One or many

`Type` is `ui.Single` by default, so opening a section closes the others. Set `Collapsible` to let the open one close too, or set `Type` to `ui.Multiple` to open any number.

## API reference

<Props of="Accordion" />
