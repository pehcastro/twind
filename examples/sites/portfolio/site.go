package main

import (
	"slices"
	"strconv"
	"strings"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/highlight"
	"github.com/twind-dev/twind/twi/icon"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/markdown"
	"github.com/twind-dev/twind/twi/theme"
	"github.com/twind-dev/twind/twi/ui"
)

type page uint8

const (
	home page = iota
	projects
	blog
	contact
)

func (p page) String() string {
	return [...]string{home: "Home", projects: "Projects", blog: "Blog", contact: "Contact"}[p]
}

const (
	name           = "Noor Valenko"
	email          = "hello@noor.example"
	tabCell        = "    "
	wordsPerMinute = 200
	dateLayout     = "2006-01-02"
	shownDate      = "Jan 2, 2006"
	recentPosts    = 3
)

type link struct{ label, url string }

func links() []link {
	return []link{{"Code", "git.example/noorv"}, {"Social", "social.example/@noor"}, {"Feed", "noor.example/feed.xml"}}
}

type project struct {
	name, about, meta string
	tags              []string
}

func work() []project {
	return []project{
		{"tide", "A terminal dashboard for long-running jobs. Redraws only what changed.", "★ 1.2k · since 2022", []string{"go", "tui"}},
		{"loupe", "Search structured logs with a query language that fits on one line.", "★ 640 · since 2023", []string{"go", "cli", "logs"}},
		{"quay", "A tiny deploy tool for teams with one server and no patience.", "★ 310 · since 2024", []string{"shell", "ops"}},
		{"inkwell", "This website. Posts in Markdown, rendered in your terminal.", "since 2026", []string{"markdown", "twind"}},
	}
}

type site struct {
	rt      *twi.Runtime
	theme   theme.Theme
	posts   []post
	at      page
	reading int
	last    int
	toaster *ui.Toaster
	syntax  map[string]*highlight.Grammar
}

func newSite(rt *twi.Runtime, t theme.Theme, posts []post) *site {
	rt.SetTheme(t)
	return &site{rt: rt, theme: t, posts: posts, reading: -1, last: -1, toaster: ui.NewToaster(rt),
		syntax: map[string]*highlight.Grammar{"go": highlight.Go(), "bash": highlight.Bash()}}
}

func el(class string, children ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{twi.Class(class)}, children...)...)
}

func txt(class, s string) twi.Node { return el(class, twi.Text(s)) }

func (s *site) show(p page) {
	if s.reading >= 0 {
		s.last = s.reading
	}
	s.at, s.reading = p, -1
	s.rt.Invalidate()
}

func (s *site) read(i int) {
	s.at, s.reading = blog, i
	s.rt.Invalidate()
}

func (s *site) flip() {
	s.theme = s.theme.WithScheme(map[theme.Scheme]theme.Scheme{theme.Light: theme.Dark, theme.Dark: theme.Light}[s.theme.Scheme])
	s.rt.SetTheme(s.theme)
	s.rt.Invalidate()
}

func (s *site) copy(what, value string) {
	if s.rt.Copy(value) == nil {
		s.toaster.Show(what+" copied", value, ui.ToastAction{})
	}
}

func (s *site) keys(e *twi.Event) {
	k := e.Key
	if k.Release || k.Modifiers != 0 {
		return
	}
	switch {
	case (k.Key == input.KeyEscape || k.Key == input.KeyBackspace) && s.reading >= 0:
		s.show(blog)
	case k.Key != input.KeyRune:
		return
	case k.Rune >= '1' && k.Rune <= '4':
		s.show(page(k.Rune - '1'))
	case k.Rune == 'm':
		s.flip()
	case k.Rune == 'q':
		s.rt.Quit()
	default:
		return
	}
	e.PreventDefault()
}

