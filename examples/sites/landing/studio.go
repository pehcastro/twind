package main

import (
	"strings"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func newStudio() func() twi.Node {
	big := func(class, s string) twi.Node {
		return txt("font-bold whitespace-pre "+class, strings.Join(strings.Split(strings.ToUpper(s), ""), " "))
	}
	work := func(n, client, kind, year string) twi.Node {
		return el("flex flex-row items-center gap-3 px-1 border-b hover:bg-accent hover:text-accent-foreground",
			txt("w-4 text-muted-foreground", n), txt("grow font-medium", client), txt("w-28 text-muted-foreground", kind), txt("w-6 text-muted-foreground", year), twi.Text("→"))
	}
	return func() twi.Node {
		return el("flex flex-col shrink-0",
			section("flex-row items-center gap-2 py-0.5",
				txt("font-semibold", "atelier nine"), el("grow"), links("", "Work", "Studio", "Journal", "Contact"),
			),
			section("gap-0 pt-4 pb-3",
				big("", "We design quiet"),
				big("", "software for"),
				big("text-muted-foreground", "loud markets."),
				el("flex flex-row items-end gap-4 pt-2",
					txt("text-muted-foreground max-w-48", "A small, independent studio for product design and front-end engineering. Twelve people, one floor, no account managers."),
					el("grow"),
					ui.Button(ui.Outline, ui.SizeDefault, twi.Text("Start a project ↗")),
				),
			),
			section("gap-0 pt-2",
				el("flex flex-row justify-between pb-0.5 border-b", txt("font-medium", "Selected work"), txt("text-muted-foreground", "2023 to 2026")),
				work("01", "Harbor Ledger", "Fintech, design system", "2026"),
				work("02", "Fieldnote", "Research app, iOS", "2025"),
				work("03", "Morrow Transit", "Live timetables", "2025"),
				work("04", "Kiln & Co.", "Shop and brand", "2024"),
				work("05", "Open Orchard", "Nonprofit platform", "2023"),
			),
			section("flex-row gap-4 pt-3",
				el("flex flex-col flex-1", txt("text-muted-foreground", "Services"), twi.Text("Product strategy"), twi.Text("Interface design"), twi.Text("Design systems"), twi.Text("Front-end builds")),
				el("flex flex-col flex-1", txt("text-muted-foreground", "Recognition"), twi.Text("Type Directors, 2025"), twi.Text("Interaction Annual, 2024"), twi.Text("Best small studio, 2023")),
				el("flex flex-col flex-1", txt("text-muted-foreground", "Say hello"), twi.Text("hello@atelier-nine.example"), twi.Text("+00 0000 000 000"), txt("underline", "Book a call")),
			),
			section("pt-3", big("", "Let us talk."), txt("text-muted-foreground", "We take on four projects a year. The next opening is in spring.")),
			footer("atelier nine", "Design and engineering, est. 2019.",
				[]string{"Elsewhere", "Journal", "Newsletter", "Jobs"},
			),
		)
	}
}
