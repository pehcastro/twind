package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pehcastro/twind/internal/edit"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/icon"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/text"
	"github.com/pehcastro/twind/twi/theme"
	"github.com/pehcastro/twind/twi/ui"
)

//go:generate go run github.com/pehcastro/twind/internal/twirgen

const (
	listRows         = 40
	cardDelay        = 700 * time.Millisecond
	cardShown        = 3 * time.Second
	loadFor          = 2 * time.Second
	shortestUsername = 2
	shortestPassword = 8
	longestBio       = 160
	focusRing        = " focus-visible:shadow-[0_0_0_1px_var(--color-ring)]"
	pill             = "shrink-0 rounded-full px-1 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
	activePill       = " data-[state=active]:bg-primary data-[state=active]:text-primary-foreground data-[state=active]:font-bold"
)

type Env struct {
	Cwd, Profile string
	Size         func() string
	Today        time.Time
}

type Start struct {
	Page, Theme  string
	Picker       bool
	Focus, Value string
	Open         string
	Slow         bool
}

type state struct {
	page, count   int
	theme, cursor int
	scheme        theme.Scheme
	picker        bool
	card, loading bool
	focus         string
}

type controls struct {
	state   state
	update  func(func(*state))
	command func(*state, string)
	auto    string
	kit     *kit
}

func (c controls) focusable(name string) []twi.NodeOption {
	opts := []twi.NodeOption{
		twi.Key(name),
		twi.Focusable(),
		twi.OnFocus(func() { c.update(func(s *state) { s.focus = name }) }),
		twi.OnBlur(func() {
			c.update(func(s *state) {
				if s.focus == name {
					s.focus = ""
				}
			})
		}),
	}
	if name == c.auto {
		opts = append(opts, twi.AutoFocus())
	}
	return opts
}

func (c controls) pressable(name string, press func(*state)) []twi.NodeOption {
	return append(c.focusable(name), twi.OnKeyDown(func(e *twi.Event) {
		if e.Key.Key == input.KeyEnter || e.Key.Key == input.KeyRune && e.Key.Rune == ' ' {
			c.update(press)
		}
	}), twi.OnClick(func(*twi.Event) { c.update(press) }))
}

func (c controls) button(name, class, label string, press func(*state), extra ...twi.NodeOption) twi.Node {
	return twi.Element(append(append(c.pressable(name, press), extra...), twi.Class(class+focusRing), twi.Text(label))...)
}

func (c controls) uiButton(name string, v ui.Variant, label string, press func(*state), extra ...twi.NodeOption) twi.Node {
	return ui.Button(v, ui.SizeDefault, append(append(c.pressable(name, press), extra...), twi.Text(label))...)
}

func (c controls) clicked(change func()) twi.NodeOption {
	return twi.OnClick(func(*twi.Event) { c.update(func(*state) { change() }) })
}

type key struct{ press, what string }

type page struct {
	name string
	view func(controls) twi.Node
	keys []key
}

func basics() []page {
	return []page{
		{"surfaces", surfaces, nil},
		{"text", textPage, nil},
		{"layout", layoutPage, nil},
		{"counter", counterPage, []key{{"+", "increment"}, {"-", "decrement"}}},
		{"list", listPage, []key{{"wheel", "scroll"}, {"pgdn", "page"}, {"home", "top"}}},
		{"motion", motionPage, []key{{"o", "menu"}, {"r", "reorder"}, {"d", "dialog"}, {"s", "sheet"}, {"n", "toast"}, {"l", "load"}, {"esc", "close"}}},
		{"selection", selectionPage, []key{{"drag", "select"}, {"2" + nbsp + "clicks", "word"}, {"3" + nbsp + "clicks", "line"}, {"ctrl+c", "copy"}, {"esc", "clear"}}},
	}
}

func pages() []page { return append(basics(), components()...) }

func themeName(t theme.Theme) string {
	return t.Name + map[theme.Scheme]string{theme.Light: "-light", theme.Dark: "-dark"}[t.Scheme]
}

func themeList() []theme.Theme {
	return slices.DeleteFunc(theme.Builtin(), func(t theme.Theme) bool { return t.Scheme == theme.Dark })
}

func pickTheme(name string) (state, bool) {
	for i, t := range themeList() {
		for _, s := range []theme.Scheme{theme.Light, theme.Dark} {
			if themeName(t.WithScheme(s)) == name {
				return state{theme: i, scheme: s}, true
			}
		}
	}
	return state{}, false
}

