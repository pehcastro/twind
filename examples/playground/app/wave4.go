package app

import (
	"strconv"
	"unicode/utf8"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func spinnerPage(c controls) twi.Node {
	k := c.kit
	spin := func(i int, idle string) twi.Node {
		if k.spinning {
			return k.spinners[i].Node()
		}
		return twi.Text(idle)
	}
	says := func(busy, done string) twi.Node {
		return twi.Text(map[bool]string{true: busy, false: done}[k.spinning])
	}
	return show("Spinner", "turns while shown; hidden, it costs no frames. Enter or a click starts and stops it",
		c.uiButton("loading", ui.Outline, map[bool]string{true: "Stop loading", false: "Start loading"}[k.spinning], func(*state) { k.spinning = !k.spinning }),
		row(
			spin(0, "✓"),
			ui.Button(ui.Secondary, ui.SizeSM, spin(1, "✓"), says("Please wait", "Submitted")),
			ui.Badge(ui.Outline, spin(2, "✓"), says("Syncing", "Synced")),
			el("w-40 flex flex-row", k.lookup.Group(
				ui.InputGroupAddon(ui.InlineStart, spin(3, "⌕")),
				ui.InputGroupAddon(ui.InlineEnd, ui.InputGroupText(says("Searching...", "12 results"))),
			)),
		),
	)
}

func inputGroupPage(c controls) twi.Node {
	k := c.kit
	return show("Input group", "addons, text and buttons inside the field's border; Tab reaches the buttons",
		el("w-50 flex flex-col gap-1",
			k.query.Group(ui.InputGroupAddon(ui.InlineStart, ui.InputGroupText(twi.Text("⌕"))), ui.InputGroupAddon(ui.InlineEnd, ui.InputGroupText(twi.Text("12 results")))),
			k.url.Group(
				ui.InputGroupAddon(ui.InlineStart, ui.InputGroupText(twi.Text("https://"))),
				ui.InputGroupAddon(ui.InlineEnd, ui.InputGroupButton(c.clicked(func() { k.pressed = "Copy" }), twi.Text("Copy"))),
			),
			k.message.Group(ui.InputGroupAddon(ui.BlockEnd,
				ui.InputGroupText(twi.Text(strconv.Itoa(utf8.RuneCountInString(k.message.Value()))+"/280")),
				el("grow"),
				ui.InputGroupButton(twi.Class("bg-primary text-primary-foreground hover:bg-primary/90"), c.clicked(func() { k.pressed = "Send" }), twi.Text("Send")),
			)),
		),
		txt("text-muted-foreground", "pressed: "+k.pressed),
	)
}

func aspectRatioPage(c controls) twi.Node {
	k := c.kit
	return show("Aspect ratio", "the height follows the width, in the terminal's own cell pixels",
		row(
			el("flex flex-col gap-1 "+map[bool]string{true: "w-50", false: "w-30"}[k.wide],
				ui.AspectRatio(twi.Class("aspect-video rounded-lg bg-linear-to-br from-primary to-chart-2")),
				txt("text-muted-foreground", "aspect-video"),
			),
			el("flex flex-col gap-1 w-16",
				ui.AspectRatio(twi.Class("items-center justify-center rounded-lg bg-muted"), twi.Text("1:1")),
				txt("text-muted-foreground", "aspect-square"),
			),
		),
		c.uiButton("wider", ui.Outline, map[bool]string{true: "Narrower", false: "Wider"}[k.wide], func(*state) { k.wide = !k.wide }),
	)
}

func comboboxPage(c controls) twi.Node {
	f := c.kit.framework
	var items []twi.NodeOption
	for _, fw := range [][2]string{{"next", "Next.js"}, {"svelte", "SvelteKit"}, {"nuxt", "Nuxt.js"}, {"remix", "Remix"}, {"astro", "Astro"}} {
		items = append(items, f.Item(fw[0], fw[1]))
	}
	value := f.Value
	if value == "" {
		value = "none"
	}
	return show("Combobox", "type to filter; arrows, Enter and Escape, or the chevron and a click",
		txt("text-muted-foreground", "value: "+value),
		f.Node(twi.Class("w-40"), f.Input(), f.Content(items...)),
	)
}

func calendarPage(c controls) twi.Node {
	cal := c.kit.calendar
	selected := "none"
	if !cal.Selected.IsZero() {
		selected = cal.Selected.Format("Monday 2 January 2006")
	}
	return show("Calendar", "arrows by day, PageUp and PageDown by month, Home and End by week",
		row(cal.Node(twi.Class("rounded-lg border shadow-sm")), txt("w-30 text-muted-foreground", "selected: "+selected)),
	)
}

func navigationMenuPage(c controls) twi.Node {
	k := c.kit
	link := func(i *ui.NavigationMenuItem, href, title, about string) twi.Node {
		return i.Link(href, txt("font-medium", title), txt("text-muted-foreground", about))
	}
	followed := k.href
	if followed == "" {
		followed = "nothing yet"
	}
	return show("Navigation menu", "hover or arrows open a panel; Enter or a click follows a link",
		txt("text-muted-foreground", "followed: "+followed),
		k.nav.Node(k.nav.List(
			k.started.Node(k.started.Trigger(twi.Text("Getting started")), k.started.Content(twi.Class("w-40"),
				link(k.started, "/docs", "Introduction", "Re-usable components built with Tailwind."),
				link(k.started, "/docs/installation", "Installation", "How to install and structure your app."),
				link(k.started, "/docs/typography", "Typography", "Styles for headings, paragraphs and lists."),
			)),
			k.parts.Node(k.parts.Trigger(twi.Text("Components")), k.parts.Content(twi.Class("w-40"),
				link(k.parts, "/docs/alert-dialog", "Alert Dialog", "A modal that interrupts with important content."),
				link(k.parts, "/docs/hover-card", "Hover Card", "A preview of content behind a link."),
				link(k.parts, "/docs/progress", "Progress", "How far a task has come."),
			)),
		)),
		el("h-12"),
	)
}
