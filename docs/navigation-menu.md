# Navigation Menu

A row of links whose panels open on hover.

<Preview name="navigation-menu-demo" />

## Usage

```go
nav := ui.NewNavigationMenu(rt)
docs := nav.Item("docs")
nav.OnSelect = func(href string) { open(href) }
```

```go
nav.Node(nav.List(
	docs.Node(docs.Trigger(twi.Text("Docs")), docs.Content(
		docs.Link("/docs", twi.Text("Introduction")),
		docs.Link("/docs/installation", twi.Text("Installation")),
	)),
))
```

Hovering a trigger opens its panel after a short pause; once one is open, moving to another switches at once. With the keyboard, Left and Right move, Enter or Down opens, Down and Up move through the links, Enter follows one and Escape closes.

## API reference

<Props of="NavigationMenu" />
