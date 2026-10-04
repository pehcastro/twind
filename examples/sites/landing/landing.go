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

func el(class string, children ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{twi.Class(class)}, children...)...)
}

func txt(class, s string) twi.Node { return el(class, twi.Text(s)) }

func section(class string, children ...twi.NodeOption) twi.Node {
	return el("flex flex-col shrink-0 w-full max-w-110 self-center px-4 "+class, children...)
}

func links(class string, names ...string) twi.Node {
	out := []twi.NodeOption{}
	for _, n := range names {
		out = append(out, txt("px-1 rounded-md hover:bg-accent hover:text-accent-foreground", n))
	}
	return el("flex flex-row items-center gap-1 "+class, out...)
}

func footer(brand, note string, columns ...[]string) twi.Node {
	cols := []twi.NodeOption{el("flex flex-col grow gap-0.5", txt("font-semibold", brand), txt("text-muted-foreground", note))}
	for _, c := range columns {
		rows := []twi.NodeOption{txt("font-medium", c[0])}
		for _, l := range c[1:] {
			rows = append(rows, txt("text-muted-foreground hover:text-foreground", l))
		}
		cols = append(cols, el("flex flex-col w-18 shrink-0", rows...))
	}
	return el("flex flex-col shrink-0 border-t mt-2",
		section("flex-row gap-2 py-1", cols...),
		section("flex-row justify-between py-0.5 border-t text-muted-foreground", twi.Text("© 2026 "+brand+". An example written for Twind."), twi.Text("Made in a terminal")),
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
	pages := [...]func() twi.Node{platform: newPlatform(), product: newProduct(rt), studio: newStudio()}
	keys := twi.OnKeyDown(func(e *twi.Event) {
		k := e.Key
		if k.Release || k.Key != input.KeyRune || k.Modifiers != 0 {
			return
		}
		switch {
		case k.Rune == 'q':
			rt.Quit()
		case k.Rune >= '1' && int(k.Rune-'1') < len(sites()):
			show(site(k.Rune - '1'))
		}
	})
	hint := func(key, s string) twi.Node {
		return el("flex flex-row items-center gap-1", ui.Kbd(twi.Text(key)), txt("text-muted-foreground", s))
	}
	return func() twi.Node {
		var triggers []twi.NodeOption
		for _, s := range sites() {
			triggers = append(triggers, tabs.Trigger(s.String(), twi.Text(s.String())))
		}
		return el("flex flex-col h-full bg-background text-foreground", keys,
			el("flex flex-row items-center shrink-0 gap-2 px-2 py-0.5 border-b bg-muted/40",
				txt("font-semibold", "Twind sites"), txt("text-muted-foreground", "three landing pages"), el("grow"),
				tabs.Node(tabs.List(triggers...)), el("grow"),
				hint("1-3", "style"), hint("PgDn", "scroll"), hint("q", "quit"),
			),
			el("flex flex-col grow min-h-0 overflow-y-auto", twi.Key(current.String()), twi.Focusable(), twi.AutoFocus(), pages[current]()),
		)
	}
}
