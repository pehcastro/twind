package demo

import (
	"strings"

	"github.com/twind-dev/twind/internal/render"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen

func el(classes string, children ...render.Node) render.Node {
	return render.Node{Classes: strings.Fields(classes), Children: children}
}

func text(s string) render.Node { return render.Node{Text: s} }

func Hello() render.Node {
	return el("flex flex-col gap-2 p-4 bg-zinc-950 text-zinc-100",
		text("Hello Twind"),
		el("border rounded-lg p-2", text("Terminal DOM")),
	)
}

func Dialog() render.Node {
	return el("flex flex-col gap-1 p-2 bg-background text-foreground",
		text("Dashboard  Projects  Settings"),
		el("text-muted-foreground", text("Build 1432 passed in 2m 11s on main, all 214 tests green across three platforms")),
		text("The quick brown fox jumps over the lazy dog, 0123456789, and a wide 中文 pair"),
		el("border rounded-lg shadow-md p-1 bg-card", text("A card on the page, under the backdrop")),
		text("Line five of the page text"),
		text("Line six of the page text"),
		text("Line seven of the page text"),
		el("fixed inset-0 bg-black/50"),
		el("fixed inset-0 flex items-center justify-center",
			el("w-48 flex flex-col gap-1 p-2 border rounded-lg shadow-md bg-card text-card-foreground",
				text("Delete project?"),
				el("text-muted-foreground", text("This cannot be undone.")),
			),
		),
	)
}

func List(hover int) render.Node {
	rows := []render.Node{text("Recent")}
	for i, name := range []string{"Raft consensus", "Debug: show FPS", "Open settings", "Toggle theme", "Quit"} {
		row := "w-20 px-1 text-muted-foreground"
		if i == hover {
			row = "w-20 px-1 bg-accent text-accent-foreground"
		}
		rows = append(rows, el(row, text(name)))
	}
	return el("flex flex-col gap-1 p-2 bg-background text-foreground",
		text("Page text above the list"),
		el("w-24 flex flex-col p-1 border rounded-lg shadow-md bg-card text-card-foreground", rows...),
	)
}