func flipScheme(s *state) {
	s.scheme = map[theme.Scheme]theme.Scheme{theme.Light: theme.Dark, theme.Dark: theme.Light}[s.scheme]
}

func el(class string, children ...twi.Node) twi.Node {
	opts := []twi.NodeOption{twi.Class(class)}
	for _, c := range children {
		opts = append(opts, c)
	}
	return twi.Element(opts...)
}

func txt(class, s string) twi.Node { return el(class, twi.Text(s)) }

func App(rt *twi.Runtime) func() twi.Node {
	env := Env{Cwd: "/home/user/twind", Profile: "truecolor", Size: func() string { return "headless" }, Today: time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC)}
	s, _ := pickTheme("twind-dark")
	return playground(rt, env, s, Start{Focus: "input"})
}

func New(rt *twi.Runtime, env Env, start Start) (func() twi.Node, error) {
	s, ok := pickTheme(start.Theme)
	if !ok {
		return nil, fmt.Errorf("theme %q: not a built-in theme", start.Theme)
	}
	s.page, s.picker = slices.IndexFunc(pages(), func(p page) bool { return p.name == start.Page }), start.Picker
	if n, err := strconv.Atoi(start.Page); err == nil && n >= 1 && n <= len(basics()) {
		s.page = n - 1
	}
	if s.page < 0 {
		return nil, fmt.Errorf("page %q: want 1 to %d or a page name", start.Page, len(basics()))
	}
	if !slices.Contains([]string{"", "dialog", "sheet", "spinner", "combobox", "navigation"}, start.Open) {
		return nil, fmt.Errorf("open %q: want dialog, sheet, spinner, combobox or navigation", start.Open)
	}
	return playground(rt, env, s, start), nil
}

