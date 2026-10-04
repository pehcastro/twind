package main

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func newProduct(rt *twi.Runtime) func() twi.Node {
	faq := ui.NewAccordion(rt)
	faq.Collapsible, faq.Value = true, []string{"trial"}
	question := func(value, q, a string) twi.Node {
		return faq.Item(value, faq.Trigger(value, twi.Text(q)), faq.Content(value, txt("text-muted-foreground", a)))
	}
	tier := func(name, price, blurb string, popular bool, perks ...string) twi.Node {
		class, cta, badge := "flex-1 py-1", ui.Outline, el("")
		if popular {
			class, cta, badge = "flex-1 py-1 border-primary shadow-md", ui.Default, ui.Badge(ui.Default, twi.Text("Most popular"))
		}
		list := []twi.NodeOption{twi.Class("flex flex-col gap-0")}
		for _, p := range perks {
			list = append(list, el("flex flex-row gap-1", txt("text-primary", "✓"), twi.Text(p)))
		}
		return ui.Card(twi.Class(class),
			ui.CardHeader(ui.CardTitle(twi.Text(name)), ui.CardDescription(twi.Text(blurb)), ui.CardAction(badge)),
			ui.CardContent(el("flex flex-col gap-1",
				el("flex flex-row items-end gap-1", txt("font-bold", price), txt("text-muted-foreground", "per seat / month")),
				el("", list...),
			)),
			ui.CardFooter(ui.Button(cta, ui.SizeDefault, twi.Class("grow"), twi.Text("Choose "+name))),
		)
	}
	row := func(name, owner, status string, v ui.Variant) twi.Node {
		return el("flex flex-row items-center gap-1 px-1 border-b",
			txt("grow truncate", name), txt("w-6 text-muted-foreground", owner), ui.Badge(v, twi.Text(status)))
	}
	return func() twi.Node {
		return el("flex flex-col shrink-0",
			section("flex-row items-center gap-2 py-0.5",
				txt("font-bold text-primary", "◧ Plainsheet"), links("text-muted-foreground", "Features", "Pricing", "Customers", "FAQ"), el("grow"),
				ui.Button(ui.Link, ui.SizeSM, twi.Text("Sign in")), ui.Button(ui.Default, ui.SizeSM, twi.Text("Try it free")),
			),
			section("flex-row items-center gap-4 pt-3 pb-2",
				el("flex flex-col w-42 shrink-0 gap-1",
					ui.Badge(ui.Secondary, twi.Text("Now with shared timelines")),
					txt("font-bold", "Plan the work. Skip the busywork."),
					txt("text-muted-foreground", "Plainsheet keeps projects, owners and deadlines on one calm page, so the team always knows what is next."),
					el("flex flex-row gap-1 pt-1",
						ui.Button(ui.Default, ui.SizeLG, twi.Class("py-0.5"), twi.Text("Start a free trial")),
						ui.Button(ui.Ghost, ui.SizeLG, twi.Class("py-0.5"), twi.Text("▶ Watch the tour")),
					),
					txt("text-muted-foreground pt-1", "★★★★★  Loved by 4,000 small teams"),
				),
				ui.Card(twi.Class("flex-1 gap-0 py-0 overflow-hidden shadow-lg"),
					el("flex flex-row items-center gap-1 px-2 py-0.5 border-b bg-muted",
						txt("text-muted-foreground", "● ● ●"), el("grow"), txt("px-2 rounded-md bg-background text-muted-foreground", "app.plainsheet.example"), el("grow")),
					el("flex flex-row",
						el("flex flex-col w-16 shrink-0 px-1 py-1 border-r bg-muted/40 whitespace-nowrap",
							txt("px-1 rounded-md bg-accent text-accent-foreground font-medium", "▤ Projects"),
							txt("px-1 text-muted-foreground", "◷ Timeline"), txt("px-1 text-muted-foreground", "◎ Goals"), txt("px-1 text-muted-foreground", "⚙ Settings")),
						el("flex flex-col grow px-1 py-1",
							el("flex flex-row items-center pb-0.5", txt("font-semibold grow", "Q3 launch"), ui.Button(ui.Default, ui.SizeXS, twi.Text("+ New"))),
							row("Pricing page copy", "Rosa", "Done", ui.Secondary),
							row("Onboarding emails", "Kenji", "In review", ui.Outline),
							row("Billing migration", "Amara", "Blocked", ui.Destructive),
							row("Launch checklist", "Theo", "Planned", ui.Outline),
						),
					),
				),
			),
			section("items-center gap-1 pt-3",
				txt("text-primary font-medium", "Pricing"),
				txt("font-bold", "Simple plans that grow with you"),
				txt("text-muted-foreground", "Every plan starts with a 14 day trial. No card needed."),
				el("flex flex-row w-full gap-2 pt-1",
					tier("Starter", "$0", "For trying it out", false, "Up to 3 projects", "2 collaborators", "Email support"),
					tier("Team", "$12", "For teams", true, "Unlimited projects", "Shared timelines", "Guest access", "Priority support"),
					tier("Business", "$29", "For whole companies", false, "Everything in Team", "Single sign-on", "Audit log", "Dedicated manager"),
				),
			),
			section("gap-1 pt-3 max-w-80",
				txt("font-bold text-center", "Frequently asked questions"),
				faq.Node(twi.Class("w-full"),
					question("trial", "How does the free trial work?", "You get every Team feature for 14 days. When it ends, pick a plan or stay on Starter."),
					question("switch", "Can I change plans later?", "Yes. Upgrades apply at once and downgrades at the end of the billing period."),
					question("import", "Can I import from spreadsheets?", "Drop in a CSV and Plainsheet maps the columns to projects, owners and dates."),
					question("data", "Where is my data stored?", "In the region you choose at sign-up, encrypted at rest and in transit."),
				),
			),
			footer("◧ Plainsheet", "Calm planning for small teams.",
				[]string{"Product", "Features", "Pricing", "Changelog"},
				[]string{"Help", "Guides", "FAQ", "Contact"},
				[]string{"Legal", "Privacy", "Terms", "Security"},
			),
		)
	}
}
