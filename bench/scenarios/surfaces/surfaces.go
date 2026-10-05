package surfaces

import (
	"strconv"
	"strings"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/theme"
)

//go:generate go run github.com/pehcastro/twind/internal/twirgen -o twir_gen.go

const (
	Columns  = 120
	Rows     = 40
	Wide     = 160
	Count    = '+'
	Hover    = 'h'
	Theme    = 't'
	nbsp     = string(rune(0xa0))
	listRows = 5
	hovered  = 2
)

type state struct {
	count int
	hover bool
}

func Themes() [2]theme.Theme {
	return [2]theme.Theme{theme.Default().WithScheme(theme.Dark), theme.Default().WithScheme(theme.Light)}
}

func App(rt *twi.Runtime) func() twi.Node {
	themes := Themes()
	st := twi.NewSignal(rt, state{})
	shown := 0
	onKey := func(k input.KeyEvent) {
		if k.Release || k.Key != input.KeyRune {
			return
		}
		s := st.Get()
		switch k.Rune {
		case Count:
			s.count++
		case Hover:
			s.hover = !s.hover
		case Theme:
			shown = 1 - shown
			rt.SetTheme(themes[shown])
			return
		default:
			return
		}
		st.Set(s)
	}
	return func() twi.Node {
		s := st.Get()
		tabs := []twi.Node{
			txt("shrink-0 rounded-full px-1 bg-emerald-400 text-emerald-950 font-bold", "twind"),
			txt("shrink h-1 overflow-hidden px-1 text-muted-foreground", "F:/localhost/ephem-sh/twind"),
			txt("shrink-0 rounded-full px-1 bg-primary text-primary-foreground font-bold", "surfaces"),
		}
		for _, name := range []string{"text", "layout", "counter"} {
			tabs = append(tabs, txt("shrink-0 px-1 text-muted-foreground", name))
		}
		tabs = append(tabs, el("grow"), txt("shrink-0 text-muted-foreground", "[t twind]"))
		rows := []twi.Node{txt("font-bold", "List")}
		for i := range listRows {
			class := "px-1"
			if s.hover && i == hovered {
				class = "px-1 bg-accent text-accent-foreground"
			}
			rows = append(rows, txt(class, "row "+strconv.Itoa(i+1)))
		}
		return twi.Element(
			twi.Class("flex flex-col h-full bg-background text-foreground"),
			twi.OnKey(onKey),
			el("flex flex-row items-center gap-2 px-2 pt-1", tabs...),
			el("grow relative overflow-hidden flex flex-col justify-center px-3",
				page(),
				el("flex flex-row gap-2 pt-1",
					el("flex-1 flex flex-col px-1 border rounded-lg shadow-md bg-card text-card-foreground", rows...),
					el("flex-1 flex flex-col px-1 border rounded-lg shadow-md bg-card text-card-foreground",
						txt("font-bold", "Counter"),
						txt("text-muted-foreground", "count "+strconv.Itoa(s.count)),
					),
				),
			),
			el("mx-2 flex flex-row items-center gap-1 px-1 border rounded-lg bg-card text-muted-foreground",
				txt("text-emerald-400 font-bold", "▌"),
				txt("grow h-1 overflow-hidden", "Ask twind to switch pages, pick a theme, or count"),
				txt("shrink-0 text-foreground", "tab"), txt("shrink-0", "pages"),
				txt("shrink-0 pl-1 text-foreground", "t"), txt("shrink-0", "theme"),
				txt("shrink-0 pl-1 text-foreground", "q"), txt("shrink-0", "quit"),
			),
			el("flex flex-row gap-3 px-3 text-muted-foreground",
				txt("text-emerald-400", "● fullscreen"),
				twi.Text("120x40"),
				twi.Text("truecolor"),
				twi.Text("twind"),
				el("grow"),
				twi.Text("surfaces"),
			),
		)
	}
}

func el(class string, children ...twi.Node) twi.Node {
	opts := []twi.NodeOption{twi.Class(class)}
	for _, c := range children {
		opts = append(opts, c)
	}
	return twi.Element(opts...)
}

func txt(class, s string) twi.Node { return el(class, twi.Text(s)) }

func card(title, body string) twi.Node {
	return el("flex-1 flex flex-col px-1 border rounded-lg shadow-md bg-card text-card-foreground",
		txt("font-bold", title),
		txt("text-muted-foreground", body),
	)
}

func page() twi.Node {
	letter := func(class string, rows ...string) twi.Node {
		return txt(class, strings.ReplaceAll(strings.Join(rows, "\n"), " ", nbsp))
	}
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
