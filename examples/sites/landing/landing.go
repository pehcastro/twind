package main

import (
	"slices"
	"strings"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/theme"
	"github.com/twind-dev/twind/twi/ui"
)

type site uint8

const (
	platform site = iota
	product
	studio
)

func sites() []site { return []site{platform, product, studio} }

func (s site) String() string {
	return [...]string{platform: "Platform", product: "Product", studio: "Studio"}[s]
}

func (s site) address() string {
	return [...]string{platform: "quarry.example", product: "plainsheet.example", studio: "atelier-nine.example"}[s]
}

func (s site) theme() theme.Theme {
	look := [...]struct {
		name   string
		scheme theme.Scheme
	}{platform: {"twind", theme.Dark}, product: {"cloud", theme.Light}, studio: {"mono", theme.Light}}[s]
	for _, t := range theme.Builtin() {
		if t.Name == look.name && t.Scheme == look.scheme {
			return t
		}
	}
	panic("landing: no built-in theme " + look.name)
}

func named(name string) (site, bool) {
	i := slices.IndexFunc(sites(), func(s site) bool { return strings.EqualFold(s.String(), name) })
	return site(max(i, 0)), i >= 0
}

type page struct {
	nav, body func() twi.Node
}

type kit struct {
	rt    *twi.Runtime
	toast *ui.Toaster
}

func el(class string, children ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{twi.Class(class)}, children...)...)
}

func txt(class, s string) twi.Node { return el(class, twi.Text(s)) }

func section(class string, children ...twi.NodeOption) twi.Node {
	return el("flex flex-col shrink-0 w-full max-w-110 self-center px-4 "+class, children...)
}

const sectionKey = "section-"

func anchor(name string) twi.NodeOption { return twi.Key(sectionKey + name) }

func (k kit) links(class string, names ...string) twi.Node {
	out := []twi.NodeOption{}
	for _, n := range names {
		out = append(out, el("rounded-full px-2 py-0.5 transition-colors duration-200 hover:bg-accent hover:text-accent-foreground focus-visible:shadow-[0_0_0_1px_var(--color-ring)]",
			twi.Text(n), twi.Focusable(), twi.OnClick(func(*twi.Event) { k.rt.ScrollIntoView(sectionKey + n) })))
	}
	return el("flex flex-row items-center "+class, out...)
}

func (k kit) copy(value string) twi.NodeOption {
	return twi.OnClick(func(*twi.Event) {
		if k.rt.Copy(value) == nil {
			k.toast.Success("Copied to clipboard", value, ui.ToastAction{})
		}
	})
}

func lights() twi.Node {
	return el("flex flex-row whitespace-pre", txt("text-red-500", "● "), txt("text-amber-400", "● "), txt("text-emerald-500", "●"))
}

func navbar(class string, children ...twi.NodeOption) twi.Node {
	return el("flex flex-row shrink-0 justify-center border-b "+class, section("flex-row items-center gap-2 py-0.5", children...))
}

func footer(brand, note string, columns ...[]string) twi.Node {
	cols := []twi.NodeOption{el("flex flex-col grow gap-0.5", txt("font-semibold", brand), txt("text-muted-foreground", note))}
	for _, c := range columns {
		rows := []twi.NodeOption{txt("font-medium pb-0.5", c[0])}
		for _, l := range c[1:] {
			rows = append(rows, txt("text-muted-foreground transition-colors duration-200 hover:text-foreground", l))
		}
		cols = append(cols, el("flex flex-col w-18 shrink-0", rows...))
	}
	return el("flex flex-col shrink-0 border-t mt-3",
		section("flex-row gap-2 py-1.5", cols...),
		section("flex-row justify-between pt-0.5 pb-1 border-t text-muted-foreground", twi.Text("© 2026 "+brand+". An example written for Twind."), twi.Text("Made in a terminal")),
	)
}