func playground(rt *twi.Runtime, env Env, start state, opening Start) func() twi.Node {
	themes := themeList()
	all := pages()
	bar := len(basics())
	counter := slices.IndexFunc(all, func(p page) bool { return p.name == "counter" })
	alphabetical := make([]int, len(all))
	for i := range alphabetical {
		alphabetical[i] = i
	}
	slices.SortFunc(alphabetical, func(a, b int) int { return strings.Compare(all[a].name, all[b].name) })
	start.cursor = start.theme
	st := twi.NewSignal(rt, start)
	shown := themes[start.theme].WithScheme(start.scheme)
	rt.SetTheme(shown)
	update := func(change func(*state)) {
		s := st.Get()
		change(&s)
		st.Set(s)
		want := themes[s.theme]
		if s.picker {
			want = themes[s.cursor]
		}
		if want = want.WithScheme(s.scheme); want.Name != shown.Name || want.Scheme != shown.Scheme {
			shown = want
			rt.SetTheme(want)
		}
	}
	k := newKit(rt, env.Today)
	k.dialog.Open, k.sheet.Open = opening.Open == "dialog", opening.Open == "sheet"
	k.spinning, k.framework.Open = opening.Open == "spinner", opening.Open == "combobox"
	if opening.Open == "navigation" {
		k.nav.Value = "components"
	}
	k.palette.OnSelect = func(name string) {
		update(func(s *state) { s.page = slices.IndexFunc(all, func(p page) bool { return p.name == name }) })
	}
	var pace []twi.NodeOption
	if opening.Slow {
		pace = append(pace, twi.Class("duration-[8s]"))
	}
	openPicker := func(s *state) { s.picker, s.cursor = true, s.theme }
	var later, loaded *twi.Timer
	command := func(s *state, cmd string) {
		n, err := strconv.Atoi(cmd)
		switch {
		case err == nil && n >= 1 && n <= bar:
			s.page = n - 1
		case cmd == "t" || cmd == "theme":
			openPicker(s)
		case cmd == "m" || cmd == "scheme":
			flipScheme(s)
		case cmd == "+" || cmd == "increment":
			s.page, s.count = counter, s.count+1
		case cmd == "-" || cmd == "decrement":
			s.page, s.count = counter, max(s.count-1, 0)
		case cmd == "later":
			if later != nil {
				later.Stop()
			}
			s.card = false
			later = rt.After(cardDelay, func() {
				update(func(s *state) { s.card = true })
				later = rt.After(cardShown, func() { update(func(s *state) { s.card = false }) })
			})
		case cmd == "reply":
			k.sent++
		case cmd == "dialog":
			k.dialog.Open = true
		case cmd == "sheet":
			k.sheet.Open = true
		case cmd == "menu":
			k.moves.toggle()
		case cmd == "reorder":
			k.moves.reorder()
		case cmd == "toast":
			k.toaster.Show("Saved to the playground", "leaves after 4 s, a hover holds it", ui.ToastAction{})
		case cmd == "load":
			if loaded != nil {
				loaded.Stop()
			}
			s.loading = true
			loaded = rt.After(loadFor, func() { update(func(s *state) { s.loading = false }) })
		case cmd == "q" || cmd == "quit":
			rt.Quit()
		default:
			if i := slices.IndexFunc(all, func(p page) bool { return p.name == cmd }); i >= 0 {
				s.page = i
			}
		}
	}
	global := []key{{"1-" + strconv.Itoa(bar), "page"}, {"ctrl+k", "components"}, {"t", "theme"}, {"m", "scheme"}, {"q", "quit"}}
	field := twi.NewInput(rt)
	field.Insert(opening.Value)
	field.Placeholder = "Ask twind: a page, theme, dialog, toast, load, later or quit"
	field.CursorClass, field.SelectionClass, field.PlaceholderClass = "bg-foreground text-background", "bg-primary text-primary-foreground", "text-muted-foreground"
	return func() twi.Node {
		c := controls{state: st.Get(), update: update, command: command, auto: opening.Focus, kit: k}
		s := c.state
		name := themeName(themes[s.theme].WithScheme(s.scheme))
		keys := slices.Concat(global, all[s.page].keys)
		tabs := []twi.Node{
			txt("shrink-0 rounded-full px-1 bg-linear-to-r from-primary-600 to-primary-400 text-primary-foreground font-bold", "twind"),
			txt("shrink h-1 overflow-hidden px-1 text-muted-foreground", text.Truncate(filepath.ToSlash(env.Cwd), 40)),
		}
		active := func(on bool) twi.NodeOption {
			return twi.Data("state", map[bool]string{true: "active", false: "inactive"}[on])
		}
		for i, p := range all[:bar] {
			tabs = append(tabs, c.button(p.name, pill+activePill, p.name, func(s *state) { s.page = i }, active(i == s.page)))
		}
		tabs = append(tabs,
			c.button("components", pill+activePill, "⌕ ui", func(*state) { k.palette.Open = true }, active(s.page >= bar)),
			el("grow"),
			k.tip.Node(twi.Class("shrink-0"),
				k.tip.Trigger(ui.Ghost, ui.SizeXS, append(c.pressable("theme", openPicker), twi.Class("rounded-full text-muted-foreground"), twi.Text(name))...),
				k.tip.Content(twi.Text("t or a click opens the picker")),
			),
			c.button("scheme", pill, string(map[theme.Scheme]icon.Name{theme.Light: icon.Sun, theme.Dark: icon.Moon}[s.scheme].Glyph()), flipScheme),
		)
		var hints []twi.Node
		for _, h := range keys {
			hints = append(hints, el("flex flex-row gap-1", ui.Kbd(twi.Text(h.press)), twi.Text(h.what)))
		}
		var goTo []ui.CommandItem
		for _, i := range alphabetical {
			item := []twi.NodeOption{twi.Text(all[i].name)}
			if i < bar {
				item = append(item, ui.CommandShortcut(twi.Text(strconv.Itoa(i+1))))
			}
			goTo = append(goTo, k.palette.Item(all[i].name, item...))
		}
		root := []twi.NodeOption{
			twi.Class("flex flex-col h-full bg-background text-foreground"),
			twi.OnKeyDown(func(e *twi.Event) {
				r := e.Key.Rune
				if e.Key.Key != input.KeyRune || e.Key.Modifiers != 0 || k.dialog.Open || k.sheet.Open || k.alert.Open || k.drawer.Open || k.palette.Open {
					return
				}
				if r >= '1' && r <= '9' {
					update(func(s *state) { command(s, string(r)) })
				}
				if i := slices.IndexFunc(keys, func(k key) bool { return k.press == string(r) }); i >= 0 {
					update(func(s *state) { command(s, keys[i].what) })
				}
			}),
			el("flex flex-row items-center gap-1 px-2 pt-1", tabs...),
			twi.Element(twi.Key(all[s.page].name), twi.Class("grow relative overflow-hidden flex flex-col justify-center px-3"), all[s.page].view(c)),
			el("flex flex-row gap-2 px-3 text-muted-foreground", hints...),
			el("mx-2 flex flex-row items-center gap-1 px-1 border rounded-lg bg-card text-muted-foreground focus-within:border-ring",
				txt("text-primary font-bold", "▌"),
				field.Node(append(c.focusable("input"),
					twi.Class("grow h-1 overflow-hidden flex flex-row text-foreground"),
					twi.OnKeyDown(func(e *twi.Event) {
						if e.Key.Key == input.KeyEnter {
							cmd := strings.ToLower(strings.TrimSpace(field.Value()))
							field.Buffer = edit.Buffer{}
							update(func(s *state) { command(s, cmd) })
						}
					}))...),
				txt("shrink-0 text-foreground", "tab"), txt("shrink-0", "focus"),
				txt("shrink-0 pl-1 text-foreground", "enter"), txt("shrink-0", "run"),
			),
			el("flex flex-row gap-2 px-3 text-muted-foreground",
				txt("text-primary", "● fullscreen"),
				twi.Text(env.Size()),
				twi.Text(env.Profile),
				twi.Text(name),
				el("grow"),
				twi.Text("focus "+s.focus),
				twi.Text(all[s.page].name),
			),
			k.dialog.Content(append(pace,
				k.dialog.Header(k.dialog.Title(twi.Text("Motion")), k.dialog.Description(twi.Text("fade-in-0 and zoom-in-95 over 200 ms; Escape, a click outside or a button plays it back out"))),
				k.dialog.Footer(k.dialog.Close(ui.Outline, ui.SizeDefault, twi.Text("Cancel")), k.dialog.Close(ui.Default, ui.SizeDefault, twi.Text("Done"))),
			)...),
			k.sheet.Content(
				k.sheet.Header(k.sheet.Title(twi.Text("Sheet")), k.sheet.Description(twi.Text("slides in from the right over 500 ms and out over 300 ms"))),
				k.sheet.Footer(k.sheet.Close(ui.Default, ui.SizeDefault, twi.Text("Close"))),
			),
			k.palette.Node(k.palette.Input("Search pages and components"), k.palette.List(k.palette.Group("Go to", goTo...))),
			k.toaster.Node(),
		}
		if s.card {
			root = append(root, el("fixed inset-0 z-50 flex items-center justify-center",
				el("w-44 flex flex-col px-1 border rounded-lg shadow-lg bg-popover text-popover-foreground",
					txt("font-bold", "Later"),
					txt("text-muted-foreground", "shown 700 ms after Enter, gone 3 s later"),
				),
			))
		}
		if s.picker {
			root = append(root, picker(themes, c, func(s *state) { s.theme, s.picker = s.cursor, false }))
		}
		return twi.Element(root...)
	}
}

