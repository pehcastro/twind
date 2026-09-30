package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func TableDemo(*twi.Runtime) func() twi.Node {
	invoices := [][4]string{
		{"INV001", "Paid", "Credit Card", "$250.00"},
		{"INV002", "Pending", "PayPal", "$150.00"},
		{"INV003", "Unpaid", "Bank Transfer", "$350.00"},
	}
	return func() twi.Node {
		var rows []twi.NodeOption
		for _, inv := range invoices {
			rows = append(rows, ui.TableRow(
				ui.TableCell(twi.Class("font-medium"), twi.Text(inv[0])),
				ui.TableCell(twi.Text(inv[1])),
				ui.TableCell(twi.Text(inv[2])),
				ui.TableCell(twi.Class("text-right"), twi.Text(inv[3])),
			))
		}
		return twi.Element(twi.Class("w-60"), ui.Table(
			ui.TableHeader(ui.TableRow(
				ui.TableHead(twi.Text("Invoice")), ui.TableHead(twi.Text("Status")),
				ui.TableHead(twi.Text("Method")), ui.TableHead(twi.Class("text-right"), twi.Text("Amount")),
			)),
			ui.TableBody(rows...),
			ui.TableFooter(ui.TableRow(
				ui.TableCell(twi.Text("Total")), ui.TableCell(), ui.TableCell(),
				ui.TableCell(twi.Class("text-right"), twi.Text("$750.00")),
			)),
			ui.TableCaption(twi.Text("A list of your recent invoices.")),
		))
	}
}
