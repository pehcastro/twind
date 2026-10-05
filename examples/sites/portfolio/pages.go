package main

import (
	"slices"
	"strconv"
	"strings"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/markdown"
	"github.com/pehcastro/twind/twi/ui"
)

type link struct{ label, glyph, url string }

func links() []link {
	return []link{{"Code", "⌥", "git.example/noorv"}, {"Social", "@", "social.example/@noor"}, {"Feed", "◉", "noor.example/feed.xml"}}
}

type project struct {
	name, glyph, about, stars, since, cover string
	tags                                    []string
}

func work() []project {
	return []project{
		{"tide", "≋", "A terminal dashboard for long-running jobs. Redraws only what changed.", "1.2k", "2022", "bg-linear-to-br from-primary-300 via-primary-500 to-primary-700", []string{"go", "tui"}},
		{"loupe", "⌕", "Search structured logs with a query language that fits on one line.", "640", "2023", "bg-linear-to-br from-chart-1 via-chart-2 to-chart-3", []string{"go", "cli", "logs"}},
		{"quay", "⇡", "A tiny deploy tool for teams with one server and no patience.", "310", "2024", "bg-linear-to-br from-primary-200 via-chart-2 to-primary-800", []string{"shell", "ops"}},
		{"inkwell", "✎", "This website. Posts in Markdown, drawn in your terminal.", "new", "2026", "bg-linear-to-br from-chart-2 via-primary-600 to-chart-5", []string{"markdown", "twind"}},
	}
}

