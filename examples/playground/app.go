package main

import (
	"path/filepath"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/text"
	"github.com/twind-dev/twind/twi/theme"
)

const pickerRows = 9

type state struct {
	page, count   int
	theme, cursor int
	picker        bool
}

type env struct {
	cwd, profile string
	size         func() string
}

type page struct {
	name string
	view func(state) twi.Node
}

func pages() []page {
	return []page{{"surfaces", surfaces}, {"text", textPage}, {"layout", layoutPage}, {"counter", counterPage}}
}

func themeName(t theme.Theme) string {
	return t.Name + map[theme.Scheme]string{theme.Light: "-light", theme.Dark: "-dark"}[t.Scheme]
}

func themeIndex(name string) int {
	for i, t := range theme.Builtin() {
		if themeName(t) == name {
			return i
		}
	}
	return -1
}

func el(class string, children ...twi.Node) twi.Node {
	opts := []twi.NodeOption{twi.Class(class)}
	for _, c := range children {
		opts = append(opts, c)
	}
	return twi.Element(opts...)
}

func txt(class, s string) twi.Node { return el(class, twi.Text(s)) }

func playground(rt *twi.Runtime, env env, start state) func() twi.Node {
	themes := theme.Builtin()
	all := pages()
	start.cursor = start.theme
	st := twi.NewSignal(rt, start)
	rt.SetTheme(themes[start.theme])
	onKey := func(k input.KeyEvent) {
		if k.Release {
			return
		}
		s := st.Get()
		wrap := func(i, n int) int { return (i%n + n) % n }
		r := k.Rune
		if k.Key != input.KeyRune {
			r = 0
		}
		switch {
		case r == 'q':
			rt.Quit()
			return
		case s.picker && k.Key == input.KeyArrowDown:
			s.cursor = wrap(s.cursor+1, len(themes))
		case s.picker && k.Key == input.KeyArrowUp:
			s.cursor = wrap(s.cursor-1, len(themes))
		case s.picker && k.Key == input.KeyEnter:
			s.theme, s.picker = s.cursor, false
			rt.SetTheme(themes[s.theme])
		case s.picker && k.Key == input.KeyEscape:
			s.picker = false
		case s.picker:
			return
		case r == 't':
			s.picker, s.cursor = true, s.theme
		case k.Key == input.KeyTab && k.Modifiers&input.ModShift != 0:
			s.page = wrap(s.page-1, len(all))
		case k.Key == input.KeyTab:
			s.page = wrap(s.page+1, len(all))
		case r >= '1' && int(r-'1') < len(all):
			s.page = int(r - '1')
		case r == '+' && all[s.page].name == "counter":
			s.count++
		case r == '-' && all[s.page].name == "counter":
			s.count--
		default:
			return
		}
		st.Set(s)
	}
	return func() twi.Node {
		s := st.Get()
		name := themeName(themes[s.theme])
		tabs := []twi.Node{
			txt("shrink-0 rounded-full px-1 bg-emerald-400 text-emerald-950 font-bold", "twind"),
			txt("shrink h-1 overflow-hidden px-1 text-muted-foreground", text.Truncate(filepath.ToSlash(env.cwd), 40)),
		}
		for i, p := range all {
			class := "shrink-0 px-1 text-muted-foreground"
			if i == s.page {
				class = "shrink-0 rounded-full px-1 bg-primary text-primary-foreground font-bold"
			}
			tabs = append(tabs, txt(class, p.name))
		}
		tabs = append(tabs, el("grow"), txt("shrink-0 text-muted-foreground", "[t "+name+"]"))
		root := []twi.NodeOption{
			twi.Class("flex flex-col h-full bg-background text-foreground"),
			twi.OnKey(onKey),
			el("flex flex-row items-center gap-2 px-2 pt-1", tabs...),
			el("grow relative overflow-hidden flex flex-col justify-center px-3", all[s.page].view(s)),
			el("mx-2 flex flex-row items-center gap-1 px-1 border rounded-lg bg-card text-muted-foreground",
				txt("text-emerald-400 font-bold", "▌"),
				txt("grow h-1 overflow-hidden", "Ask twind to switch pages, pick a theme, or count"),
				txt("shrink-0 text-foreground", "tab"), txt("shrink-0", "pages"),
				txt("shrink-0 pl-1 text-foreground", "t"), txt("shrink-0", "theme"),
				txt("shrink-0 pl-1 text-foreground", "q"), txt("shrink-0", "quit"),
			),
			el("flex flex-row gap-3 px-3 text-muted-foreground",
				txt("text-emerald-400", "● fullscreen"),
				twi.Text(env.size()),
				twi.Text(env.profile),
				twi.Text(name),
				el("grow"),
				twi.Text(all[s.page].name),
			),
		}
		if s.picker {
			root = append(root, picker(themes, s))
		}
		return twi.Element(root...)
	}
}

func picker(themes []theme.Theme, s state) twi.Node {
	first := min(max(s.cursor-pickerRows/2, 0), len(themes)-pickerRows)
	rows := []twi.Node{txt("px-1 font-bold", "Theme"), txt("px-1 pb-1 text-muted-foreground", "↑ ↓ move, Enter applies, Esc closes")}
	for i := first; i < first+pickerRows; i++ {
		class, mark := "px-1", "  "
		if i == s.cursor {
			class = "px-1 bg-accent text-accent-foreground font-bold"
		}
		if i == s.theme {
			mark = "● "
		}
		rows = append(rows, txt(class, mark+themeName(themes[i])))
	}
	return el("fixed inset-0 z-10 bg-black/50 flex items-center justify-center",
		el("w-44 flex flex-col px-1 border rounded-lg shadow-lg bg-popover text-popover-foreground", rows...),
	)
}