func (s *site) view() twi.Node {
	nav := []twi.NodeOption{twi.Class("flex flex-row items-center gap-1")}
	for p := range contact + 1 {
		class := "text-muted-foreground"
		if p == s.at {
			class = "text-foreground underline"
		}
		nav = append(nav, ui.Button(ui.Ghost, ui.SizeSM, twi.Key("nav-"+p.String()), twi.Class(class), twi.OnClick(func(*twi.Event) { s.show(p) }), twi.Text(p.String())))
	}
	area := []twi.NodeOption{twi.Key(s.at.String()), twi.Class("flex-1 min-h-0 items-center px-2 py-1 focus-visible:shadow-none"), el("flex flex-col shrink-0 w-full max-w-84 gap-1", s.page())}
	if s.reading >= 0 {
		area = append(area, twi.Key("post-"+s.posts[s.reading].file), twi.AutoFocus())
	}
	return el("flex flex-col h-full bg-background text-foreground", twi.OnKeyDown(s.keys),
		el("flex flex-row shrink-0 items-center justify-center border-b px-2",
			el("flex flex-row grow items-center gap-2 max-w-84",
				el("flex flex-row items-center gap-1", twi.OnClick(func(*twi.Event) { s.show(home) }),
					txt("px-1 rounded-md bg-primary text-primary-foreground font-bold", "nv"), txt("font-semibold", name)),
				el("grow"),
				twi.Element(nav...),
				ui.Button(ui.Ghost, ui.SizeSM, twi.Key("scheme"), twi.OnClick(func(*twi.Event) { s.flip() }),
					twi.Text(string(map[theme.Scheme]icon.Name{theme.Light: icon.Sun, theme.Dark: icon.Moon}[s.theme.Scheme].Glyph()))),
			),
		),
		ui.ScrollArea(area...),
		el("flex flex-row shrink-0 justify-center border-t px-2 text-muted-foreground",
			el("flex flex-row grow max-w-84 justify-between gap-2",
				txt("shrink-0 whitespace-nowrap", "© 2026 "+name+" · made with Twind"),
				txt("hidden md:flex truncate", "1-4 pages · esc back · m scheme · q quit"))),
		s.toaster.Node(),
	)
}

func (s *site) page() twi.Node {
	switch {
	case s.reading >= 0:
		return s.article(s.reading)
	case s.at == home:
		return s.home()
	case s.at == projects:
		return s.projects()
	case s.at == blog:
		return intro("Writing", "Notes on terminals, tools and the work around them. Newest first.", s.list(len(s.posts)))
	case s.at == contact:
		return s.contact()
	}
	panic("portfolio: unknown page " + strconv.Itoa(int(s.at)))
}

func heading(s string) twi.Node { return txt("font-semibold", s) }

func intro(title, about string, body ...twi.NodeOption) twi.Node {
	return el("flex flex-col gap-1", append([]twi.NodeOption{txt("font-bold", title), txt("text-muted-foreground", about)}, body...)...)
}

func (s *site) home() twi.Node {
	social := []twi.NodeOption{twi.Class("flex flex-row flex-wrap gap-1")}
	for _, l := range links() {
		social = append(social, ui.Button(ui.Outline, ui.SizeSM, twi.OnClick(func(*twi.Event) { s.copy(l.label+" link", l.url) }), twi.Text(l.label+" ↗")))
	}
	return el("flex flex-col gap-1",
		el("flex flex-row items-center gap-2",
			ui.Avatar(ui.SizeDefault, ui.AvatarFallback(twi.Text("NV"))),
			el("flex flex-col", txt("font-bold", name), txt("text-muted-foreground", "Software engineer. Terminals, tooling and quiet software."))),
		txt("", "I build small tools for people who live in the terminal, and I write about how they work. Right now I am on the platform team at a logistics company, keeping deploys boring."),
		twi.Element(social...),
		heading("Now"),
		el("flex flex-col",
			txt("", "• Writing a renderer that sends fewer bytes than it has to."),
			txt("", "• Reading about text layout, slowly."),
			txt("", "• Learning to bake bread that is not a brick.")),
		heading("Recent writing"),
		s.list(recentPosts),
		el("flex flex-row", ui.Button(ui.Link, ui.SizeXS, twi.OnClick(func(*twi.Event) { s.show(blog) }), twi.Text("All posts ›"))),
	)
}

func (s *site) list(n int) twi.Node {
	shown := s.posts[:min(n, len(s.posts))]
	rows := []twi.NodeOption{twi.Class("flex flex-col")}
	for i, p := range shown {
		row := []twi.NodeOption{twi.Key("row-" + p.file), twi.Focusable(), twi.OnClick(func(*twi.Event) { s.read(i) }), twi.OnKeyDown(func(e *twi.Event) {
			step := map[input.Key]int{input.KeyArrowDown: 1, input.KeyArrowUp: -1}[e.Key.Key]
			if j := i + step; step != 0 && !e.Key.Release && j >= 0 && j < len(shown) && s.rt.Focus("row-"+shown[j].file) {
				e.PreventDefault()
				e.StopPropagation()
			}
		}),
			txt("w-12 shrink-0 text-muted-foreground", p.date),
			el("flex flex-col min-w-0", txt("font-medium", p.title), txt("text-muted-foreground truncate", p.summary)),
		}
		if i == s.last {
			row = append(row, twi.AutoFocus())
		}
		rows = append(rows, el("flex flex-row gap-2 rounded-md px-1 hover:bg-accent focus-visible:bg-accent", row...))
	}
	return twi.Element(rows...)
}

