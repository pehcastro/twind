package docsapp

import (
	"fmt"
	"io/fs"
	"slices"

	"github.com/twind-dev/twind/apps/documentation/components"
	"github.com/twind-dev/twind/docs"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/highlight"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/markdown"
	"github.com/twind-dev/twind/twi/theme"
	"github.com/twind-dev/twind/twi/ui"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen

type Start struct{ Page, Theme string }

type entry struct {
	group, slug, title string
	page               markdown.Page
}

type site struct {
	rt                  *twi.Runtime
	demos               map[string]components.Demo
	source              fs.FS
	entries             []entry
	previews            map[string]*preview
	grammars            map[string]*highlight.Grammar
	colours             map[highlight.Kind]string
	themes              []theme.Theme
	palette             *ui.CommandDialog
	sidebar             *ui.Sidebar
	page, theme, trying int
	picker              bool
}

func newSite(rt *twi.Runtime, demos map[string]components.Demo, source fs.FS) *site {
	return &site{
		rt: rt, demos: demos, source: source, previews: map[string]*preview{},
		entries: []entry{
			{group: "Getting started", slug: "introduction"},
			{group: "Getting started", slug: "installation"},
			{group: "Getting started", slug: "theming"},
			{group: "Components", slug: "button"},
			{group: "Components", slug: "dialog"},
			{group: "Components", slug: "tabs"},
		},
		grammars: map[string]*highlight.Grammar{"go": highlight.Go(), "bash": highlight.Bash(), "json": highlight.JSON(), "toml": highlight.TOML()},
		colours: map[highlight.Kind]string{
			highlight.Keyword: "text-violet-700 dark:text-violet-400", highlight.String: "text-green-700 dark:text-green-400",
			highlight.Escape: "text-amber-700 dark:text-amber-300", highlight.Number: "text-orange-700 dark:text-orange-300",
			highlight.Comment: "text-muted-foreground italic", highlight.Function: "text-blue-700 dark:text-blue-400",
			highlight.Punctuation: "text-muted-foreground", highlight.Property: "text-sky-700 dark:text-sky-300",
			highlight.Boolean: "text-orange-700 dark:text-orange-300", highlight.Variable: "text-amber-700 dark:text-amber-300",
			highlight.Builtin: "text-blue-700 dark:text-blue-400", highlight.Regex: "text-amber-700 dark:text-amber-300",
			highlight.Datetime: "text-orange-700 dark:text-orange-300", highlight.TableHeader: "text-primary",
		},
		themes:  theme.Builtin(),
		palette: ui.NewCommandDialog(rt),
		sidebar: ui.NewSidebar(rt),
	}
}

func (s *site) load(pages fs.FS) error {
	tags := map[string]markdown.Component{
		"Preview": {Attrs: []string{"name"}, Build: func(a map[string]string) twi.Node { return s.previewNode(a["name"]) }},
		"Props":   {Attrs: []string{"of"}, Build: func(a map[string]string) twi.Node { return propsTable(a["of"]) }},
	}
	listed := map[string]bool{}
	for i := range s.entries {
		e := &s.entries[i]
		file := e.slug + ".md"
		src, err := fs.ReadFile(pages, file)
		if err != nil {
			return err
		}
		if e.page, err = markdown.Parse(file, string(src), tags); err != nil {
			return err
		}
		if len(e.page.Anchors) == 0 || e.page.Anchors[0].Level != 1 {
			return fmt.Errorf("%s: the page does not open with a level 1 heading", file)
		}
		e.title, listed[file] = e.page.Anchors[0].Text, true
		if err := s.resolve(file, e.page.Blocks); err != nil {
			return err
		}
	}
	files, err := fs.Glob(pages, "*.md")
	if err != nil {
		return err
	}
	for _, f := range files {
		if !listed[f] {
			return fmt.Errorf("%s: no sidebar entry opens it", f)
		}
	}
	return nil
}

func (s *site) resolve(file string, blocks []markdown.Block) error {
	for _, b := range blocks {
		var err error
		switch {
		case b.Kind == markdown.Tag && b.Name == "Preview":
			err = s.addPreview(b.Attrs["name"])
		case b.Kind == markdown.Tag && b.Name == "Props":
			if _, ok := props()[b.Attrs["of"]]; !ok {
				err = fmt.Errorf("no props table for %q", b.Attrs["of"])
			}
		}
		if err != nil {
			return fmt.Errorf("%s:%d: %w", file, b.Line, err)
		}
		for _, nested := range append([][]markdown.Block{b.Children}, b.Items...) {
			if err := s.resolve(file, nested); err != nil {
				return err
			}
		}
	}
	return nil
}

func New(rt *twi.Runtime, start Start) (func() twi.Node, error) {
	s := newSite(rt, components.All(), components.Source)
	if err := s.load(docs.Pages); err != nil {
		return nil, err
	}
	s.page = slices.IndexFunc(s.entries, func(e entry) bool { return e.slug == start.Page })
	s.theme = slices.IndexFunc(s.themes, func(t theme.Theme) bool { return themeName(t) == start.Theme })
	if s.page < 0 {
		return nil, fmt.Errorf("page %q: not a page of the documentation", start.Page)
	}
	if s.theme < 0 {
		return nil, fmt.Errorf("theme %q: not a built-in theme", start.Theme)
	}
	rt.SetTheme(s.themes[s.theme])
	s.palette.OnSelect = func(value string) {
		if i := slices.IndexFunc(s.entries, func(e entry) bool { return e.title == value }); i >= 0 {
			s.open(i)
		}
		if i := slices.IndexFunc(s.themes, func(t theme.Theme) bool { return themeName(t) == value }); i >= 0 {
			s.theme = i
			rt.SetTheme(s.themes[i])
		}
	}
	return s.view, nil
}

func App(rt *twi.Runtime) func() twi.Node {
	view, err := New(rt, Start{Page: "introduction", Theme: "zinc-dark"})
	if err != nil {
		panic(err)
	}
	return view
}

func themeName(t theme.Theme) string {
	return t.Name + map[theme.Scheme]string{theme.Light: "-light", theme.Dark: "-dark"}[t.Scheme]
}

func (s *site) open(page int) {
	s.page = page
	s.rt.Invalidate()
}

func el(class string, children ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{twi.Class(class)}, children...)...)
}

