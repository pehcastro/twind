package app

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func newProduct(k kit) page {
	faq := ui.NewAccordion(k.rt)
	faq.Collapsible, faq.Value = true, []string{"trial"}
	billing := ui.NewTabs(k.rt)
	billing.Value = "Monthly"
	question := func(value, q, a string) twi.Node {
		return faq.Item(value, faq.Trigger(value, twi.Text(q)), faq.Content(value, txt("text-muted-foreground", a)))
	}
	tier := func(name, monthly, yearly, blurb string, popular bool, perks ...string) twi.Node {
		class, cta := "flex-1 py-1 transition duration-200 hover:-translate-y-0.5 hover:shadow-lg", ui.ButtonOutline
		badge := el("")
		if popular {
			class, cta = "flex-1 py-1 border-primary shadow-xl -translate-y-0.5 bg-linear-to-b from-primary/10 to-card", ui.ButtonDefault
			badge = ui.Badge(ui.BadgeDefault, twi.Class("py-0.5"), twi.Text("Most popular"))
		}
		price := monthly
		if billing.Value == "Yearly" {
			price = yearly
		}
		list := []twi.NodeOption{}
		for _, p := range perks {
			list = append(list, el("flex flex-row gap-1", txt("text-primary", "✓"), twi.Text(p)))
		}
		return ui.Card(twi.Class(class),
			ui.CardHeader(ui.CardTitle(twi.Text(name)), ui.CardDescription(twi.Text(blurb)), ui.CardAction(badge)),
			ui.CardContent(el("flex flex-col gap-1",
				el("flex flex-row items-end gap-1", txt("font-bold", price), txt("text-muted-foreground", "per seat / month")),
				el("flex flex-col", list...),
			)),
			ui.CardFooter(twi.Class("pt-0.5"), ui.Button(cta, ui.ButtonSizeDefault, twi.Class("grow py-0.5"), twi.Text("Choose "+name))),
		)
	}
	task := func(name, owner, pad, bar, span string) twi.Node {
		return el("flex flex-row items-center gap-1",
			txt("w-10 shrink-0 truncate", name),
			el("flex flex-row grow min-w-0 whitespace-nowrap", el("shrink-0 "+pad), txt("shrink-0 rounded-full px-1 text-white shadow-sm "+bar+" "+span, owner)))
	}
	avatar := func(class, initials string) twi.Node {
		return txt("rounded-full px-1 font-medium text-white shadow-[0_0_0_1px_var(--color-background)] "+class, initials)
	}
	feature := func(glyph, title, body string) twi.Node {
		return el("flex flex-col flex-1 gap-0.5",
			el("flex flex-row", txt("rounded-lg bg-primary/15 px-1 py-0.5 font-bold text-primary", glyph)),
			txt("font-semibold pt-0.5", title), txt("text-muted-foreground", body))
	}
	nav := func() twi.Node {
		return navbar("bg-background shadow-sm",
			el("flex flex-row items-center gap-1", txt("flex flex-row w-4 justify-center rounded-md bg-linear-to-br from-primary-400 to-primary-700 py-0.5 font-bold text-white", "◧"), txt("font-bold py-0.5", "Plainsheet")),
			k.links("text-muted-foreground", "Features", "Pricing", "FAQ"), el("grow"),
			ui.Button(ui.ButtonGhost, ui.ButtonSizeSM, twi.Class("rounded-lg py-0.5"), twi.Text("Sign in")),
			ui.Button(ui.ButtonDefault, ui.ButtonSizeSM, twi.Class("rounded-lg py-0.5 shadow-md"), twi.Text("Try it free →")),
		)
	}
	body := func() twi.Node {
		return el("flex flex-col shrink-0 gap-3 pt-2",
			section("flex-row items-center gap-3",
				el("flex flex-col w-46 shrink-0 gap-1",
					el("flex flex-row", txt("rounded-full bg-primary/10 px-2 py-0.5 text-primary shadow-[0_0_0_1px_var(--color-primary)]", "✦ Now with shared timelines →")),
					el("flex flex-col gap-0.5 pt-1", display("PLAN THE", 1, "text-foreground"), el("flex flex-row gap-1", display("WORK", 1, "text-foreground"), display(".", 1, "text-primary"))),
					el("flex flex-row", txt("rounded-md bg-linear-to-r from-primary/30 to-primary/5 px-1 py-0.5 -ml-1 font-bold", "Skip the busywork.")),
					txt("text-muted-foreground", "Plainsheet keeps projects, owners and deadlines on one calm page, so the team always knows what is next."),
					el("flex flex-row items-center gap-1 pt-1",
						ui.Button(ui.ButtonDefault, ui.ButtonSizeLG, twi.Class("rounded-lg py-0.5 shadow-lg"), twi.Text("Start a free trial")),
						ui.Button(ui.ButtonGhost, ui.ButtonSizeLG, twi.Class("rounded-lg py-0.5"), twi.Text("▶ Watch the tour")),
					),
					el("flex flex-row items-center gap-1 pt-1",
						el("flex flex-row", avatar("bg-primary-400", "RK"), avatar("bg-primary-500", "AM"), avatar("bg-primary-700", "TS"), avatar("bg-primary-900", "+9")),
						el("flex flex-col", txt("text-amber-500", "★★★★★"), txt("text-muted-foreground", "Loved by 4,000 small teams"))),
				),
				el("flex flex-col flex-1 min-w-0 rounded-2xl bg-linear-to-br from-primary-200 via-primary-100 to-primary-300 p-2 shadow-lg",
					el("flex flex-col overflow-hidden rounded-xl border bg-card shadow-xl",
						el("flex flex-row items-center gap-1 px-2 py-0.5 border-b bg-muted",
							lights(), el("grow"),
							txt("px-2 rounded-md bg-background text-muted-foreground", "app.plainsheet.example"), el("grow")),
						el("flex flex-row",
							el("flex flex-col w-15 shrink-0 gap-0.5 px-1 py-1 border-r bg-muted/40 whitespace-nowrap",
								txt("px-1 rounded-md bg-accent text-accent-foreground font-medium", "◷ Timeline"),
								txt("px-1 text-muted-foreground", "▤ Projects"), txt("px-1 text-muted-foreground", "◎ Goals"), txt("px-1 text-muted-foreground", "≡ Settings")),
							el("flex flex-col grow min-w-0 gap-0.5 px-2 py-1",
								el("flex flex-row items-center", txt("font-semibold grow", "Q3 launch"), txt("rounded-md bg-primary px-1 text-primary-foreground", "+ New")),
								el("flex flex-row gap-1 text-muted-foreground", el("w-10 shrink-0"), txt("w-6", "Jul"), txt("w-6", "Aug"), txt("w-6", "Sep")),
								task("Copy", "Rosa", "w-0", "bg-linear-to-r from-primary-400 to-primary-600", "w-7"),
								task("Onboarding", "Kenji", "w-3", "bg-linear-to-r from-primary-500 to-primary-700", "w-9"),
								task("Billing", "Amara", "w-7", "bg-linear-to-r from-primary-600 to-primary-800", "w-7"),
								task("Launch", "Theo", "w-11", "bg-linear-to-r from-primary-700 to-primary-900", "w-6"),
							),
						),
					),
				),
			),
			section("items-center gap-1",
				txt("text-muted-foreground", "Teams that plan with Plainsheet"),
				el("flex flex-row justify-between w-full px-4 font-semibold text-muted-foreground",
					twi.Text("Fernway"), twi.Text("Copperleaf"), twi.Text("Tidewell"), twi.Text("Marlow & Pike"), twi.Text("Sundial")),
			),
			section("gap-1", anchor("Features"),
				txt("text-primary font-medium", "Features"),
				txt("font-bold", "Everything in one calm page."),
				el("flex flex-row gap-4 pt-1",
					feature("◷", "Timelines that move", "Drag a bar and every date after it follows."),
					feature("◎", "Goals with owners", "Each goal has one name next to it, never a committee."),
					feature("⇄", "Imports in a minute", "Bring a spreadsheet; columns map themselves."),
				),
			),
			section("items-center gap-1", anchor("Pricing"),
				txt("text-primary font-medium", "Pricing"),
				txt("font-bold", "Simple plans that grow with you"),
				txt("text-muted-foreground", "Every plan starts with a 14 day trial. No card needed."),
				billing.Node(billing.List(billing.Trigger("Monthly", twi.Text("Monthly")), billing.Trigger("Yearly", twi.Text("Yearly"), txt("text-primary", "-20%")))),
				el("flex flex-row w-full gap-2 pt-1",
					tier("Starter", "$0", "$0", "For trying it out", false, "Up to 3 projects", "2 collaborators", "Email support"),
					tier("Team", "$12", "$10", "For teams", true, "Unlimited projects", "Shared timelines", "Guest access", "Priority support"),
					tier("Business", "$29", "$24", "For whole companies", false, "Everything in Team", "Single sign-on", "Audit log", "Dedicated manager"),
				),
			),
			section("max-w-80",
				el("flex flex-col gap-1 rounded-2xl border bg-linear-to-br from-primary/10 via-card to-primary-100 px-4 py-1.5 shadow-md",
					txt("text-primary font-bold", "“"),
					txt("font-medium", "We replaced three tools and a weekly status meeting. Everyone just opens the timeline now."),
					el("flex flex-row items-center gap-1 pt-0.5", avatar("bg-primary-700", "LN"), el("flex flex-col", txt("font-semibold", "Lena Novak"), txt("text-muted-foreground", "Head of Operations, Fernway"))),
				),
			),
			section("gap-1 max-w-80", anchor("FAQ"),
				txt("font-bold text-center", "Frequently asked questions"),
				faq.Node(twi.Class("w-full"),
					question("trial", "How does the free trial work?", "You get every Team feature for 14 days. When it ends, pick a plan or stay on Starter."),
					question("switch", "Can I change plans later?", "Yes. Upgrades apply at once and downgrades at the end of the billing period."),
					question("import", "Can I import from spreadsheets?", "Drop in a CSV and Plainsheet maps the columns to projects, owners and dates."),
					question("data", "Where is my data stored?", "In the region you choose at sign-up, encrypted at rest and in transit."),
				),
			),
			section("items-center gap-1",
				txt("font-bold", "Give your team one calm page."),
				ui.Button(ui.ButtonDefault, ui.ButtonSizeLG, twi.Class("rounded-lg py-0.5 shadow-lg"), twi.Text("Start a free trial")),
			),
			footer("◧ Plainsheet", "Calm planning for small teams.",
				[]string{"Product", "Features", "Pricing", "Changelog"},
				[]string{"Help", "Guides", "FAQ", "Contact"},
				[]string{"Legal", "Privacy", "Terms", "Security"},
			),
		)
	}
	return page{nav: nav, body: body}
}
