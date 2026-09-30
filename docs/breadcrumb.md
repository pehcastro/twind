# Breadcrumb

Displays the path to the current page as a row of links.

<Preview name="breadcrumb-demo" />

## Usage

```go
ui.Breadcrumb(ui.BreadcrumbList(
	ui.BreadcrumbItem(ui.BreadcrumbLink(twi.OnClick(goHome), twi.Text("Home"))),
	ui.BreadcrumbSeparator(),
	ui.BreadcrumbItem(ui.BreadcrumbPage(twi.Text("Breadcrumb"))),
))
```

A link is plain text until you give it a `twi.OnClick`. The breadcrumb at the top of this page is one.

## API reference

<Props of="Breadcrumb" />
