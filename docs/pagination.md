# Pagination

Previous, numbered pages and next.

<Preview name="pagination-demo" />

## Usage

```go
ui.Pagination(ui.PaginationContent(
	ui.PaginationItem(ui.PaginationPrevious(twi.OnClick(back))),
	ui.PaginationItem(ui.PaginationLink(true, twi.Text("1"))),
	ui.PaginationItem(ui.PaginationLink(false, twi.OnClick(toPage2), twi.Text("2"))),
	ui.PaginationItem(ui.PaginationEllipsis()),
	ui.PaginationItem(ui.PaginationNext(twi.OnClick(forward))),
))
```

The parts are buttons; the page you are on is up to you, passed as `isActive`.

## API reference

<Props of="Pagination" />
