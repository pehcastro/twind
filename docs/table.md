# Table

Rows and columns of data.

<Preview name="table-demo" />

## Usage

```go
ui.Table(
	ui.TableHeader(ui.TableRow(ui.TableHead(twi.Text("Invoice")), ui.TableHead(twi.Text("Amount")))),
	ui.TableBody(
		ui.TableRow(ui.TableCell(twi.Text("INV001")), ui.TableCell(twi.Text("$250.00"))),
	),
	ui.TableCaption(twi.Text("A list of your recent invoices.")),
)
```

Cells in a row share its width equally; a class such as `w-12` or `grow-2` on a head and its cells changes a column. The tables on these pages, the props tables included, are this component.

## API reference

<Props of="Table" />
