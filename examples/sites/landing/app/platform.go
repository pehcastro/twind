package app

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func newPlatform(k kit) page {
	feature := func(glyph, chip, title, body string) twi.Node {
		return el("flex flex-col gap-0.5 rounded-xl border bg-card px-2 py-1 shadow-md transition duration-200 hover:-translate-y-0.5 hover:shadow-xl hover:border-primary/60",
			el("flex flex-row", txt("flex flex-row w-4 justify-center rounded-md py-0.5 font-bold text-zinc-950 "+chip, glyph)),
			txt("font-semibold pt-0.5", title), txt("text-muted-foreground", body))
	}
	stat := func(value, label string) twi.Node {
		return el("flex flex-col flex-1 items-center", txt("font-bold text-primary", value), txt("text-muted-foreground", label))
	}
	step := func(mark, class, name, took string) twi.Node {
		return el("flex flex-row gap-1", txt(class, mark), txt("grow", name), txt("text-muted-foreground", took))
	}
	code := func(parts ...string) twi.Node {
		out := []twi.NodeOption{}
		for i := 0; i < len(parts); i += 2 {
			out = append(out, txt(parts[i], parts[i+1]))
		}
		return el("flex flex-row whitespace-pre", out...)
	}
	bars := func(heights ...string) twi.Node {
		out := []twi.NodeOption{}
		for _, h := range heights {
			out = append(out, el("w-2 rounded-t-md bg-linear-to-t from-primary-700 to-primary-300 "+h))
		}
		return el("flex flex-row items-end gap-0.5 h-5", out...)
	}
	const keyword, name, value, plain = "text-chart-1", "text-chart-2", "text-chart-4", "text-foreground"
	window := func(title string, children ...twi.NodeOption) twi.Node {
		return el("flex flex-col overflow-hidden rounded-xl border bg-card shadow-xl",
			el("flex flex-row items-center gap-1 px-2 py-0.5 border-b bg-muted/40 whitespace-pre",
				lights(), txt("text-muted-foreground pl-1", title)),
			el("flex flex-row gap-3 px-2 py-1", children...))
	}
	nav := func() twi.Node {
		return navbar("bg-background/80 shadow-md",
			el("flex flex-row items-center gap-1", txt("flex flex-row w-4 justify-center rounded-md bg-linear-to-br from-primary-300 to-primary-600 py-0.5 font-bold text-zinc-950", "▲"), txt("font-bold py-0.5", "Quarry")),
			k.links("text-muted-foreground", "Features", "Code", "Pricing"), el("grow"),
			ui.Button(ui.Ghost, ui.SizeSM, twi.Class("rounded-full py-0.5"), twi.Text("Log in")),
			ui.Button(ui.Default, ui.SizeSM, twi.Class("rounded-full py-0.5 shadow-md"), twi.Text("Sign up")),
		)
	}
	body := func() twi.Node {
		return el("flex flex-col shrink-0 gap-3 pt-1",
			section("",
				el("flex flex-col items-center gap-1 rounded-2xl bg-linear-to-b from-primary/20 via-primary/5 to-background px-4 pt-2 pb-1",
					el("flex flex-row items-center gap-1 rounded-full bg-background/70 pl-0.5 pr-2 py-0.5 shadow-[0_0_0_1px_var(--color-border)] transition-colors duration-200 hover:bg-accent",
						txt("rounded-full bg-primary px-1 font-medium text-primary-foreground", "New"), twi.Text("Edge functions in every region"), txt("text-muted-foreground", "→")),
					el("pt-1", display("PUSH TO SHIP.", 1, "text-primary", "text-primary-400", "text-chart-3", "text-chart-2")),
					txt("font-bold text-center", "The whole stack, live from one git push."),
					txt("text-muted-foreground text-center max-w-64", "Quarry builds, previews and deploys your app on every commit, then keeps it fast for people everywhere."),
					el("flex flex-row items-center gap-2 pt-1",
						ui.Button(ui.Default, ui.SizeLG, twi.Class("rounded-full py-0.5 shadow-lg"), twi.Text("▲ Start deploying")),
						ui.Button(ui.Outline, ui.SizeLG, twi.Class("rounded-full py-0.5"), twi.Text("Get a demo")),
					),
					el("flex flex-row items-center gap-1 rounded-lg bg-muted/60 px-2 py-0.5 mt-0.5 shadow-[0_0_0_1px_var(--color-border)] transition-colors duration-200 hover:bg-muted",
						twi.Focusable(), k.copy("npx quarry deploy"),
						txt("text-primary", "$"), twi.Text("npx quarry deploy"), txt("text-muted-foreground pl-2", "⧉ copy")),
				),
				el("pt-2", window("quarry.example/deployments/main",
					el("flex flex-col flex-1 gap-0.5",
						el("flex flex-row items-center gap-1", txt("font-semibold grow", "main · 4f2a91c"), txt("rounded-full bg-primary/15 px-1 text-primary", "● Building")),
						step("✓", "text-primary", "Cloned repository", "0.4 s"),
						step("✓", "text-primary", "Installed packages", "6.1 s"),
						step("✓", "text-primary", "Built 214 pages", "12.3 s"),
						step("◌", "text-chart-4", "Deploying to 38 regions", "…"),
						el("flex flex-row h-0.5 mt-0.5 rounded-full bg-muted", el("w-3/4 rounded-full bg-linear-to-r from-primary-300 via-primary to-chart-2")),
					),
					el("flex flex-col flex-1 gap-0.5",
						el("flex flex-row", txt("font-semibold grow", "Response time"), txt("text-muted-foreground", "p95, last 24 h")),
						bars("h-2", "h-3", "h-2.5", "h-4", "h-3.5", "h-1.5", "h-2", "h-1", "h-1.5", "h-1"),
						el("flex flex-row text-muted-foreground", txt("grow", "120 ms → 38 ms"), txt("text-primary", "-68%")),
					),
				)),
			),
			section("items-center gap-1",
				txt("text-muted-foreground", "Trusted by product teams at"),
				el("flex flex-row justify-between w-full px-4 font-bold text-muted-foreground",
					twi.Text("◆ Halcyon"), twi.Text("◎ Brightfold"), twi.Text("▣ Orbit Labs"), twi.Text("◈ Mosaic"), twi.Text("◉ Northpass")),
			),
			section("", el("flex flex-row gap-2 rounded-xl bg-muted/40 py-1 shadow-[0_0_0_1px_var(--color-border)]",
				stat("99.99%", "uptime"), stat("38", "regions"), stat("120 ms", "cold start"), stat("4.2M", "deploys a week"))),
			section("gap-1 pt-1", anchor("Features"),
				txt("text-primary font-medium", "Platform"),
				txt("font-bold", "Everything a team needs to go from idea to production."),
				el("grid grid-cols-3 gap-2 pt-1",
					feature("◆", "bg-linear-to-br from-primary-300 to-primary-600", "Preview every branch", "Each pull request gets its own live address to review."),
					feature("◎", "bg-linear-to-br from-primary-400 to-chart-2", "Global edge network", "Requests are served from the region closest to the reader."),
					feature("▣", "bg-linear-to-br from-chart-2 to-chart-3", "Serverless functions", "Write an endpoint, push it, and it scales from zero."),
					feature("◉", "bg-linear-to-br from-chart-3 to-chart-4", "Instant rollbacks", "Go back to any earlier deploy in one click, no rebuild."),
					feature("▤", "bg-linear-to-br from-chart-4 to-chart-5", "Built-in analytics", "Real visitor timings, without a script on the page."),
					feature("◈", "bg-linear-to-br from-chart-5 to-primary-400", "Secrets per stage", "Separate values for preview, staging and production."),
				),
			),
			section("flex-row items-center gap-4 pt-1", anchor("Code"),
				el("flex flex-col flex-1 gap-1",
					txt("text-primary font-medium", "Developer experience"),
					txt("font-bold", "Configure in code, not in a dashboard."),
					txt("text-muted-foreground", "One typed file describes routes, caching and regions. Review it like any other change."),
					el("flex flex-row items-center gap-1 pt-1",
						ui.Button(ui.Secondary, ui.SizeDefault, twi.Class("rounded-full py-0.5"), twi.Text("Read the docs")),
						ui.Button(ui.Link, ui.SizeDefault, twi.Class("py-0.5"), twi.Text("See examples →"))),
				),
				el("flex flex-col flex-1", window("quarry.config.ts",
					el("flex flex-col",
						code(keyword, "export default ", plain, "{"),
						code(name, "  framework", plain, ": ", value, "\"vite\"", plain, ","),
						code(name, "  regions", plain, ": [", value, "\"fra\"", plain, ", ", value, "\"iad\"", plain, ", ", value, "\"gru\"", plain, "],"),
						code(name, "  cache", plain, ": { ", name, "maxAge", plain, ": ", keyword, "3600", plain, " },"),
						code(name, "  preview", plain, ": ", keyword, "true", plain, ","),
						code(plain, "}"),
					),
				)),
			),
			section("pt-1", anchor("Pricing"),
				el("flex flex-col items-center gap-1 rounded-2xl bg-linear-to-r from-primary-300 via-primary to-chart-2 px-4 py-2 text-zinc-950 shadow-xl",
					txt("font-bold", "Your next deploy is one command away."),
					txt("", "Free for personal projects. Teams from $20 a seat."),
					el("flex flex-row gap-2 pt-0.5",
						txt("rounded-full bg-zinc-950 px-3 py-0.5 font-medium text-white shadow-lg transition-colors duration-200 hover:bg-zinc-800", "Create a free project"),
						txt("rounded-full bg-white/40 px-3 py-0.5 font-medium transition-colors duration-200 hover:bg-white/70", "Talk to sales")),
				),
			),
			footer("▲ Quarry", "The platform for frontend teams.",
				[]string{"Product", "Previews", "Functions", "Analytics"},
				[]string{"Resources", "Docs", "Guides", "Status"},
				[]string{"Company", "About", "Careers", "Contact"},
			),
		)
	}
	return page{nav: nav, body: body}
}