func landing(rt *twi.Runtime, start site) func() twi.Node {
	tabs, current := ui.NewTabs(rt), start
	show := func(s site) {
		current, tabs.Value = s, s.String()
		rt.SetTheme(s.theme())
		rt.Invalidate()
	}
	tabs.OnChange = func(v string) {
		if s, ok := named(v); ok {
			show(s)
		}
	}
	show(start)
	k := kit{rt, ui.NewToaster(rt)}
	pages := [...]page{platform: newPlatform(k), product: newProduct(k), studio: newStudio(k)}
	keys := twi.OnKeyDown(func(e *twi.Event) {
		key := e.Key
		if key.Release || key.Key != input.KeyRune || key.Modifiers != 0 {
			return
		}
		switch {
		case key.Rune == 'q':
			rt.Quit()
		case key.Rune >= '1' && int(key.Rune-'1') < len(sites()):
			show(site(key.Rune - '1'))
		}
	})
	return func() twi.Node {
		var triggers []twi.NodeOption
		for _, s := range sites() {
			triggers = append(triggers, tabs.Trigger(s.String(), twi.Text(s.String())))
		}
		p := pages[current]
		return el("flex flex-col h-full bg-background text-foreground", keys,
			el("flex flex-row items-center shrink-0 gap-2 px-2 py-0.5 bg-muted/60",
				lights(),
				tabs.Node(tabs.List(triggers...)),
				el("flex flex-row grow min-w-0 justify-center",
					el("flex flex-row items-center w-50 min-w-0 rounded-full bg-background px-2 text-muted-foreground whitespace-nowrap shadow-[0_0_0_1px_var(--color-border)]",
						txt("text-chart-2 pr-1", "⊙"), twi.Text("https://"), txt("text-foreground", current.address()))),
				txt("shrink-0 whitespace-nowrap text-muted-foreground", "1-3 sites · q quit"),
			),
			el("flex flex-col grow min-h-0", twi.Key(current.String()),
				p.nav(),
				el("flex flex-col grow min-h-0 overflow-y-auto", twi.Focusable(), twi.AutoFocus(),
					el("flex flex-col shrink-0 animate-in fade-in-0 slide-in-from-bottom-2 duration-300", p.body())),
			),
			k.toast.Node(),
		)
	}
}

func display(s string, scale int, colours ...string) twi.Node {
	letters := []twi.NodeOption{}
	runes := []rune(s)
	for i, r := range runes {
		g := glyph(r)
		w, h, gap := len(g[0])*scale, len(g)*scale, scale
		if i == len(runes)-1 {
			gap = 0
		}
		lit := func(x, y int) int {
			if y < h && g[y/scale][x/scale] == '#' {
				return 1
			}
			return 0
		}
		rows := []twi.NodeOption{}
		for y := 0; y < h; y += 2 {
			var b strings.Builder
			for x := range w {
				b.WriteString([...]string{" ", "▀", "▄", "█"}[lit(x, y)+2*lit(x, y+1)])
			}
			rows = append(rows, twi.Text(b.String()+strings.Repeat(" ", gap)))
		}
		letters = append(letters, el("flex flex-col whitespace-pre "+colours[i*len(colours)/len(runes)], rows...))
	}
	return el("flex flex-row shrink-0 font-bold", letters...)
}

func glyph(r rune) [5]string {
	switch r {
	case ' ':
		return [5]string{"...", "...", "...", "...", "..."}
	case '.':
		return [5]string{".", ".", ".", ".", "#"}
	case '\'':
		return [5]string{"#", "#", ".", ".", "."}
	case 'A':
		return [5]string{".###.", "#...#", "#####", "#...#", "#...#"}
	case 'E':
		return [5]string{"#####", "#....", "####.", "#....", "#####"}
	case 'F':
		return [5]string{"#####", "#....", "####.", "#....", "#...."}
	case 'H':
		return [5]string{"#...#", "#...#", "#####", "#...#", "#...#"}
	case 'I':
		return [5]string{"###", ".#.", ".#.", ".#.", "###"}
	case 'K':
		return [5]string{"#...#", "#..#.", "###..", "#..#.", "#...#"}
	case 'L':
		return [5]string{"#....", "#....", "#....", "#....", "#####"}
	case 'N':
		return [5]string{"#...#", "##..#", "#.#.#", "#..##", "#...#"}
	case 'O':
		return [5]string{".###.", "#...#", "#...#", "#...#", ".###."}
	case 'P':
		return [5]string{"####.", "#...#", "####.", "#....", "#...."}
	case 'Q':
		return [5]string{".###.", "#...#", "#...#", "#..#.", ".##.#"}
	case 'R':
		return [5]string{"####.", "#...#", "####.", "#..#.", "#...#"}
	case 'S':
		return [5]string{".####", "#....", ".###.", "....#", "####."}
	case 'T':
		return [5]string{"#####", "..#..", "..#..", "..#..", "..#.."}
	case 'U':
		return [5]string{"#...#", "#...#", "#...#", "#...#", ".###."}
	case 'W':
		return [5]string{"#...#", "#...#", "#.#.#", "##.##", "#...#"}
	}
	panic("landing: no glyph for " + string(r))
}