func (s *site) article(i int) twi.Node {
	p := s.posts[i]
	pager := []twi.NodeOption{twi.Class("flex flex-row justify-between pt-1 border-t"), el("")}
	if i > 0 {
		pager[1] = ui.Button(ui.Ghost, ui.SizeSM, twi.OnClick(func(*twi.Event) { s.read(i - 1) }), twi.Text("‹ "+s.posts[i-1].title))
	}
	if i+1 < len(s.posts) {
		pager = append(pager, ui.Button(ui.Ghost, ui.SizeSM, twi.OnClick(func(*twi.Event) { s.read(i + 1) }), twi.Text(s.posts[i+1].title+" ›")))
	}
	return el("flex flex-col gap-1",
		el("flex flex-row", ui.Button(ui.Ghost, ui.SizeXS, twi.OnClick(func(*twi.Event) { s.show(blog) }), twi.Text("‹ All posts"))),
		markdown.Render(p.page, markdown.Options{Highlight: s.code, Follow: s.follow}),
		twi.Element(pager...),
	)
}

func (s *site) follow(target string) {
	if i := slices.IndexFunc(s.posts, func(p post) bool { return p.file == target }); i >= 0 {
		s.read(i)
	}
}

func (s *site) code(language, src string) twi.Node {
	colour := map[highlight.Kind]string{
		highlight.Keyword: "text-syntax-keyword", highlight.String: "text-syntax-string", highlight.Escape: "text-syntax-constant",
		highlight.Number: "text-syntax-number", highlight.Comment: "text-syntax-comment italic", highlight.Function: "text-syntax-function",
		highlight.Builtin: "text-syntax-constant", highlight.Variable: "text-syntax-parameter", highlight.Punctuation: "text-syntax-punctuation",
	}
	lines := [][]twi.NodeOption{nil}
	emit := func(kind highlight.Kind, text string) {
		for i, piece := range strings.Split(text, "\n") {
			if i > 0 {
				lines = append(lines, nil)
			}
			if piece != "" {
				lines[len(lines)-1] = append(lines[len(lines)-1], txt(colour[kind], strings.ReplaceAll(piece, "\t", tabCell)))
			}
		}
	}
	src = strings.TrimSuffix(src, "\n")
	if g := s.syntax[language]; g != nil {
		for span := range highlight.Tokens(src, g) {
			emit(span.Kind, src[span.Start:span.End])
		}
	} else {
		emit(highlight.Text, src)
	}
	rows := make([]twi.NodeOption, len(lines))
	for i, line := range lines {
		rows[i] = el("flex flex-row h-1 shrink-0", line...)
	}
	return el("flex flex-col pb-1", rows...)
}

func (s *site) projects() twi.Node {
	cards := []twi.NodeOption{twi.Class("grid grid-cols-1 sm:grid-cols-2 gap-2")}
	for _, p := range work() {
		tags := []twi.NodeOption{twi.Class("flex flex-row flex-wrap gap-1")}
		for _, t := range p.tags {
			tags = append(tags, ui.Badge(ui.Outline, twi.Text(t)))
		}
		cards = append(cards, ui.Card(twi.Class("py-1"),
			ui.CardHeader(ui.CardTitle(twi.Text(p.name)), ui.CardDescription(twi.Text(p.about))),
			ui.CardContent(twi.Element(tags...)),
			ui.CardFooter(txt("text-muted-foreground", p.meta)),
		))
	}
	return intro("Projects", "Things I made and still look after. All open source.", twi.Element(cards...))
}

func (s *site) contact() twi.Node {
	rows := []twi.NodeOption{twi.Class("flex flex-col")}
	for _, l := range append([]link{{"Email", email}}, links()...) {
		rows = append(rows, el("flex flex-row items-center gap-2",
			txt("w-8 shrink-0 text-muted-foreground", l.label), txt("grow min-w-0 truncate", l.url),
			ui.Button(ui.Outline, ui.SizeXS, twi.OnClick(func(*twi.Event) { s.copy(l.label, l.url) }), twi.Text("Copy"))))
	}
	return intro("Contact", "The fastest way to reach me is email. I answer within a few days.",
		ui.Card(twi.Class("py-1"),
			ui.CardHeader(ui.CardTitle(twi.Text("Say hello")), ui.CardDescription(twi.Text("Questions about a project, a post, or a job. Short is fine."))),
			ui.CardContent(twi.Element(rows...))),
		ui.Alert(ui.Default, twi.Class("px-2 py-1"), ui.AlertTitle(twi.Text("Open to work")), ui.AlertDescription(twi.Text("Part time, remote, from November. Tooling or platform teams."))),
	)
}