func txt(class, s string) twi.Node { return el(class, twi.Text(s)) }

func (s *site) view() twi.Node {
	e := s.entries[s.page]
	return el("flex flex-col h-full bg-background text-foreground",
		twi.OnKeyDown(func(ev *twi.Event) {
			k := ev.Key
			if k.Key == input.KeyRune && k.Rune == 't' && k.Modifiers == 0 && !s.palette.Open && !s.picker {
				s.openPicker()
			}
		}),
		el("flex flex-row shrink-0 items-center gap-2 border-b px-2",
			txt("font-bold", "twind"),
			txt("text-muted-foreground", "Docs"),
			el("grow"),
			s.palette.Trigger(ui.Outline, ui.SizeSM, twi.Key("search"), twi.Class("w-40 justify-between text-muted-foreground"),
				twi.Text("Search documentation..."), ui.KbdGroup(ui.Kbd(twi.Text("Ctrl")), ui.Kbd(twi.Text("K")))),
			ui.Button(ui.Ghost, ui.SizeSM, twi.Key("theme"), twi.OnClick(func(*twi.Event) { s.openPicker() }), twi.Text("◐ "+themeName(s.themes[s.theme]))),
		),
		el("flex flex-row flex-1 min-h-0", s.sidebar.Provider(s.nav(), ui.SidebarInset(el("flex flex-row flex-1 min-h-0", s.content(e), s.outline(e))))),
		s.search(),
		s.pickerNode(),
	)
}

func (s *site) nav() twi.Node {
	var groups []twi.NodeOption
	for i := 0; i < len(s.entries); {
		group := s.entries[i].group
		var items []twi.NodeOption
		for ; i < len(s.entries) && s.entries[i].group == group; i++ {
			at, e := i, s.entries[i]
			items = append(items, ui.SidebarMenuItem(ui.SidebarMenuButton(ui.SizeDefault, i == s.page,
				twi.Key("nav-"+e.slug), twi.OnClick(func(*twi.Event) { s.open(at) }), twi.Text(e.title))))
		}
		groups = append(groups, ui.SidebarGroup(ui.SidebarGroupLabel(twi.Text(group)), ui.SidebarGroupContent(ui.SidebarMenu(items...))))
	}
	return s.sidebar.Node(ui.SidebarContent(append(groups, twi.Class("py-1"))...))
}

