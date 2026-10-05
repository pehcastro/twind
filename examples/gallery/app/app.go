package app

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/icon"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/ui"
)

const focusRing = "focus-visible:shadow-[0_0_0_1px_var(--color-ring),0_0_0_3px_color-mix(in_oklab,var(--color-ring)_50%,transparent)]"

type page uint8

const (
	dashboard page = iota
	forms
	overlays
	settings
)

func pages() []page { return []page{dashboard, forms, overlays, settings} }

func (p page) String() string {
	return [...]string{dashboard: "Dashboard", forms: "Forms", overlays: "Overlays", settings: "Settings"}[p]
}

func el(class string, children ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{twi.Class(class)}, children...)...)
}

func txt(class, s string) twi.Node { return el(class, twi.Text(s)) }

func onKeys(rt *twi.Runtime, handle func(input.KeyEvent) bool) twi.NodeOption {
	return twi.OnKeyDown(func(e *twi.Event) {
		if !e.Key.Release && handle(e.Key) {
			e.PreventDefault()
			e.StopPropagation()
			rt.Invalidate()
		}
	})
}

func clicked(rt *twi.Runtime, act func()) twi.NodeOption {
	return twi.OnClick(func(*twi.Event) {
		act()
		rt.Invalidate()
	})
}

func pressed(k input.KeyEvent) bool {
	return k.Key == input.KeyEnter || k.Key == input.KeyRune && k.Rune == ' '
}

func gallery(rt *twi.Runtime, s start) func() twi.Node {
	current, cursor, navFocused := s.page, s.page, false
	overlayView, modal := newOverlays(rt, s.open)
	toaster, dark := ui.NewToaster(rt), ui.NewSwitch(rt)
	views := [...]func() twi.Node{dashboard: newDashboard(rt), forms: newForms(rt), overlays: overlayView, settings: newSettings(rt, s.theme, toaster, dark)}
	flip := func() {
		dark.Checked = !dark.Checked
		dark.OnChange(dark.Checked)
		rt.Invalidate()
	}
	keys := twi.OnKeyDown(func(e *twi.Event) {
		k := e.Key
		if k.Release || k.Key != input.KeyRune || k.Modifiers != 0 || modal() {
			return
		}
		switch k.Rune {
		case 'q':
			rt.Quit()
		case 'm':
			flip()
		}
	})
	focus := func(on bool) func(*twi.Event) {
		return func(*twi.Event) {
			navFocused = on
			rt.Invalidate()
		}
	}
	navKeys := onKeys(rt, func(k input.KeyEvent) bool {
		switch {
		case k.Key == input.KeyArrowDown:
			cursor = min(cursor+1, settings)
		case k.Key == input.KeyArrowUp:
			cursor = max(cursor, 1) - 1
		case k.Key == input.KeyHome:
			cursor = dashboard
		case k.Key == input.KeyEnd:
			cursor = settings
		case pressed(k):
			current = cursor
		default:
			return false
		}
		return true
	})
	open := func(p page) twi.NodeOption { return clicked(rt, func() { current, cursor = p, p }) }
	doc := func(glyph, s string) twi.Node {
		return el("flex flex-row items-center gap-1 px-1 rounded-md text-muted-foreground hover:bg-accent hover:text-accent-foreground",
			clicked(rt, func() { toaster.Show(s+" opens in the full app", "The gallery shows four pages.", ui.ToastAction{}) }),
			txt("w-1", glyph), twi.Text(s))
	}
	hint := func(key, s string) twi.Node {
		return el("flex flex-row items-center gap-1", ui.Kbd(twi.Text(key)), txt("text-muted-foreground", s))
	}
	return func() twi.Node {
		nav := []twi.NodeOption{twi.Focusable(), twi.AutoFocus(), twi.OnFocus(focus(true)), twi.OnBlur(focus(false)), navKeys}
		for _, p := range pages() {
			class, label := "flex flex-row items-center gap-1 px-1 rounded-md hover:bg-accent hover:text-accent-foreground", ""
			if p == current {
				class += " bg-accent text-accent-foreground font-medium"
			}
			if navFocused && p == cursor {
				label = "underline"
			}
			nav = append(nav, el(class, open(p), txt("w-1 text-muted-foreground", [...]string{"▦", "≡", "▣", "◎"}[p]), txt(label, p.String())))
		}
		return el("flex flex-row h-full bg-background text-foreground", keys,
			el("flex flex-col w-20 shrink-0 border-r bg-muted/40 lg:w-26",
				el("flex flex-row items-center gap-1 px-2 pt-1 border-b",
					txt("px-1 rounded-md bg-primary text-primary-foreground font-bold", "◆"), txt("font-semibold", "Acme Inc.")),
				el("flex flex-col grow gap-1 px-1 pt-1",
					txt("px-1 text-muted-foreground", "Platform"),
					el("flex flex-col", nav...),
					txt("px-1 pt-1 text-muted-foreground", "Documents"),
					el("flex flex-col", doc("◫", "Data Library"), doc("◩", "Reports"), doc("◪", "Assistant")),
					el("grow"),
					el("flex flex-col gap-1 px-1", hint("↑↓", "move"), hint("⏎", "open"), hint("tab", "focus"), hint("m", "scheme"), hint("q", "quit")),
				),
				el("flex flex-row items-center gap-1 px-2 py-1 border-t",
					ui.Avatar(ui.AvatarSizeSM, ui.AvatarFallback(twi.Text("CN"))),
					el("flex flex-col min-w-0", txt("font-medium", "shadcn"), txt("text-muted-foreground truncate", "m@example.com")),
				),
			),
			el("flex flex-col grow min-w-0",
				el("flex flex-row items-center shrink-0 gap-2 px-2 pt-1 border-b lg:px-4",
					ui.Breadcrumb(ui.BreadcrumbList(
						ui.BreadcrumbItem(ui.BreadcrumbLink(open(dashboard), twi.Text("Acme"))), ui.BreadcrumbSeparator(),
						ui.BreadcrumbItem(ui.BreadcrumbPage(twi.Text(current.String()))),
					)),
					el("grow"),
					txt("text-muted-foreground", "Twind gallery"),
					ui.Badge(ui.BadgeOutline, twi.Text("v0.7")),
					el("px-1 rounded-full text-muted-foreground hover:bg-accent hover:text-accent-foreground", clicked(rt, flip),
						twi.Text(string(map[bool]icon.Name{false: icon.Sun, true: icon.Moon}[dark.Checked].Glyph()))),
				),
				el("flex flex-col grow min-h-0 px-2 py-1 lg:px-4", views[current]()),
			),
			toaster.Node(),
		)
	}
}
