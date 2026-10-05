package app

import (
	"strconv"
	"strings"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
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

func card(class, title, body string, rest ...twi.Node) twi.Node {
	header := []twi.NodeOption{ui.CardTitle(twi.Text(title))}
	if body != "" {
		header = append(header, ui.CardDescription(twi.Text(body)))
	}
	children := []twi.NodeOption{twi.Class(class), ui.CardHeader(header...)}
	if len(rest) > 0 {
		children = append(children, ui.CardContent(el("flex flex-col gap-1", rest...)))
	}
	return ui.Card(children...)
}

func surfaces(controls) twi.Node {
	letter := func(class string, rows ...string) twi.Node { return txt(class, glyphs(rows...)) }
	return el("flex flex-col gap-1",
		el("flex flex-row justify-center gap-1",
			letter("text-primary-800 dark:text-primary-600", "▀█▀", " █ ", " ▀ "),
			letter("text-primary-700 dark:text-primary-500", "█   █", "█ █ █", " ▀ ▀ "),
			letter("text-primary-600 dark:text-primary-400", "█", "█", "▀"),
			letter("text-primary-500 dark:text-primary-300", "█▄  █", "█ ▀▄█", "▀   ▀"),
			letter("text-primary-400 dark:text-primary-200", "█▀▀▄", "█  █", "▀▀▀ "),
		),
		txt("text-center text-muted-foreground pb-1", "a DOM, real Tailwind and flexbox, drawn in terminal cells"),
		el("relative flex flex-row gap-2",
			card("flex-1", "Cards", "ui.Card: bg-card, a hairline border, rounded-xl, shadow-sm"),
			card("flex-1", "Tokens", "background, card, border and primary come from the theme"),
			card("flex-1", "Compositing", "a popover and a chip float over this row"),
			txt("absolute -top-1 right-3 z-10 rounded-full px-1 bg-primary text-primary-foreground", "floating chip"),
			el("absolute top-3 right-6 z-20 w-30 flex flex-col px-1 border rounded-md shadow-lg bg-popover text-popover-foreground",
				txt("font-bold", "Popover"),
				txt("text-muted-foreground", "over the cards and the pills"),
			),
		),
		el("flex flex-row gap-2 pt-1",
			ui.Badge(ui.BadgeDefault, twi.Text("default")),
			ui.Badge(ui.BadgeSecondary, twi.Text("secondary")),
			ui.Badge(ui.BadgeDestructive, twi.Text("destructive")),
			ui.Badge(ui.BadgeOutline, twi.Text("outline")),
			ui.Badge(ui.BadgeDefault, twi.Class("bg-primary-200 text-primary-900"), twi.Text("primary-200")),
		),
		el("flex flex-row gap-2 items-center",
			txt("text-muted-foreground", "build"),
			el("grow rounded-full bg-muted", el("w-2/3 rounded-full bg-linear-to-r from-primary-700 via-primary to-primary-300", twi.Text(" "))),
			txt("text-muted-foreground", "66%"),
		),
	)
}

func textPage(controls) twi.Node {
	panel := func(title string, body ...twi.Node) twi.Node { return card("flex-1", title, "", body...) }
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

func layoutPage(controls) twi.Node {
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

func listPage(c controls) twi.Node {
	words := []string{"apricot", "blueberry", "cherry", "damson", "elderberry", "fig", "gooseberry", "huckleberry"}
	rows := make([]twi.Node, listRows)
	for i := range rows {
		name := "row " + strconv.Itoa(i+1)
		class := "px-1"
		if c.state.focus == name {
			class = "px-1 bg-accent text-accent-foreground"
		}
		rows[i] = twi.Element(append(c.focusable(name), twi.Class(class), twi.Text(name+" "+words[i%len(words)]))...)
	}
	return el("flex flex-col gap-1",
		heading("List", "the wheel, PageUp, PageDown, Home, End and Tab scroll the card"),
		el("flex flex-row gap-2",
			el("w-44 h-12 shrink-0 flex flex-col border rounded-lg bg-card text-card-foreground overflow-y-auto", rows...),
			el("flex-1 flex flex-col px-1 border rounded-lg bg-card text-card-foreground",
				txt("font-bold", "Fixed beside the list"),
				twi.Text("This card does not scroll. A wheel over it does nothing; a wheel over the list moves only the list."),
				txt("text-muted-foreground pt-1", "Three rows per notch, a page per PageDown, and the focused row is always brought into view."),
			),
		),
	)
}

func counterPage(c controls) twi.Node {
	font := map[rune][3]string{
		'0': {"█▀█", "█ █", "▀▀▀"}, '1': {"▀█ ", " █ ", "▀▀▀"}, '2': {"▀▀█", "█▀▀", "▀▀▀"},
		'3': {"▀▀█", " ▀█", "▀▀▀"}, '4': {"█ █", "▀▀█", "  ▀"}, '5': {"█▀▀", "▀▀█", "▀▀▀"},
		'6': {"█▀▀", "█▀█", "▀▀▀"}, '7': {"▀▀█", "  █", "  ▀"}, '8': {"█▀█", "█▀█", "▀▀▀"},
		'9': {"█▀█", "▀▀█", "▀▀▀"},
	}
	var rows [3]string
	var disabled []twi.NodeOption
	if c.state.count == 0 {
		disabled = []twi.NodeOption{twi.Disabled()}
	}
	for _, r := range strconv.Itoa(c.state.count) {
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
			c.uiButton("decrement", ui.ButtonSecondary, "- decrement", func(s *state) { s.count = max(s.count-1, 0) }, disabled...),
			c.uiButton("increment", ui.ButtonDefault, "+ increment", func(s *state) { s.count++ }),
		),
	)
}

func motionPage(c controls) twi.Node {
	hover := "flex-1 transition-colors hover:bg-accent hover:text-accent-foreground"
	run := func(cmd string) func(*state) { return func(s *state) { c.command(s, cmd) } }
	loaded := []twi.Node{txt("text-muted-foreground", "loaded, still")}
	if c.state.loading {
		loaded = []twi.Node{ui.Skeleton(twi.Class("h-1 w-full animate-pulse")), ui.Skeleton(twi.Class("h-1 w-2/3 animate-pulse"))}
	}
	return el("flex flex-col gap-1",
		heading("Motion", "frames only while something moves; at rest the runtime sleeps"),
		el("flex flex-row gap-2",
			card(hover, "Hover", "eases to accent in 150 ms"),
			ui.Card(twi.Class("flex-1"), ui.CardHeader(ui.CardTitle(twi.Text("Spring")), ui.CardDescription(twi.Text("press o")), ui.CardAction(c.springMenu()))),
			card(hover, "Reorder", "r: last row to the top", el("flex flex-row items-center gap-2", c.flipList(), c.uiButton("reorder", ui.ButtonOutline, "Rotate", run("reorder")))),
		),
		el("flex flex-row gap-2",
			card(hover, "Dialog", "zoom-in-95, 200 ms", c.uiButton("open dialog", ui.ButtonOutline, "Open dialog", run("dialog"))),
			card(hover, "Sheet", "slides in, 500 ms", c.uiButton("open sheet", ui.ButtonOutline, "Open sheet", run("sheet"))),
			card(hover, "Loading", "l: pulse for 2 s", loaded...),
			card(hover, "Toast", "sonner's stack", c.uiButton("show toast", ui.ButtonOutline, "Show toast", run("toast"))),
		),
	)
}

func selectionPage(controls) twi.Node {
	return el("flex flex-col gap-1",
		heading("Selection", "drag inside a card: the highlight stays in it, and ctrl+c copies with OSC 52"),
		el("flex flex-row gap-2",
			card("flex-1", "A rendering target", "", twi.Text("Application code thinks in nodes, classes, events and focus. Only the renderer knows about cells and escape sequences.")),
			card("flex-1", "Text is data", "", twi.Text("Nothing that takes a string writes it to the terminal unsanitised. A control byte is shown inert, never obeyed.")),
			card("flex-1", "Width is never len", "", twi.Text("Width comes from a versioned table, so 漢字 and 한국어 select and copy whole, never half a cell."), txt("select-none text-muted-foreground", "select-none: never copied")),
		),
	)
}
