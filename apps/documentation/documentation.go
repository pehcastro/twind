package docsapp

import (
	"errors"
	"fmt"
	"image"
	"io"
	"io/fs"
	"slices"
	"strings"

	"github.com/twind-dev/twind/apps/documentation/blocks"
	"github.com/twind-dev/twind/apps/documentation/components"
	"github.com/twind-dev/twind/docs"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/highlight"
	"github.com/twind-dev/twind/twi/icon"
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
	parsed             bool
}

type site struct {
	rt                  *twi.Runtime
	catalogs            []components.Catalog
	pages               fs.FS
	commands            []ui.CommandGroup
	entries             []entry
	folds               map[string]*ui.Collapsible
	previews            map[string]*preview
	props               map[string][]prop
	grammars            map[string]*highlight.Grammar
	colours             map[highlight.Kind]string
	themes              []theme.Theme
	palette             *ui.CommandDialog
	sidebar             *ui.Sidebar
	area, body          *twi.Ref
	heads               map[string]*twi.Ref
	page, theme, trying int
	heldRow, returnRow  int
	scheme              theme.Scheme
	section             int
	picker              bool
	copied              string
	copying             *twi.Timer
}

func newSite(rt *twi.Runtime, catalogs ...components.Catalog) *site {
	var entries []entry
	folds := map[string]*ui.Collapsible{}
	for _, group := range [][]string{
		{"Getting started", "introduction", "installation", "theming", "cli"},
		{"Guides", "layout", "text", "motion", "events", "driving"},
		{"Components", "accordion", "alert", "alert-dialog", "aspect-ratio", "attachment", "avatar", "badge", "breadcrumb", "bubble", "button", "button-group",
			"calendar", "card", "carousel", "checkbox", "collapsible", "combobox", "command", "context-menu", "dialog", "drawer", "dropdown-menu",
			"field", "hover-card", "input", "input-group", "input-otp", "item", "kbd", "label", "marker", "menubar", "message", "message-scroller", "native-select",
			"navigation-menu", "pagination", "popover", "progress", "radio-group", "resizable", "scroll-area", "select", "separator", "sheet",
			"sidebar", "skeleton", "slider", "spinner", "switch", "table", "tabs", "textarea", "toaster", "toggle", "toggle-group", "tooltip"},
		{"Examples", "blocks"},
	} {
		for _, slug := range group[1:] {
			entries = append(entries, entry{group: group[0], slug: slug})
		}
		folds[group[0]] = ui.NewCollapsible(rt)
		folds[group[0]].Open = group[0] != "Components"
	}
	return &site{
		rt: rt, catalogs: catalogs, entries: entries, folds: folds, previews: map[string]*preview{}, props: props(),
		grammars: map[string]*highlight.Grammar{},
		colours: map[highlight.Kind]string{
			highlight.Keyword: "text-violet-700 dark:text-violet-400", highlight.String: "text-green-700 dark:text-green-400",
			highlight.Escape: "text-amber-700 dark:text-amber-300", highlight.Number: "text-orange-700 dark:text-orange-300",
			highlight.Comment: "text-muted-foreground italic", highlight.Function: "text-blue-700 dark:text-blue-400",
			highlight.Punctuation: "text-muted-foreground", highlight.Property: "text-sky-700 dark:text-sky-300",
			highlight.Boolean: "text-orange-700 dark:text-orange-300", highlight.Variable: "text-amber-700 dark:text-amber-300",
			highlight.Builtin: "text-blue-700 dark:text-blue-400", highlight.Regex: "text-amber-700 dark:text-amber-300",
			highlight.Datetime: "text-orange-700 dark:text-orange-300", highlight.TableHeader: "text-primary",
		},
		themes:  slices.DeleteFunc(theme.Builtin(), func(t theme.Theme) bool { return t.Scheme == theme.Dark }),
		palette: ui.NewCommandDialog(rt),
		sidebar: ui.NewSidebar(rt),
		area:    twi.NewRef(rt),
		body:    twi.NewRef(rt),
		heads:   map[string]*twi.Ref{},
		heldRow: -1, returnRow: -1,
	}
}

