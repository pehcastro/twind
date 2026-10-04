package main

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func newPlatform() func() twi.Node {
	feature := func(glyph, title, body string) twi.Node {
		return ui.Card(twi.Class("py-1 gap-0.5"),
			ui.CardHeader(txt("text-primary", glyph), ui.CardTitle(twi.Text(title)), ui.CardDescription(twi.Text(body))),
		)
	}
	stat := func(value, label string) twi.Node {
		return el("flex flex-col flex-1 items-center", txt("font-bold text-primary", value), txt("text-muted-foreground", label))
	}
	code := func(parts ...string) twi.Node {
		out := []twi.NodeOption{}
		for i := 0; i < len(parts); i += 2 {
			out = append(out, txt(parts[i], parts[i+1]))
		}
		return el("flex flex-row whitespace-pre", out...)
	}
	const keyword, name, value, plain = "text-chart-1", "text-chart-2", "text-chart-4", "text-foreground"
	return func() twi.Node {
		return el("flex flex-col shrink-0",
			section("flex-row items-center gap-2 py-0.5",
				txt("font-bold", "▲ Quarry"), links("text-muted-foreground", "Product", "Docs", "Pricing", "Changelog"), el("grow"),
				ui.Button(ui.Ghost, ui.SizeSM, twi.Text("Log in")), ui.Button(ui.Default, ui.SizeSM, twi.Text("Sign up")),
			),
			section("items-center gap-1 pt-3 pb-2",
				ui.Badge(ui.Outline, twi.Text("New  Edge functions in every region  →")),
				txt("font-bold text-center pt-1", "Ship the whole stack"),
				txt("font-bold text-center text-primary", "from one push."),
				txt("text-muted-foreground text-center max-w-64", "Quarry builds, previews and deploys your app on every commit, then keeps it fast for people everywhere."),
				el("flex flex-row gap-2 pt-1",
					ui.Button(ui.Default, ui.SizeLG, twi.Class("py-0.5"), twi.Text("▲ Start deploying")),
					ui.Button(ui.Outline, ui.SizeLG, twi.Text("Get a demo")),
				),
				el("flex flex-row items-center gap-1 px-2 py-0.5 mt-1 rounded-md border bg-muted/40",
					txt("text-muted-foreground", "$"), twi.Text("npx quarry deploy"), txt("text-muted-foreground pl-2", "⧉")),
			),
			section("flex-row gap-2 py-1 border-y", stat("99.99%", "uptime"), stat("38", "regions"), stat("120 ms", "cold start"), stat("4.2M", "deploys a week")),
			section("gap-1 pt-2",
				txt("text-primary font-medium", "Platform"),
				txt("font-bold", "Everything a team needs to go from idea to production."),
				el("grid grid-cols-3 gap-2 pt-1",
					feature("◆", "Preview every branch", "Each pull request gets its own live address to review."),
					feature("◎", "Global edge network", "Requests are served from the region closest to the reader."),
					feature("▣", "Serverless functions", "Write an endpoint, push it, and it scales from zero."),
					feature("◉", "Instant rollbacks", "Go back to any earlier deploy in one click, no rebuild."),
					feature("▤", "Built-in analytics", "Real visitor timings, without a script on the page."),
					feature("◈", "Secrets per stage", "Separate values for preview, staging and production."),
				),
			),
			section("flex-row gap-3 pt-2",
				el("flex flex-col flex-1 gap-1 pt-1",
					txt("text-primary font-medium", "Developer experience"),
					txt("font-bold", "Configure in code, not in a dashboard."),
					txt("text-muted-foreground", "One typed file describes routes, caching and regions. Review it like any other change."),
					el("flex flex-row gap-1 pt-1", ui.Button(ui.Secondary, ui.SizeDefault, twi.Text("Read the docs")), ui.Button(ui.Link, ui.SizeDefault, twi.Text("See examples →"))),
				),
				ui.Card(twi.Class("flex-1 gap-0 py-0 overflow-hidden"),
					el("flex flex-row items-center gap-1 px-2 py-0.5 border-b bg-muted/40",
						txt("text-destructive", "●"), txt("text-chart-4", "●"), txt("text-chart-2", "●"), txt("text-muted-foreground pl-1", "quarry.config.ts")),
					el("flex flex-col px-2 py-1",
						code(keyword, "export default ", plain, "{"),
						code(name, "  framework", plain, ": ", value, "\"vite\"", plain, ","),
						code(name, "  regions", plain, ": [", value, "\"fra\"", plain, ", ", value, "\"iad\"", plain, ", ", value, "\"gru\"", plain, "],"),
						code(name, "  cache", plain, ": { ", name, "maxAge", plain, ": ", keyword, "3600", plain, " },"),
						code(name, "  preview", plain, ": ", keyword, "true", plain, ","),
						code(plain, "}"),
					),
				),
			),
			section("items-center gap-1 py-3",
				txt("font-bold text-center", "Your next deploy is one command away."),
				ui.Button(ui.Default, ui.SizeLG, twi.Class("py-0.5"), twi.Text("Create a free project")),
			),
			footer("▲ Quarry", "The platform for frontend teams.",
				[]string{"Product", "Previews", "Functions", "Analytics"},
				[]string{"Resources", "Docs", "Guides", "Status"},
				[]string{"Company", "About", "Careers", "Contact"},
			),
		)
	}
}
