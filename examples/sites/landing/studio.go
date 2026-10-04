package main

import (
	"github.com/twind-dev/twind/twi"
)

func newStudio(k kit) page {
	work := func(n, client, cover, kind, year string) twi.Node {
		return el("group flex flex-row items-center gap-3 px-1 py-0.5 border-b transition-colors duration-200 hover:bg-foreground hover:text-background",
			txt("w-4 text-muted-foreground group-hover:text-background", n), txt("grow font-semibold", client),
			el("w-10 h-1 rounded-md shadow-sm "+cover),
			txt("w-26 text-muted-foreground group-hover:text-background", kind), txt("w-5 text-muted-foreground group-hover:text-background", year),
			txt("transition duration-200 group-hover:translate-x-1", "→"))
	}
	column := func(title string, lines ...string) twi.Node {
		rows := []twi.NodeOption{txt("text-muted-foreground pb-0.5", title)}
		for _, l := range lines {
			rows = append(rows, twi.Text(l))
		}
		return el("flex flex-col flex-1", rows...)
	}
	nav := func() twi.Node {
		return navbar("bg-background",
			txt("font-bold py-0.5", "atelier nine"),
			el("flex flex-row items-center gap-1 rounded-full px-2 py-0.5 bg-emerald-500/10 text-emerald-700", twi.Text("●"), twi.Text("Taking projects for spring")),
			el("grow"), k.links("", "Work", "Studio", "Contact"),
		)
	}
	body := func() twi.Node {
		return el("flex flex-col shrink-0 gap-3 pt-3",
			section("gap-1",
				txt("text-muted-foreground whitespace-pre", "W E   D E S I G N"),
				display("QUIET", 2, "text-foreground"),
				display("SOFTWARE.", 2, "text-muted-foreground"),
				el("flex flex-row items-end gap-4 pt-1",
					txt("text-muted-foreground max-w-48", "A small, independent studio for product design and front-end engineering. Twelve people, one floor, no account managers."),
					el("grow"),
					txt("rounded-full bg-foreground px-3 py-0.5 font-medium text-background shadow-md transition-colors duration-200 hover:bg-background hover:text-foreground hover:shadow-[0_0_0_1px_var(--color-foreground)]", "Start a project ↗"),
				),
			),
			section("", anchor("Work"),
				el("flex flex-row justify-between pb-0.5 border-b", txt("font-semibold", "Selected work"), txt("text-muted-foreground", "2023 to 2026")),
				work("01", "Harbor Ledger", "bg-linear-to-r from-primary-200 to-primary-700", "Fintech, design system", "2026"),
				work("02", "Fieldnote", "bg-linear-to-l from-primary-100 to-primary-500", "Research app, iOS", "2025"),
				work("03", "Morrow Transit", "bg-linear-to-r from-primary-400 to-primary-900", "Live timetables", "2025"),
				work("04", "Kiln & Co.", "bg-linear-to-l from-primary-300 to-primary-800", "Shop and brand", "2024"),
				work("05", "Open Orchard", "bg-linear-to-r from-primary-100 to-primary-400", "Nonprofit platform", "2023"),
			),
			section("flex-row gap-4", anchor("Studio"),
				column("Services", "Product strategy", "Interface design", "Design systems", "Front-end builds"),
				column("Recognition", "Type Directors, 2025", "Interaction Annual, 2024", "Best small studio, 2023"),
				column("Studio", "Twelve people", "One floor in Lisbon", "Est. 2019"),
			),
			section("gap-1 pt-1", anchor("Contact"),
				display("LET'S", 2, "text-foreground"),
				display("TALK.", 2, "text-foreground"),
				el("flex flex-row items-center gap-2 pt-1",
					txt("py-0.5 font-semibold underline transition-colors duration-200 hover:text-muted-foreground", "hello@atelier-nine.example"),
					el("rounded-full px-2 py-0.5 shadow-[0_0_0_1px_var(--color-border)] transition-colors duration-200 hover:bg-accent", twi.Focusable(), k.copy("hello@atelier-nine.example"), twi.Text("⧉ Copy")),
					txt("py-0.5 text-muted-foreground", "We take on four projects a year."),
				),
			),
			footer("atelier nine", "Design and engineering, est. 2019.",
				[]string{"Elsewhere", "Journal", "Newsletter", "Jobs"},
			),
		)
	}
	return page{nav: nav, body: body}
}