func (s *site) load(pages fs.FS) error {
	listed, titles := map[string]bool{}, map[string]bool{}
	var head [titleBytes]byte
	for i := range s.entries {
		e := &s.entries[i]
		file := e.slug + ".md"
		f, err := pages.Open(file)
		if err != nil {
			return err
		}
		n, err := f.Read(head[:])
		if err := errors.Join(err, f.Close()); err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		first, _, _ := strings.Cut(string(head[:n]), "\n")
		title, ok := strings.CutPrefix(strings.TrimSpace(first), "# ")
		if !ok {
			return fmt.Errorf("%s: the page does not open with a level 1 heading", file)
		}
		if titles[title] {
			return fmt.Errorf("%s: another page is also titled %q", file, title)
		}
		e.title, listed[file], titles[title] = title, true, true
	}
	s.pages = pages
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

func (s *site) parse(i int) error {
	e := &s.entries[i]
	if e.parsed {
		return nil
	}
	tags := map[string]markdown.Component{
		"Preview": {Attrs: []string{"name"}, Build: func(a map[string]string) twi.Node { return s.previewNode(a["name"]) }},
		"Props":   {Attrs: []string{"of"}, Build: func(a map[string]string) twi.Node { return s.propsTable(a["of"]) }},
	}
	file := e.slug + ".md"
	src, err := fs.ReadFile(s.pages, file)
	if err != nil {
		return err
	}
	page, err := markdown.Parse(file, string(src), tags)
	if err != nil {
		return err
	}
	if err := s.resolve(file, page.Blocks); err != nil {
		return err
	}
	e.page, e.parsed = page, true
	return nil
}

func (s *site) resolve(file string, blocks []markdown.Block) error {
	for _, b := range blocks {
		var err error
		switch {
		case b.Kind == markdown.Tag && b.Name == "Preview":
			err = s.addPreview(b.Attrs["name"])
		case b.Kind == markdown.Tag && b.Name == "Props":
			if _, ok := s.props[b.Attrs["of"]]; !ok {
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
	s, err := build(rt, start)
	if err != nil {
		return nil, err
	}
	return s.view, nil
}

func (s *site) remember() {
	s.rt.Remember("docs.page", s.pageMemo, s.recallPage)
	s.rt.Remember("docs.theme", s.themeMemo, s.recallTheme)
}

func (s *site) pageMemo() string { return s.entries[s.page].slug }

func (s *site) recallPage(slug string) {
	if i := slices.IndexFunc(s.entries, func(e entry) bool { return e.slug == slug }); i >= 0 {
		s.open(i)
	}
}

func (s *site) themeMemo() string { return themeName(s.themes[s.theme].WithScheme(s.scheme)) }

func (s *site) recallTheme(name string) {
	for i, t := range s.themes {
		for _, scheme := range []theme.Scheme{theme.Light, theme.Dark} {
			if themeName(t.WithScheme(scheme)) == name {
				s.theme, s.scheme = i, scheme
				s.show(i)
			}
		}
	}
}

func build(rt *twi.Runtime, start Start) (*site, error) {
	s := newSite(rt, components.All(), blocks.All())
	if err := s.load(docs.Pages); err != nil {
		return nil, err
	}
	s.page = slices.IndexFunc(s.entries, func(e entry) bool { return e.slug == start.Page })
	s.theme = -1
	s.recallTheme(start.Theme)
	if s.page < 0 {
		return nil, fmt.Errorf("page %q: not a page of the documentation", start.Page)
	}
	if s.theme < 0 {
		return nil, fmt.Errorf("theme %q: not a built-in theme", start.Theme)
	}
	if err := s.parse(s.page); err != nil {
		return nil, err
	}
	s.folds[s.entries[s.page].group].Open = true
	s.palette.Key = "palette"
	s.palette.OnSelect = func(value string) {
		if i := slices.IndexFunc(s.entries, func(e entry) bool { return e.title == value }); i >= 0 {
			s.open(i)
			s.returnRow = s.heldRow
		}
		if i := slices.IndexFunc(s.themes, func(t theme.Theme) bool { return t.Name == value }); i >= 0 {
			s.theme = i
			s.show(i)
		}
	}
	return s, nil
}

func (s *site) show(i int) {
	s.rt.SetTheme(s.themes[i].WithScheme(s.scheme))
	s.rt.Invalidate()
}

func (s *site) flipScheme() {
	s.scheme = map[theme.Scheme]theme.Scheme{theme.Light: theme.Dark, theme.Dark: theme.Light}[s.scheme]
	s.show(s.theme)
}

func App(rt *twi.Runtime) func() twi.Node {
	view, err := New(rt, Start{Page: "introduction", Theme: "twind-dark"})
	if err != nil {
		panic(err)
	}
	return view
}

func themeName(t theme.Theme) string {
	return t.Name + map[theme.Scheme]string{theme.Light: "-light", theme.Dark: "-dark"}[t.Scheme]
}

func (s *site) open(page int) {
	if err := s.parse(page); err != nil {
		panic(err)
	}
	s.page, s.section, s.folds[s.entries[page].group].Open = page, 0, true
	s.rt.Invalidate()
}

func el(class string, children ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{twi.Class(class)}, children...)...)
}

func txt(class, s string) twi.Node { return el(class, twi.Text(s)) }

func (s *site) view() twi.Node {
	s.remember()
	e := s.entries[s.page]
	return el("flex flex-col h-full bg-background text-foreground",
		twi.OnKeyDown(func(ev *twi.Event) {
			k := ev.Key
			if k.Key != input.KeyRune || k.Modifiers != 0 || s.palette.Open || s.picker {
				return
			}
			switch k.Rune {
			case 't':
				s.openPicker()
			case 'm':
				s.flipScheme()
			}
		}),
		el("flex flex-row shrink-0 items-center gap-2 border-b px-2 pt-1",
			txt("font-bold", "twind"),
			txt("text-muted-foreground", "Docs"),
			el("grow"),
			s.palette.Trigger(ui.Outline, ui.SizeSM, twi.Key("search"), twi.Class("md:w-40 justify-between text-muted-foreground"),
				txt("md:hidden", "Search"), txt("hidden md:flex", "Search documentation..."),
				ui.KbdGroup(twi.Class("hidden md:flex"), ui.Kbd(twi.Text("Ctrl")), ui.Kbd(twi.Text("K")))),
			ui.Button(ui.Ghost, ui.SizeSM, twi.Key("theme"), twi.OnClick(func(*twi.Event) { s.openPicker() }),
				txt("hidden md:flex", "Theme: "), twi.Text(themeName(s.themes[s.theme].WithScheme(s.scheme)))),
			ui.Button(ui.Ghost, ui.SizeSM, twi.Key("scheme"), twi.OnClick(func(*twi.Event) { s.flipScheme() }),
				twi.Text(string(map[theme.Scheme]icon.Name{theme.Light: icon.Sun, theme.Dark: icon.Moon}[s.scheme].Glyph()))),
		),
		el("flex flex-row flex-1 min-h-0", s.sidebar.Provider(s.nav(), ui.SidebarInset(el("flex flex-row flex-1 min-h-0", s.content(e), s.outline(e))))),
		s.search(),
		s.pickerNode(),
	)
}

func (s *site) nav() twi.Node {
	var groups []twi.NodeOption
	for i := 0; i < len(s.entries); {
		group, fold := s.entries[i].group, s.folds[s.entries[i].group]
		var items []twi.NodeOption
		for ; i < len(s.entries) && s.entries[i].group == group; i++ {
			at, e := i, s.entries[i]
			if fold.Open {
				items = append(items, ui.SidebarMenuButton(ui.SizeDefault, i == s.page,
					twi.Key("nav-"+e.slug), twi.OnClick(func(*twi.Event) { s.open(at) }), twi.Text(e.title),
					twi.OnFocus(func() { s.focusRow(at) }), twi.OnBlur(func() {
						if !s.palette.Open {
							s.heldRow = -1
						}
					})))
			}
		}
		chevron := map[bool]string{true: "⌄", false: "›"}[fold.Open]
		groups = append(groups, ui.SidebarGroup(
			fold.Trigger(ui.Ghost, ui.SizeSM, twi.Key("group-"+group), twi.Class("justify-between px-1 text-sidebar-foreground/70"), twi.Text(group), twi.Text(chevron)),
			fold.Content(ui.SidebarMenu(items...)),
		))
	}
	return s.sidebar.Node(ui.SidebarContent(append(groups, twi.Class("py-1"))...))
}

func (s *site) focusRow(at int) {
	s.heldRow = at
	if at != s.returnRow {
		return
	}
	s.returnRow = -1
	if at != s.page {
		s.rt.Focus("nav-" + s.entries[s.page].slug)
	}
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
	return ui.ScrollArea(twi.Key("page-"+e.slug), twi.Class("flex-1 min-w-0 px-3 py-1"), twi.Measure(s.area), twi.OnScroll(func(image.Point) { s.track(e) }),
		el("flex flex-col shrink-0 gap-1", twi.Measure(s.body),
			ui.Breadcrumb(ui.BreadcrumbList(
				ui.BreadcrumbItem(ui.BreadcrumbLink(twi.Text("Docs"))), ui.BreadcrumbSeparator(),
				ui.BreadcrumbItem(ui.BreadcrumbLink(twi.Text(e.group))), ui.BreadcrumbSeparator(),
				ui.BreadcrumbItem(ui.BreadcrumbPage(twi.Text(e.title))),
			)),
			s.sections(e),
			el("flex flex-row justify-between pt-1", pager...),
		),
	)
}

func (s *site) follow(target string) {
	if i := slices.IndexFunc(s.entries, func(e entry) bool { return e.slug+".md" == target }); i >= 0 {
		s.open(i)
	}
}

func (s *site) sections(e entry) twi.Node {
	options := markdown.Options{Highlight: s.code, Follow: s.follow, Copy: s.copy, Copied: s.copied}
	var out []twi.NodeOption
	for blocks := e.page.Blocks; len(blocks) > 0; {
		end := 1 + slices.IndexFunc(blocks[1:], func(b markdown.Block) bool { return b.Kind == markdown.Heading && outlined(b.Level) })
		if end == 0 {
			end = len(blocks)
		}
		section := []twi.NodeOption{markdown.Render(markdown.Page{Blocks: blocks[:end]}, options)}
		if b := blocks[0]; b.Kind == markdown.Heading && outlined(b.Level) {
			if s.heads[b.ID] == nil {
				s.heads[b.ID] = twi.NewRef(s.rt)
			}
			section = append(section, twi.Key("section-"+b.ID), twi.Measure(s.heads[b.ID]))
			if b.Level == 2 {
				section = append(section, twi.Class("not-first:mt-1"))
			}
		}
		out, blocks = append(out, el("flex flex-col", section...)), blocks[end:]
	}
	return el("flex flex-col gap-1", out...)
}

func outlined(level int) bool { return level == 2 || level == 3 }

func anchors(e entry) []markdown.Anchor {
	return slices.DeleteFunc(slices.Clone(e.page.Anchors), func(a markdown.Anchor) bool { return !outlined(a.Level) })
}

func (s *site) track(e entry) {
	current, area := 0, s.area.Bounds()
	for i, a := range anchors(e) {
		if head := s.heads[a.ID]; head != nil && !head.Bounds().Empty() && head.Bounds().Min.Y <= area.Min.Y {
			current = i
		}
	}
	clickedBelowTheLastTop := s.section > current && s.body.Bounds().Max.Y < area.Max.Y
	if current != s.section && !clickedBelowTheLastTop {
		s.section = current
		s.rt.Invalidate()
	}
}

func (s *site) outline(e entry) twi.Node {
	items := []twi.NodeOption{txt("font-medium", "On this page")}
	for i, a := range anchors(e) {
		class := "text-muted-foreground"
		if i == s.section {
			class = "text-foreground"
		}
		if a.Level == 3 {
			class += " pl-2"
		}
		items = append(items, twi.Element(twi.Key("toc-"+a.ID), twi.Focusable(), twi.Class("rounded-sm hover:text-foreground focus-visible:shadow-[0_0_0_1px_var(--color-ring)]", class), twi.Text(a.Text), twi.OnClick(func(*twi.Event) {
			s.section = i
			s.rt.ScrollIntoView("section-" + a.ID)
			s.rt.Invalidate()
		})))
	}
	return el("hidden md:flex flex-col w-24 shrink-0 px-2 py-1", items...)
}

func (s *site) search() twi.Node {
	p := s.palette
	if s.commands == nil {
		for i := 0; i < len(s.entries); {
			group := s.entries[i].group
			var items []ui.CommandItem
			for ; i < len(s.entries) && s.entries[i].group == group; i++ {
				items = append(items, p.Item(s.entries[i].title))
			}
			s.commands = append(s.commands, p.Group(group, items...))
		}
		var themes []ui.CommandItem
		for _, t := range s.themes {
			themes = append(themes, p.Item(t.Name))
		}
		s.commands = append(s.commands, p.Group("Theme", themes...))
	}
	return p.Node(p.Input("Search documentation..."), p.List(s.commands...))
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
		s.show(s.trying)
	}
	restore := func() {
		s.picker = false
		s.show(s.theme)
	}
	apply := func() {
		s.theme = s.trying
		restore()
	}
	list := []twi.NodeOption{twi.Key("themes"), twi.Focusable(), twi.Class("flex flex-col"), twi.OnKeyDown(func(ev *twi.Event) {
		switch ev.Key.Key {
		case input.KeyArrowDown:
			show(s.trying + 1)
		case input.KeyArrowUp:
			show(s.trying - 1)
		case input.KeyEnter:
			apply()
		}
	})}
	for i, t := range s.themes {
		class, mark := "px-1", "  "
		if i == s.trying {
			class = "px-1 bg-accent text-accent-foreground font-medium"
		}
		if i == s.theme {
			mark = "● "
		}
		list = append(list, twi.Element(twi.Class(class), twi.Text(mark+t.Name),
			twi.OnPointerEnter(func() { show(i) }), twi.OnClick(func(*twi.Event) { apply() })))
	}
	return twi.Element(twi.Key("picker"), twi.FocusScope(), twi.Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
		twi.OnKeyDown(func(ev *twi.Event) {
			if ev.Key.Key == input.KeyEscape {
				restore()
			}
			ev.StopPropagation()
		}),
		el("flex flex-col w-40 rounded-lg border bg-popover text-popover-foreground shadow-lg",
			twi.OnPointerDownOutside(restore),
			txt("px-1 font-semibold", "Theme"),
			txt("px-1 text-muted-foreground", "↑ ↓ preview, Enter keeps, Esc restores"),
			ui.DropdownMenuSeparator(),
			twi.Element(list...),
		),
	)
}
