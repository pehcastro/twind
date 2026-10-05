package components

import (
	"strconv"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func PaginationDemo(rt *twi.Runtime) func() twi.Node {
	page := 0
	turn := func(to func() int) twi.NodeOption {
		return twi.OnClick(func(*twi.Event) {
			page = min(max(to(), 0), 2)
			rt.Invalidate()
		})
	}
	return func() twi.Node {
		links := []twi.NodeOption{ui.PaginationItem(ui.PaginationPrevious(turn(func() int { return page - 1 })))}
		for i := range 3 {
			links = append(links, ui.PaginationItem(ui.PaginationLink(ui.Active(i == page), turn(func() int { return i }), twi.Text(strconv.Itoa(i+1)))))
		}
		links = append(links, ui.PaginationItem(ui.PaginationEllipsis()), ui.PaginationItem(ui.PaginationNext(turn(func() int { return page + 1 }))))
		return twi.Element(twi.Class("flex flex-col w-56 items-center gap-1"),
			ui.Pagination(ui.PaginationContent(links...)),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text("page "+strconv.Itoa(page+1)+" of 3")),
		)
	}
}
