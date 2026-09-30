package main

import (
	"strconv"
	"strings"

	"github.com/twind-dev/twind/twi"
)

const nbsp = string(rune(0xa0))

func glyphs(rows ...string) string {
	return strings.ReplaceAll(strings.Join(rows, "\n"), " ", nbsp)
}

func heading(title, sub string) twi.Node {
	return el("flex flex-col items-center pb-1",
		txt("font-bold text-foreground", title),
		txt("text-muted-foreground", sub),
	)
}

func card(title, body string) twi.Node {
	return el("flex-1 flex flex-col px-1 border rounded-lg shadow-md bg-card text-card-foreground",
		txt("font-bold", title),
		txt("text-muted-foreground", body),
	)
}

func surfaces(state) twi.Node {
	letter := func(class string, rows ...string) twi.Node { return txt(class, glyphs(rows...)) }
	return el("flex flex-col gap-1",
		el("flex flex-row justify-center gap-1",
			letter("text-emerald-300", "▀█▀", " █ ", " ▀ "),
			letter("text-emerald-400", "█   █", "█ █ █", " ▀ ▀ "),
			letter("text-teal-400", "█", "█", "▀"),
			letter("text-cyan-400", "█▄  █", "█ ▀▄█", "▀   ▀"),
			letter("text-sky-400", "█▀▀▄", "█  █", "▀▀▀ "),
		),
		txt("text-center text-muted-foreground pb-1", "a DOM, real Tailwind and flexbox, drawn in terminal cells"),
		el("relative flex flex-row gap-2",
			card("Cards", "bg-card, a hairline border, rounded-lg, shadow-md"),
			card("Tokens", "background, card, border and primary come from the theme"),
			card("Compositing", "a popover and a chip float over this row"),
			txt("absolute -top-1 right-3 z-10 rounded-full px-1 bg-primary text-primary-foreground", "floating chip"),
			el("absolute top-3 right-6 z-20 w-30 flex flex-col px-1 border rounded-md shadow-lg bg-popover text-popover-foreground",
				txt("font-bold", "Popover"),
				txt("text-muted-foreground", "over the cards and the pills"),
			),
		),
		el("flex flex-row gap-2 pt-1",
			txt("rounded-full px-1 bg-secondary text-secondary-foreground", "secondary"),
			txt("rounded-full px-1 bg-accent text-accent-foreground", "accent"),
			txt("rounded-full px-1 bg-destructive text-white", "destructive"),
			txt("rounded-full px-1 bg-emerald-400 text-emerald-950", "emerald"),
		),
		el("flex flex-row gap-2 items-center",
			txt("text-muted-foreground", "build"),
			el("grow rounded-full bg-muted", el("w-2/3 rounded-full bg-linear-to-r from-emerald-400 to-sky-500", twi.Text(" "))),
			txt("text-muted-foreground", "66%"),
		),
	)
}

func textPage(state) twi.Node {
	panel := func(title string, body ...twi.Node) twi.Node {
		return el("flex-1 flex flex-col px-1 border rounded-lg bg-card text-card-foreground", append([]twi.Node{txt("font-bold", title)}, body...)...)
	}
	return el("flex flex-col gap-1",
		heading("Text", "alignment, wrapping, clipping, wide glyphs and hostile input"),
		el("flex flex-row gap-2",
			panel("Alignment", txt("text-left", "left"), txt("text-center", "centre"), txt("text-right", "right")),
			panel("Wrapping", twi.Text("A long sentence wraps at word boundaries inside its card, measured in cells from the width table, never in bytes.")),
		),
		el("flex flex-row gap-2",
			panel("Wide glyphs", twi.Text("漢字とかな 한국어 中文"), twi.Text("emoji 🚀 🌈 🎉")),
			panel("Clipped at 20 cells", el("w-20 overflow-hidden bg-muted", txt("w-60", "this line is longer than its box and is cut"))),
			panel("Hostile input, inert",
				txt("text-muted-foreground", "fed C1 CSI 2J, BEL, RLO:"),
				twi.Text("clear"+string(rune(0x9b))+"2J bell"+string(rune(0x07))+" bidi"+string(rune(0x202e))+"exe.txt"),
			),
		),
	)
}

func layoutPage(state) twi.Node {
	box := func(class, label string) twi.Node {
		return txt("px-1 rounded-md bg-secondary text-secondary-foreground "+class, label)
	}
	return el("flex flex-col gap-1",
		heading("Layout", "flexbox in integer cells"),
		txt("text-muted-foreground", "flex-row: two fixed, grow 1 and grow 2"),
		el("flex flex-row gap-1",
			box("w-12", "fixed 12"),
			box("grow bg-primary text-primary-foreground", "grow"),
			box("grow-2 bg-accent", "grow 2"),
			box("w-12", "fixed 12"),
		),
		el("flex flex-row gap-2 pt-1",
			el("w-24 flex flex-col px-1 border rounded-lg bg-card",
				txt("font-bold", "flex-col"),
				box("", "one"), box("", "two"), box("", "three"),
			),
			el("grow relative flex flex-col items-center justify-center px-1 border rounded-lg bg-card",
				txt("font-bold", "centred in its box"),
				txt("text-muted-foreground", "items-center justify-center"),
				txt("absolute -top-1 right-2 rounded-full px-1 bg-primary text-primary-foreground", "absolute chip"),
			),
			el("w-24 flex flex-col px-1 border rounded-lg bg-card",
				txt("font-bold", "overflow-hidden"),
				el("overflow-hidden", el("w-40 flex flex-row gap-1", box("", "one"), box("", "two"), box("", "three"), box("", "four"))),
				txt("text-muted-foreground", "a 40-cell row in 22"),
			),
		),
	)
}

func counterPage(s state) twi.Node {
	font := map[rune][3]string{
		'0': {"█▀█", "█ █", "▀▀▀"}, '1': {"▀█ ", " █ ", "▀▀▀"}, '2': {"▀▀█", "█▀▀", "▀▀▀"},
		'3': {"▀▀█", " ▀█", "▀▀▀"}, '4': {"█ █", "▀▀█", "  ▀"}, '5': {"█▀▀", "▀▀█", "▀▀▀"},
		'6': {"█▀▀", "█▀█", "▀▀▀"}, '7': {"▀▀█", "  █", "  ▀"}, '8': {"█▀█", "█▀█", "▀▀▀"},
		'9': {"█▀█", "▀▀█", "▀▀▀"}, '-': {"   ", "▀▀▀", "   "},
	}
	var rows [3]string
	for _, r := range strconv.Itoa(s.count) {
		for i := range rows {
			rows[i] += font[r][i] + " "
		}
	}
	return el("flex flex-col items-center gap-1",
		heading("Counter", "one signal; + and - set it and the runtime redraws"),
		el("w-30 flex flex-col items-center px-1 border rounded-lg shadow-lg bg-card",
			txt("font-bold text-primary", glyphs(rows[:]...)),
			txt("text-muted-foreground pt-1", "count"),
		),
		el("flex flex-row gap-3 pt-1",
			txt("rounded-full px-1 bg-secondary text-secondary-foreground", "- decrement"),
			txt("rounded-full px-1 bg-primary text-primary-foreground", "+ increment"),
		),
	)
}