func glyphs() map[rune][7]string {
	return map[rune][7]string{
		'N': {"#...#", "##..#", "#.#.#", "#..##", "#...#", "#...#", "#...#"},
		'O': {".###.", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
		'R': {"####.", "#...#", "#...#", "####.", "#.#..", "#..#.", "#...#"},
		'V': {"#...#", "#...#", "#...#", "#...#", "#...#", ".#.#.", "..#.."},
		'A': {".###.", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"},
		'L': {"#....", "#....", "#....", "#....", "#....", "#....", "#####"},
		'E': {"#####", "#....", "#....", "####.", "#....", "#....", "#####"},
		'K': {"#...#", "#..#.", "#.#..", "##...", "#.#..", "#..#.", "#...#"},
		' ': {"..", "..", "..", "..", "..", "..", ".."},
	}
}

func wordmark(word string) twi.Node {
	shade := [7]string{"bg-lime-200", "bg-lime-300", "bg-lime-400", "bg-emerald-400", "bg-emerald-500", "bg-teal-500", "bg-teal-600"}
	font := glyphs()
	rows := []twi.NodeOption{twi.Class("flex flex-col")}
	for y, dot := range shade {
		var line []twi.NodeOption
		for _, r := range word {
			for _, c := range font[r][y] + "." {
				line = append(line, el(map[rune]string{'#': "w-1 h-0.5 rounded-xs " + dot, '.': "w-1 h-0.5"}[c]))
			}
		}
		rows = append(rows, el("flex flex-row", line...))
	}
	return twi.Element(rows...)
}

func section(title, about string, body ...twi.NodeOption) twi.Node {
	return el("flex flex-col gap-1", append([]twi.NodeOption{
		el("flex flex-col", txt("text-primary font-semibold", "◆ "+strings.ToUpper(title)), txt("text-muted-foreground", about)),
	}, body...)...)
}

func pill(s string) twi.Node {
	return txt("rounded-full bg-muted px-1 text-muted-foreground", s)
}

func (s *site) home() twi.Node {
	social := []twi.NodeOption{twi.Class("flex flex-row flex-wrap items-center gap-1")}
	for _, l := range links() {
		social = append(social, ui.Button(ui.Ghost, ui.SizeSM, twi.Class("rounded-full py-0.5"), twi.OnClick(func(*twi.Event) { s.copy(l.label+" link", l.url) }), twi.Text(l.glyph+" "+l.label)))
	}
	stat := func(value, label string) twi.Node {
		return el("flex flex-col grow rounded-xl bg-background/60 px-2 py-0.5 shadow-md", txt("font-bold text-foreground", value), txt("text-muted-foreground", label))
	}
	return el("flex flex-col gap-1.5",
		el("flex flex-col gap-1 rounded-2xl border bg-linear-to-br from-primary/15 via-card to-chart-2/15 px-4 py-1.5 shadow-xl",
			el("flex flex-row", txt("rounded-full bg-emerald-500/15 px-2 py-0.5 text-emerald-500 shadow-[0_0_0_1px_var(--color-emerald-500)]", "● open to work from November")),
			el("hidden sm:flex", wordmark("NOOR VALENKO")),
			txt("sm:hidden font-bold text-primary", strings.ToUpper(name)),
			txt("font-semibold", "Software engineer. Terminals, tooling and quiet software."),
			txt("text-muted-foreground", "I build small tools for people who live in the terminal, and I write about how they work. Right now I keep deploys boring on the platform team of a logistics company."),
			el("flex flex-row flex-wrap items-center gap-1",
				ui.Button(ui.Default, ui.SizeLG, twi.Class("rounded-full py-0.5 shadow-lg"), twi.OnClick(func(*twi.Event) { s.show(blog) }), twi.Text("Read the blog →")),
				ui.Button(ui.Outline, ui.SizeLG, twi.Class("rounded-full py-0.5"), twi.OnClick(func(*twi.Event) { s.show(contact) }), twi.Text("Get in touch")),
				twi.Element(social...)),
			el("flex flex-row gap-1", stat("4", "projects kept alive"), stat("2.1k", "stars, give or take"), stat(strconv.Itoa(len(s.posts)), "posts written")),
		),
		section("Selected work", "Two of the tools I spend my evenings on.", s.cards(featured),
			el("flex flex-row", ui.Button(ui.Link, ui.SizeXS, twi.OnClick(func(*twi.Event) { s.show(projects) }), twi.Text("All projects →")))),
		section("Latest writing", "Short posts, mostly about terminals.", s.list(recentPosts),
			el("flex flex-row", ui.Button(ui.Link, ui.SizeXS, twi.OnClick(func(*twi.Event) { s.show(blog) }), twi.Text("All posts →")))),
	)
}

func (s *site) cards(n int) twi.Node {
	grid := []twi.NodeOption{twi.Class("grid grid-cols-1 sm:grid-cols-2 gap-2")}
	for _, p := range work()[:n] {
		tags := []twi.NodeOption{twi.Class("flex flex-row flex-wrap gap-1")}
		for _, t := range p.tags {
			tags = append(tags, pill(t))
		}
		grid = append(grid, el("group flex flex-col overflow-hidden rounded-xl border bg-card shadow-md transition duration-200 hover:-translate-y-0.5 hover:shadow-xl hover:border-primary/60 focus-visible:shadow-[0_0_0_1px_var(--color-ring)]",
			twi.Key("card-"+p.name), twi.Focusable(), twi.OnClick(func(*twi.Event) { s.copy(p.name+" repository", "git.example/noorv/"+p.name) }),
			el("h-2 "+p.cover),
			el("flex flex-col gap-0.5 px-2 py-1",
				el("flex flex-row items-center gap-1", txt("rounded-md bg-muted px-1 font-bold text-primary", p.glyph), txt("font-semibold transition-colors duration-200 group-hover:text-primary", p.name)),
				txt("text-muted-foreground", p.about),
				twi.Element(tags...),
				el("flex flex-row justify-between text-muted-foreground", txt("", "★ "+p.stars), txt("", "since "+p.since), txt("text-primary group-hover:underline", "repo ↗"))),
		))
	}
	return twi.Element(grid...)
}

func (s *site) list(n int) twi.Node {
	shown := s.posts[:min(n, len(s.posts))]
	rows := []twi.NodeOption{twi.Class("flex flex-col gap-1")}
	for i, p := range shown {
		tags := []twi.NodeOption{twi.Class("flex flex-row flex-wrap items-center gap-1"), txt("text-muted-foreground", strconv.Itoa(p.minutes)+" min read")}
		for _, t := range p.tags {
			tags = append(tags, pill(t))
		}
		row := []twi.NodeOption{twi.Key("row-" + p.file), twi.Focusable(), twi.OnClick(func(*twi.Event) { s.read(i) }), twi.OnKeyDown(func(e *twi.Event) {
			step := map[input.Key]int{input.KeyArrowDown: 1, input.KeyArrowUp: -1}[e.Key.Key]
			if j := i + step; step != 0 && !e.Key.Release && j >= 0 && j < len(shown) && s.rt.Focus("row-"+shown[j].file) {
				e.PreventDefault()
				e.StopPropagation()
			}
		}),
			el("flex flex-col items-center justify-center w-7 shrink-0 rounded-lg bg-muted py-0.5",
				txt("font-bold", strconv.Itoa(p.day.Day())), txt("text-muted-foreground", strings.ToUpper(p.day.Format("Jan")))),
			el("flex flex-col grow min-w-0",
				txt("font-semibold transition-colors duration-200 group-hover:text-primary", p.title),
				txt("text-muted-foreground truncate", p.summary),
				twi.Element(tags...)),
			txt("self-center text-muted-foreground transition-colors duration-200 group-hover:text-primary", "→"),
		}
		if i == s.last {
			row = append(row, twi.AutoFocus())
		}
		rows = append(rows, el("group flex flex-row items-center gap-2 rounded-xl border bg-card px-2 py-0.5 shadow-sm transition duration-200 hover:shadow-lg hover:border-primary/50 focus-visible:border-primary focus-visible:bg-accent", row...))
	}
	return twi.Element(rows...)
}

func (s *site) article(i int) twi.Node {
	p := s.posts[i]
	step := func(to int, label, align string) twi.Node {
		if to < 0 || to >= len(s.posts) {
			return el("grow basis-0")
		}
		return el("flex flex-col grow basis-0 rounded-xl border bg-card px-2 py-0.5 shadow-sm transition duration-200 hover:shadow-lg hover:border-primary/50 "+align,
			twi.Focusable(), twi.OnClick(func(*twi.Event) { s.read(to) }), txt("text-muted-foreground", label), txt("font-semibold", s.posts[to].title))
	}
	tags := []twi.NodeOption{twi.Class("flex flex-row flex-wrap items-center gap-1"),
		txt("text-muted-foreground", p.day.Format(shownDate)+" · "+strconv.Itoa(p.minutes)+" min read")}
	for _, t := range p.tags {
		tags = append(tags, pill(t))
	}
	return el("flex flex-col items-center",
		el("flex flex-col w-full max-w-76 gap-1",
			el("flex flex-row", ui.Button(ui.Ghost, ui.SizeSM, twi.Class("rounded-full py-0.5"), twi.OnClick(func(*twi.Event) { s.show(blog) }), twi.Text("← All posts"))),
			el("flex flex-col gap-0.5 rounded-2xl border bg-linear-to-br from-primary/15 via-card to-chart-2/10 px-3 py-1 shadow-lg",
				twi.Element(tags...),
				txt("font-bold text-primary", p.title),
				txt("", p.summary)),
			markdown.Render(p.body, markdown.Options{Highlight: s.code, Follow: s.follow}),
			el("flex flex-row gap-2 pt-1", step(i-1, "← Newer", "items-start"), step(i+1, "Older →", "items-end")),
		))
}

func (s *site) follow(target string) {
	if i := slices.IndexFunc(s.posts, func(p post) bool { return p.file == target }); i >= 0 {
		s.read(i)
	}
}

func (s *site) code(language, src string) twi.Node {
	return el("flex flex-col pb-0.5", s.highlighter.Code(language, src))
}

func (s *site) contact() twi.Node {
	cards := []twi.NodeOption{twi.Class("grid grid-cols-1 sm:grid-cols-3 gap-2")}
	for _, l := range links() {
		cards = append(cards, el("group flex flex-col gap-0.5 rounded-xl border bg-card px-2 py-1 shadow-md transition duration-200 hover:-translate-y-0.5 hover:shadow-xl hover:border-primary/60",
			twi.Focusable(), twi.OnClick(func(*twi.Event) { s.copy(l.label+" link", l.url) }),
			el("flex flex-row items-center gap-1", txt("rounded-lg bg-muted px-1 text-primary font-bold", l.glyph), txt("font-semibold", l.label)),
			txt("text-muted-foreground truncate", l.url),
			txt("text-primary group-hover:underline", "Copy link")))
	}
	return section("Contact", "The fastest way to reach me is email. I answer within a few days.",
		el("flex flex-col items-center gap-1 rounded-2xl border bg-linear-to-br from-primary/20 via-card to-chart-2/20 px-4 py-1.5 shadow-xl",
			txt("font-bold text-primary", "Let's make something quiet."),
			txt("text-muted-foreground text-center", "Questions about a project, a post, or a job. Short is fine."),
			el("flex flex-row items-center gap-1 rounded-full bg-background/70 pl-2 pr-0.5 py-0.5 shadow-md",
				txt("py-0.5 font-medium", email),
				ui.Button(ui.Default, ui.SizeSM, twi.Key("copy-email"), twi.Class("rounded-full py-0.5"), twi.OnClick(func(*twi.Event) { s.copy("Email", email) }), twi.Text("Copy")))),
		twi.Element(cards...),
	)
}