func (s *site) content(e entry) twi.Node {
	step := func(to int, label string) twi.Node {
		return ui.Button(ui.Secondary, ui.SizeSM, twi.Key("pager-"+s.entries[to].slug), twi.OnClick(func(*twi.Event) { s.open(to) }), twi.Text(label))
	}
	pager := []twi.NodeOption{el("")}
	if s.page > 0 {
		pager[0] = step(s.page-1, "‹ "+s.entries[s.page-1].title)
	}
	if s.page+1 < len(s.entries) {
		pager = append(pager, step(s.page+1, s.entries[s.page+1].title+" ›"))
	}
	return ui.ScrollArea(twi.Key("page-"+e.slug), twi.Class("flex-1 min-w-0 px-3 py-1"),
		el("flex flex-col shrink-0 gap-1",
			ui.Breadcrumb(ui.BreadcrumbList(
				ui.BreadcrumbItem(ui.BreadcrumbLink(twi.Text("Docs"))), ui.BreadcrumbSeparator(),
				ui.BreadcrumbItem(ui.BreadcrumbLink(twi.Text(e.group))), ui.BreadcrumbSeparator(),
				ui.BreadcrumbItem(ui.BreadcrumbPage(twi.Text(e.title))),
			)),
			markdown.Render(e.page, markdown.Options{Highlight: s.code, Follow: s.follow}),
			el("flex flex-row justify-between pt-1", pager...),
		),
	)
}

func (s *site) follow(target string) {
	if i := slices.IndexFunc(s.entries, func(e entry) bool { return e.slug+".md" == target }); i >= 0 {
		s.open(i)
	}
}

func (s *site) outline(e entry) twi.Node {
	items := []twi.NodeOption{txt("font-medium", "On this page")}
	for _, a := range e.page.Anchors {
		switch a.Level {
		case 2:
			items = append(items, txt("text-muted-foreground", a.Text))
		case 3:
			items = append(items, txt("pl-2 text-muted-foreground", a.Text))
		}
	}
	return el("flex flex-col w-24 shrink-0 px-2 py-1", items...)
}

func (s *site) search() twi.Node {
	p := s.palette
	var groups []ui.CommandGroup
	for i := 0; i < len(s.entries); {
		group := s.entries[i].group
		var items []ui.CommandItem
		for ; i < len(s.entries) && s.entries[i].group == group; i++ {
			items = append(items, p.Item(s.entries[i].title))
		}
		groups = append(groups, p.Group(group, items...))
	}
	var themes []ui.CommandItem
	for _, t := range s.themes {
		themes = append(themes, p.Item(themeName(t)))
	}
	return p.Node(p.Input("Search documentation..."), p.List(append(groups, p.Group("Theme", themes...))...))
}

func (s *site) openPicker() {
	s.picker, s.trying = true, s.theme
	s.rt.Invalidate()
}

func (s *site) pickerNode() twi.Node {
	if !s.picker {
		return el("hidden")
	}
	show := func(i int) {
		s.trying = (i + len(s.themes)) % len(s.themes)
		s.rt.SetTheme(s.themes[s.trying])
		s.rt.Invalidate()
	}
	restore := func() {
		s.picker = false
		s.rt.SetTheme(s.themes[s.theme])
		s.rt.Invalidate()
	}
	apply := func() {
		s.theme = s.trying
		restore()
	}
	list := []twi.NodeOption{twi.Key("themes"), twi.Focusable(), twi.Class("flex flex-col rounded-md focus-visible:shadow-[0_0_0_1px_var(--color-ring)]"), twi.OnKeyDown(func(ev *twi.Event) {
		switch ev.Key.Key {
		case input.KeyArrowDown:
			show(s.trying + 1)
		case input.KeyArrowUp:
			show(s.trying - 1)
		case input.KeyEnter:
			apply()
		}
	})}
	first := min(max(s.trying-pickerRows/2, 0), len(s.themes)-pickerRows)
	for i := first; i < first+pickerRows; i++ {
		class, mark := "px-1", "  "
		if i == s.trying {
			class = "px-1 bg-accent text-accent-foreground font-medium"
		}
		if i == s.theme {
			mark = "● "
		}
		list = append(list, twi.Element(twi.Class(class), twi.Text(mark+themeName(s.themes[i])),
			twi.OnPointerEnter(func() { show(i) }), twi.OnClick(func(*twi.Event) { apply() })))
	}
	return twi.Element(twi.Key("picker"), twi.FocusScope(), twi.Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
		twi.OnKeyDown(func(ev *twi.Event) {
			if ev.Key.Key == input.KeyEscape {
				restore()
			}
			ev.StopPropagation()
		}),
		el("flex flex-col w-40 gap-1 rounded-lg border bg-popover px-2 py-1 text-popover-foreground shadow-lg",
			twi.OnPointerDownOutside(restore),
			txt("font-semibold", "Theme"),
			txt("text-muted-foreground", "↑ ↓ preview, Enter keeps, Esc restores"),
			twi.Element(list...),
		),
	)
}
