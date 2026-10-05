package main

import (
	"strings"
	"time"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func wave3b(rt *twi.Runtime, open string) func() twi.Node {
	accordion, repos, toaster := ui.NewAccordion(rt), ui.NewCollapsible(rt), ui.NewToaster(rt)
	inline, palette := ui.NewCommand(rt), ui.NewCommandDialog(rt)
	accordion.Collapsible, accordion.Value = true, []string{"item-1"}
	chosen := "nothing yet"
	pick := func(v string) {
		chosen = v
		rt.Invalidate()
	}
	inline.OnSelect, palette.OnSelect = pick, pick
	undo := ui.ToastAction{Label: "Undo", OnClick: func() { pick("Undo") }}
	event := func() { toaster.Show("Event has been created", "Sunday, December 03, 2023 at 9:00 AM", undo) }
	what, search, _ := strings.Cut(open, "=")
	switch what {
	case "collapsible":
		repos.Open = true
	case "toasts":
		toaster.Duration = time.Minute
		rt.Dispatch(func() {
			toaster.Error("Event has not been created", "", ui.ToastAction{})
			toaster.Success("Event has been created", "", ui.ToastAction{})
			event()
		})
	case "command":
		palette.Open = true
		palette.Search(search)
	}
	paragraphs := func(texts ...string) []twi.NodeOption {
		var out []twi.NodeOption
		for _, s := range texts {
			out = append(out, label(s))
		}
		return []twi.NodeOption{el("flex flex-col gap-1", out...)}
	}
	item := func(value, title string, texts ...string) twi.Node {
		return accordion.Item(value, accordion.Trigger(value, label(title)), accordion.Content(value, paragraphs(texts...)...))
	}
	repo := func(name string) twi.Node { return text("rounded-md border px-2 font-mono", name) }
	icon := func(glyph, name string, rest ...twi.NodeOption) []twi.NodeOption {
		return append([]twi.NodeOption{text("w-2 text-muted-foreground", glyph), label(name)}, rest...)
	}
	commands := func(c *ui.Command) twi.Node {
		shortcut := func(s string) twi.Node { return ui.CommandShortcut(label(s)) }
		return c.List(
			c.Group("Suggestions", c.Item("Calendar", icon("▦", "Calendar")...), c.Item("Search Emoji", icon("◡", "Search Emoji")...), c.Item("Calculator", icon("±", "Calculator")...)),
			c.Separator(),
			c.Group("Settings",
				c.Item("Profile", icon("◉", "Profile", shortcut("⌘P"))...),
				c.Item("Billing", icon("▭", "Billing", shortcut("⌘B"))...),
				c.Item("Settings", icon("✲", "Settings", shortcut("⌘S"))...),
			),
		)
	}
	outline, size := ui.Outline, ui.SizeDefault
	return func() twi.Node {
		return el("flex flex-row grow gap-4",
			el("flex flex-col gap-2 flex-1 min-w-40",
				section("Accordion", accordion.Node(
					item("item-1", "Product Information",
						"Our flagship product combines cutting-edge technology with sleek design. Built with premium materials, it offers unparalleled performance and reliability.",
						"Key features include advanced processing capabilities, and an intuitive user interface designed for both beginners and experts."),
					item("item-2", "Shipping Details",
						"We offer worldwide shipping through trusted courier partners. Standard delivery takes 3-5 business days, while express shipping ensures delivery within 1-2 business days.",
						"All orders are carefully packaged and fully insured. Track your shipment in real-time through our dedicated tracking portal."),
					item("item-3", "Return Policy",
						"We stand behind our products with a comprehensive 30-day return policy. If you're not completely satisfied, simply return the item in its original condition.",
						"Our hassle-free return process includes free return shipping and full refunds processed within 48 hours of receiving the returned item."),
				)),
				el("flex flex-row flex-wrap gap-x-4 gap-y-1",
					section("Collapsible", repos.Node(twi.Class("w-44 gap-1"),
						el("flex flex-row items-center justify-between gap-4 px-2", text("font-semibold", "@peduarte starred 3 repositories"), repos.Trigger(ui.Ghost, ui.SizeIcon, label("↕"))),
						repo("@radix-ui/primitives"),
						repos.Content(twi.Class("gap-1"), repo("@radix-ui/colors"), repo("@stitches/react")),
					)),
					section("Sonner",
						row(ui.Button(outline, size, twi.OnClick(func(*twi.Event) { event() }), label("Show Toast"))),
						row(
							ui.Button(outline, size, twi.OnClick(func(*twi.Event) { toaster.Success("Event has been created", "", ui.ToastAction{}) }), label("Success")),
							ui.Button(outline, size, twi.OnClick(func(*twi.Event) { toaster.Error("Event has not been created", "", ui.ToastAction{}) }), label("Error")),
						),
					),
				),
			),
			el("flex flex-col gap-2 w-56 shrink-0",
				section("Command", inline.Node(twi.Class("rounded-lg border shadow-md"), inline.Input("Type a command or search..."), commands(inline))),
				section("Command dialog", el("flex flex-row items-center gap-1 text-muted-foreground", label("Press"), ui.KbdGroup(ui.Kbd(label("Ctrl")), ui.Kbd(label("K"))))),
				text("text-muted-foreground", "Chosen: "+chosen),
			),
			palette.Node(palette.Input("Type a command or search..."), commands(palette.Command)),
			toaster.Node(),
		)
	}
}
