package components

import (
	"strconv"
	"unicode/utf8"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func InputGroupDemo(rt *twi.Runtime) func() twi.Node {
	query, url, message, pressed := ui.NewInput(rt), ui.NewInput(rt), ui.NewTextarea(rt), "nothing yet"
	query.Placeholder, message.Placeholder = "Search...", "Ask, search or chat..."
	url.Insert("twind.dev")
	press := func(name string) twi.NodeOption {
		return twi.OnClick(func(*twi.Event) {
			pressed = name
			rt.Invalidate()
		})
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col w-48 gap-1"),
			query.Group(
				ui.InputGroupAddon(ui.InlineStart, ui.InputGroupText(twi.Text("⌕"))),
				ui.InputGroupAddon(ui.InlineEnd, ui.InputGroupText(twi.Text("12 results"))),
			),
			url.Group(
				ui.InputGroupAddon(ui.InlineStart, ui.InputGroupText(twi.Text("https://"))),
				ui.InputGroupAddon(ui.InlineEnd, ui.InputGroupButton(press("Copy"), twi.Text("Copy"))),
			),
			message.Group(ui.InputGroupAddon(ui.BlockEnd,
				ui.InputGroupText(twi.Text(strconv.Itoa(utf8.RuneCountInString(message.Value()))+"/280")),
				twi.Element(twi.Class("grow")),
				ui.InputGroupButton(press("Send"), twi.Text("Send")),
			)),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text("pressed: "+pressed)),
		)
	}
}