func picker(themes []theme.Theme, c controls, apply func(*state)) twi.Node {
	s := c.state
	list := append(c.focusable("themes"), twi.Class("flex flex-col rounded-md"+focusRing), twi.OnKeyDown(func(e *twi.Event) {
		step := map[input.Key]int{input.KeyArrowDown: 1, input.KeyArrowUp: -1}[e.Key.Key]
		switch {
		case step != 0:
			c.update(func(s *state) { s.cursor = (s.cursor + step + len(themes)) % len(themes) })
		case e.Key.Key == input.KeyEnter:
			c.update(apply)
		}
	}))
	for i, t := range themes {
		class, mark := "px-1", "  "
		if i == s.cursor {
			class = "px-1 bg-accent text-accent-foreground font-bold"
		}
		if i == s.theme {
			mark = "● "
		}
		list = append(list, twi.Element(twi.Class(class), twi.Text(mark+t.Name),
			twi.OnPointerEnter(func() { c.update(func(s *state) { s.cursor = i }) }),
			twi.OnClick(func(*twi.Event) { c.update(apply) }),
		))
	}
	closePicker := func(s *state) { s.picker = false }
	return twi.Element(twi.Key("picker"), twi.FocusScope(), twi.Class("fixed inset-0 z-50 bg-black/50 flex items-center justify-center"),
		twi.OnKeyDown(func(e *twi.Event) {
			if e.Key.Key == input.KeyEscape {
				c.update(closePicker)
			}
			e.StopPropagation()
		}),
		twi.Element(twi.Class("w-44 flex flex-col gap-1 px-1 border rounded-lg shadow-lg bg-popover text-popover-foreground"),
			twi.OnPointerDownOutside(func() { c.update(closePicker) }),
			el("flex flex-col",
				txt("px-1 font-bold", "Theme"),
				txt("px-1 text-muted-foreground", "↑ ↓ preview, Enter keeps, Esc restores"),
			),
			twi.Element(list...),
			c.button("close", "self-end rounded-full px-1 bg-secondary text-secondary-foreground hover:bg-secondary/80", "close", closePicker),
		),
	)
}
