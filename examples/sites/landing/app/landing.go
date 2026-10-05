package app

import (
	"slices"
	"strings"
	"time"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/theme"
	"github.com/pehcastro/twind/twi/ui"
)

type site uint8

const (
	platform site = iota
	product
	studio
	event
	store
	project
)

func sites() []site { return []site{platform, product, studio, event, store, project} }

func (s site) String() string {
	return [...]string{platform: "Platform", product: "Product", studio: "Studio", event: "Event", store: "Store", project: "Project"}[s]
}

func (s site) address() string {
	return [...]string{platform: "quarry.example", product: "plainsheet.example", studio: "atelier-nine.example",
		event: "fieldwork.example/26", store: "hearth.example/mug", project: "taskn.example"}[s]
}

func (s site) theme() theme.Theme {
	look := [...]struct {
		name   string
		scheme theme.Scheme
	}{platform: {"twind", theme.Dark}, product: {"cloud", theme.Light}, studio: {"mono", theme.Light},
		event: {"sukuna", theme.Dark}, store: {"dream", theme.Light}, project: {"dew", theme.Dark}}[s]
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
	nav, body, bar func() twi.Node
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

const (
	sectionKey    = "section-"
	eventStartsIn = 12*24*time.Hour + 4*time.Hour + 31*time.Minute + 9*time.Second
	typeDelay     = 60 * time.Millisecond
	lineDelay     = 350 * time.Millisecond
	loopPause     = 3 * time.Second
	demoPrompt    = "taskn run build --watch"
)

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
	pages := [...]page{platform: newPlatform(k), product: newProduct(k), studio: newStudio(k), event: newEvent(k), store: newStore(k), project: newProject(k)}
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
		view := []twi.NodeOption{twi.Key(current.String()),
			p.nav(),
			el("flex flex-col grow min-h-0 overflow-y-auto", twi.Focusable(), twi.AutoFocus(),
				el("flex flex-col shrink-0 animate-in fade-in-0 slide-in-from-bottom-2 duration-300", p.body())),
		}
		if p.bar != nil {
			view = append(view, p.bar())
		}
		return el("flex flex-col h-full bg-background text-foreground", keys,
			el("flex flex-row items-center shrink-0 gap-2 px-2 py-0.5 bg-muted/60",
				lights(),
				tabs.Node(tabs.List(triggers...)),
				el("flex flex-row grow min-w-0 justify-center",
					el("flex flex-row items-center w-50 min-w-0 overflow-hidden rounded-full bg-background px-2 text-muted-foreground whitespace-nowrap shadow-[0_0_0_1px_var(--color-border)]",
						txt("text-chart-2 pr-1", "⊙"), txt("text-foreground", current.address()))),
				txt("shrink-0 whitespace-nowrap text-muted-foreground", "1-6 · q quit"),
			),
			el("flex flex-col grow min-h-0", view...),
			k.toast.Node(),
		)
	}
}
