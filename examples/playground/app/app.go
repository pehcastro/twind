package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/edit"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/text"
	"github.com/twind-dev/twind/twi/theme"
	"github.com/twind-dev/twind/twi/ui"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen

const (
	pickerRows = 9
	listRows   = 40
	cardDelay  = 700 * time.Millisecond
	cardShown  = 3 * time.Second
	loadFor    = 2 * time.Second
	focusRing  = " focus-visible:shadow-[0_0_0_1px_var(--color-ring)]"
	pill       = "shrink-0 rounded-full px-1 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
)

type Env struct {
	Cwd, Profile string
	Size         func() string
}

type Start struct {
	Page         int
	Theme        string
	Picker       bool
	Focus, Value string
	Open         string
	Slow         bool
}

type state struct {
	page, count   int
	theme, cursor int
	picker, tip   bool
	card, loading bool
	focus         string
}

type controls struct {
	state   state
	update  func(func(*state))
	command func(*state, string)
	auto    string
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

type key struct{ press, what string }

type page struct {
	name string
	view func(controls) twi.Node
	keys []key
}

func pages() []page {
	return []page{
		{"surfaces", surfaces, nil},
		{"text", textPage, nil},
		{"layout", layoutPage, nil},
		{"counter", counterPage, []key{{"+", "increment"}, {"-", "decrement"}}},
		{"list", listPage, []key{{"wheel", "scroll"}, {"pgdn", "page"}, {"home", "top"}}},
		{"motion", motionPage, []key{{"d", "dialog"}, {"s", "sheet"}, {"n", "toast"}, {"l", "load"}, {"esc", "close"}}},
		{"selection", selectionPage, []key{{"drag", "select"}, {"2" + nbsp + "clicks", "word"}, {"3" + nbsp + "clicks", "line"}, {"ctrl+c", "copy"}, {"esc", "clear"}}},
	}
}

func themeName(t theme.Theme) string {
	return t.Name + map[theme.Scheme]string{theme.Light: "-light", theme.Dark: "-dark"}[t.Scheme]
}

func themeIndex(name string) int {
	return slices.IndexFunc(theme.Builtin(), func(t theme.Theme) bool { return themeName(t) == name })
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
	return playground(rt, Env{Cwd: "/home/user/twind", Profile: "truecolor", Size: func() string { return "headless" }}, state{theme: themeIndex("zinc-dark")}, Start{Focus: "input"})
}

func New(rt *twi.Runtime, env Env, start Start) (func() twi.Node, error) {
	s := state{page: start.Page - 1, theme: themeIndex(start.Theme), picker: start.Picker}
	if s.page < 0 || s.page >= len(pages()) {
		return nil, fmt.Errorf("page %d: want 1 to %d", start.Page, len(pages()))
	}
	if s.theme < 0 {
		return nil, fmt.Errorf("theme %q: not a built-in theme", start.Theme)
	}
	if !slices.Contains([]string{"", "dialog", "sheet"}, start.Open) {
		return nil, fmt.Errorf("open %q: want dialog or sheet", start.Open)
	}
	return playground(rt, env, s, start), nil
}

func playground(rt *twi.Runtime, env Env, start state, opening Start) func() twi.Node {
	themes := theme.Builtin()
	all := pages()
	counter := slices.IndexFunc(all, func(p page) bool { return p.name == "counter" })
	start.cursor = start.theme
	st := twi.NewSignal(rt, start)
	rt.SetTheme(themes[start.theme])
	update := func(change func(*state)) {
		s := st.Get()
		change(&s)
		st.Set(s)
	}
	dialog, sheet, toaster := ui.NewDialog(rt), ui.NewSheet(rt, ui.Right), ui.NewToaster(rt)
	dialog.Open, sheet.Open = opening.Open == "dialog", opening.Open == "sheet"
	var pace []twi.NodeOption
	if opening.Slow {
		pace = append(pace, twi.Class("duration-[8s]"))
	}
	openPicker := func(s *state) { s.picker, s.cursor = true, s.theme }
	var later, loaded *twi.Timer
	command := func(s *state, cmd string) {
		n, err := strconv.Atoi(cmd)
		switch {
		case err == nil && n >= 1 && n <= len(all):
			s.page = n - 1
		case cmd == "t" || cmd == "theme":
			openPicker(s)
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
		case cmd == "dialog":
			dialog.Open = true
		case cmd == "sheet":
			sheet.Open = true
		case cmd == "toast":
			toaster.Show("Saved to the playground", "leaves after 4 s, a hover holds it", ui.ToastAction{})
		case cmd == "load":
			if loaded != nil {
				loaded.Stop()
			}
			s.loading = true
			loaded = rt.After(loadFor, func() { update(func(s *state) { s.loading = false }) })
		case cmd == "q" || cmd == "quit":
			rt.Quit()
		default:
			for i, p := range all {
				if p.name == cmd {
					s.page = i
				}
			}
		}
	}
	global := []key{{"1-" + strconv.Itoa(len(all)), "page"}, {"t", "theme"}, {"q", "quit"}}
	field := twi.NewInput(rt)
	field.Insert(opening.Value)
	field.Placeholder = "Ask twind: a page, theme, dialog, toast, load, later or quit"
	field.CursorClass, field.SelectionClass, field.PlaceholderClass = "bg-foreground text-background", "bg-primary text-primary-foreground", "text-muted-foreground"
	return func() twi.Node {
		c := controls{state: st.Get(), update: update, command: command, auto: opening.Focus}
		s := c.state
		name := themeName(themes[s.theme])
		keys := slices.Concat(global, all[s.page].keys)
		tabs := []twi.Node{
			txt("shrink-0 rounded-full px-1 bg-emerald-400 text-emerald-950 font-bold", "twind"),
			txt("shrink h-1 overflow-hidden px-1 text-muted-foreground", text.Truncate(filepath.ToSlash(env.Cwd), 40)),
		}
		for i, p := range all {
			tab := twi.Data("state", map[bool]string{true: "active", false: "inactive"}[i == s.page])
			tabs = append(tabs, c.button(p.name, pill+" data-[state=active]:bg-primary data-[state=active]:text-primary-foreground data-[state=active]:font-bold", p.name, func(s *state) { s.page = i }, tab))
		}
		bar := "mx-2 flex flex-row items-center gap-1 px-1 border rounded-lg bg-card text-muted-foreground focus-within:border-ring"
		themeButton := []twi.NodeOption{c.button("theme", pill, "theme "+name, openPicker)}
		if s.tip {
			themeButton = append(themeButton, txt("absolute top-full right-0 z-50 whitespace-nowrap rounded-md px-1 bg-foreground text-background", "t or a click opens the picker"))
		}
		tabs = append(tabs, el("grow"), twi.Element(append(themeButton, twi.Class("relative flex shrink-0"),
			twi.OnPointerEnter(func() { update(func(s *state) { s.tip = true }) }),
			twi.OnPointerLeave(func() { update(func(s *state) { s.tip = false }) }),
		)...))
		var hints []twi.Node
		for _, k := range keys {
			hints = append(hints, el("flex flex-row gap-1", ui.Kbd(twi.Text(k.press)), twi.Text(k.what)))
		}
		root := []twi.NodeOption{
			twi.Class("flex flex-col h-full bg-background text-foreground"),
			twi.OnKeyDown(func(e *twi.Event) {
				r := e.Key.Rune
				if e.Key.Key != input.KeyRune || e.Key.Modifiers != 0 || dialog.Open || sheet.Open {
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
			el(bar,
				txt("text-emerald-400 font-bold", "▌"),
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
			el("flex flex-row gap-3 px-3 text-muted-foreground",
				txt("text-emerald-400", "● fullscreen"),
				twi.Text(env.Size()),
				twi.Text(env.Profile),
				twi.Text(name),
				el("grow"),
				twi.Text("focus "+s.focus),
				twi.Text(all[s.page].name),
			),
			dialog.Content(append(pace,
				dialog.Header(dialog.Title(twi.Text("Motion")), dialog.Description(twi.Text("fade-in-0 and zoom-in-95 over 200 ms; Escape, a click outside or a button plays it back out"))),
				dialog.Footer(dialog.Close(ui.Outline, ui.SizeDefault, twi.Text("Cancel")), dialog.Close(ui.Default, ui.SizeDefault, twi.Text("Done"))),
			)...),
			sheet.Content(
				sheet.Header(sheet.Title(twi.Text("Sheet")), sheet.Description(twi.Text("slides in from the right over 500 ms and out over 300 ms"))),
				sheet.Footer(sheet.Close(ui.Default, ui.SizeDefault, twi.Text("Close"))),
			),
			toaster.Node(),
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
			root = append(root, picker(themes, c, func(s *state) {
				s.theme, s.picker = s.cursor, false
				rt.SetTheme(themes[s.theme])
			}))
		}
		return twi.Element(root...)
	}
}

func picker(themes []theme.Theme, c controls, apply func(*state)) twi.Node {
	s := c.state
	first := min(max(s.cursor-pickerRows/2, 0), len(themes)-pickerRows)
	list := append(c.focusable("themes"), twi.Class("flex flex-col rounded-md"+focusRing), twi.OnKeyDown(func(e *twi.Event) {
		step := map[input.Key]int{input.KeyArrowDown: 1, input.KeyArrowUp: -1}[e.Key.Key]
		switch {
		case step != 0:
			c.update(func(s *state) { s.cursor = (s.cursor + step + len(themes)) % len(themes) })
		case e.Key.Key == input.KeyEnter:
			c.update(apply)
		}
	}))
	for i := first; i < first+pickerRows; i++ {
		class, mark := "px-1", "  "
		if i == s.cursor {
			class = "px-1 bg-accent text-accent-foreground font-bold"
		}
		if i == s.theme {
			mark = "● "
		}
		list = append(list, twi.Element(twi.Class(class), twi.Text(mark+themeName(themes[i])),
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
				txt("px-1 text-muted-foreground", "↑ ↓ move, Enter applies, Tab, Esc closes"),
			),
			twi.Element(list...),
			c.button("close", "self-end rounded-full px-1 bg-secondary text-secondary-foreground hover:bg-secondary/80", "close", closePicker),
		),
	)
}
