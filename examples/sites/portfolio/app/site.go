package app

import (
	"slices"
	"strconv"
	"strings"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/icon"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/markdown"
	"github.com/pehcastro/twind/twi/theme"
	"github.com/pehcastro/twind/twi/ui"
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
	wordsPerMinute = 200
	dateLayout     = "2006-01-02"
	shownDate      = "Jan 2, 2006"
	recentPosts    = 2
	featured       = 2
	flipScheme     = "Switch light and dark"
)

type site struct {
	rt          *twi.Runtime
	theme       theme.Theme
	posts       []post
	at          page
	reading     int
	last        int
	toaster     *ui.Toaster
	palette     *ui.CommandDialog
	highlighter markdown.Highlighter
}

func newSite(rt *twi.Runtime, t theme.Theme, posts []post) *site {
	rt.SetTheme(t)
	s := &site{rt: rt, theme: t, posts: posts, reading: -1, last: -1, toaster: ui.NewToaster(rt), palette: ui.NewCommandDialog(rt)}
	s.palette.Key = "palette"
	s.palette.OnSelect = func(value string) {
		if value == flipScheme {
			s.flip()
		} else if !s.open(value) {
			s.read(slices.IndexFunc(posts, func(p post) bool { return p.title == value }))
		}
	}
	return s
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

func (s *site) open(name string) bool {
	for p := range contact + 1 {
		if strings.EqualFold(p.String(), name) {
			s.show(p)
			return true
		}
	}
	i := slices.IndexFunc(s.posts, func(p post) bool { return p.file == name || p.file == name+".md" })
	if i >= 0 {
		s.read(i)
	}
	return i >= 0
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
	s.rt.Copy(value)
	s.toaster.Success(what+" copied", value, ui.ToastAction{})
}

func (s *site) keys(e *twi.Event) {
	k := e.Key
	if k.Release || k.Modifiers != 0 || s.palette.Open {
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
	nav := []twi.NodeOption{twi.Class("hidden sm:flex flex-row items-center gap-1 rounded-full bg-muted/60 px-1 py-0.5 shadow-[0_0_0_1px_var(--color-border)]")}
	for p := range contact + 1 {
		class := "rounded-full px-2 text-muted-foreground transition-colors duration-200 hover:bg-background/70 hover:text-foreground focus-visible:shadow-[0_0_0_1px_var(--color-ring)]"
		if p == s.at {
			class = "rounded-full px-2 bg-primary/15 text-primary font-medium transition-colors duration-200"
		}
		nav = append(nav, el(class, twi.Text(p.String()), twi.Key("nav-"+p.String()), twi.Focusable(), twi.OnClick(func(*twi.Event) { s.show(p) })))
	}
	key := s.at.String()
	if s.reading >= 0 {
		key = "post-" + s.posts[s.reading].file
	}
	area := []twi.NodeOption{twi.Key(key), twi.Class("flex-1 min-h-0 items-center px-2 focus-visible:shadow-none"),
		el("flex flex-col shrink-0 w-full max-w-88 gap-1.5 py-1.5 animate-in fade-in-0 slide-in-from-bottom-2 duration-300", s.page())}
	if s.reading >= 0 {
		area = append(area, twi.AutoFocus())
	}
	return el("flex flex-col h-full bg-background text-foreground", twi.OnKeyDown(s.keys),
		el("flex flex-row shrink-0 justify-center border-b bg-card/60 shadow-md px-2 py-0.5",
			el("flex flex-row grow items-center gap-2 max-w-88",
				el("flex flex-row shrink-0 items-center gap-1 whitespace-nowrap", twi.OnClick(func(*twi.Event) { s.show(home) }),
					txt("rounded-lg bg-linear-to-br from-primary-300 to-primary-600 px-1 py-0.5 font-bold text-zinc-950 shadow-md", "nv"),
					txt("py-0.5 font-semibold", name)),
				el("grow"),
				twi.Element(nav...),
				s.palette.Trigger(ui.ButtonOutline, ui.ButtonSizeSM, twi.Key("search"), twi.Class("rounded-full py-0.5 text-muted-foreground"),
					twi.Text(string(icon.Search.Glyph())), txt("hidden lg:flex", "Search"), ui.Kbd(twi.Class("hidden md:flex"), twi.Text("Ctrl K"))),
				ui.Button(ui.ButtonGhost, ui.ButtonSizeIcon, twi.Key("scheme"), twi.Class("rounded-full py-0.5"), twi.OnClick(func(*twi.Event) { s.flip() }),
					twi.Text(string(map[theme.Scheme]icon.Name{theme.Light: icon.Sun, theme.Dark: icon.Moon}[s.theme.Scheme].Glyph()))),
			),
		),
		ui.ScrollArea(area...),
		el("flex flex-row shrink-0 justify-center border-t bg-card/60 px-2 py-0.5 text-muted-foreground",
			el("flex flex-row grow max-w-88 items-center justify-between gap-2",
				el("flex flex-row items-center gap-1 shrink-0", txt("rounded-full bg-linear-to-r from-primary-300 to-primary-600 w-2", " "), twi.Text("© 2026 "+name+" · built with Twind")),
				txt("hidden md:flex truncate", "Ctrl K search · 1-4 pages · m theme · q quit"))),
		s.search(),
		s.toaster.Node(),
	)
}

func (s *site) search() twi.Node {
	p := s.palette
	var pages, posts []ui.CommandItem
	for pg := range contact + 1 {
		pages = append(pages, p.Item(pg.String()))
	}
	for _, post := range s.posts {
		posts = append(posts, p.Item(post.title))
	}
	return p.Node(p.Input("Jump to a page or a post..."), p.List(p.Group("Pages", pages...), p.Group("Posts", posts...), p.Group("Theme", p.Item(flipScheme))))
}

func (s *site) page() twi.Node {
	switch {
	case s.reading >= 0:
		return s.article(s.reading)
	case s.at == home:
		return s.home()
	case s.at == projects:
		return section("Projects", "Things I made and still look after. All open source, all small on purpose.", s.cards(len(work())))
	case s.at == blog:
		return section("Writing", "Notes on terminals, tools and the work around them. Newest first.", s.list(len(s.posts)))
	case s.at == contact:
		return s.contact()
	}
	panic("portfolio: unknown page " + strconv.Itoa(int(s.at)))
}
