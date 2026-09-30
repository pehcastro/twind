package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/edit"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/text"
	"github.com/twind-dev/twind/twi/theme"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen

const (
	pickerRows = 9
	listRows   = 40
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
}

type state struct {
	page, count   int
	theme, cursor int
	picker, tip   bool
	focus         string
}

type controls struct {
	state  state
	update func(func(*state))
	auto   string
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

func (c controls) button(name, class, label string, press func(*state), extra ...twi.NodeOption) twi.Node {
	return twi.Element(append(append(c.focusable(name), extra...), twi.Class(class+focusRing), twi.Text(label), twi.OnKeyDown(func(e *twi.Event) {
		if e.Key.Key == input.KeyEnter || e.Key.Key == input.KeyRune && e.Key.Rune == ' ' {
			c.update(press)
		}
	}), twi.OnClick(func(*twi.Event) { c.update(press) }))...)
}

type page struct {
	name string
	view func(controls) twi.Node
}

func pages() []page {
	return []page{{"surfaces", surfaces}, {"text", textPage}, {"layout", layoutPage}, {"counter", counterPage}, {"list", listPage}}
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
	return playground(rt, Env{Cwd: "/home/user/twind", Profile: "truecolor", Size: func() string { return "headless" }}, state{theme: themeIndex("zinc-dark")}, "input", "")
}

func New(rt *twi.Runtime, env Env, start Start) (func() twi.Node, error) {
	s := state{page: start.Page - 1, theme: themeIndex(start.Theme), picker: start.Picker}
	if s.page < 0 || s.page >= len(pages()) {
		return nil, fmt.Errorf("page %d: want 1 to %d", start.Page, len(pages()))
	}
	if s.theme < 0 {
		return nil, fmt.Errorf("theme %q: not a built-in theme", start.Theme)
	}
	return playground(rt, env, s, start.Focus, start.Value), nil
}

func playground(rt *twi.Runtime, env Env, start state, auto, value string) func() twi.Node {
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
	openPicker := func(s *state) { s.picker, s.cursor = true, s.theme }
	command := func(s *state, cmd string) {
		n, err := strconv.Atoi(cmd)
		switch {
		case err == nil && n >= 1 && n <= len(all):
			s.page = n - 1
		case cmd == "t" || cmd == "theme":
			openPicker(s)
		case cmd == "+":
			s.page, s.count = counter, s.count+1
		case cmd == "-":
			s.page, s.count = counter, max(s.count-1, 0)
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
	field := twi.NewInput(rt)
	field.Insert(value)
	field.Placeholder = "Ask twind: a page name or number, theme, +, - or quit"
	field.CursorClass, field.SelectionClass, field.PlaceholderClass = "bg-foreground text-background", "bg-primary text-primary-foreground", "text-muted-foreground"
	return func() twi.Node {
		c := controls{state: st.Get(), update: update, auto: auto}
		s := c.state
		name := themeName(themes[s.theme])
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
		root := []twi.NodeOption{
			twi.Class("flex flex-col h-full bg-background text-foreground"),
			twi.OnKeyDown(func(e *twi.Event) {
				r := e.Key.Rune
				shortcut := strings.ContainsRune("123456789tq", r) || strings.ContainsRune("+-", r) && all[s.page].name == "counter"
				if e.Key.Key == input.KeyRune && e.Key.Modifiers == 0 && shortcut {
					update(func(s *state) { command(s, string(r)) })
				}
			}),
			el("flex flex-row items-center gap-2 px-2 pt-1", tabs...),
			twi.Element(twi.Key(all[s.page].name), twi.Class("grow relative overflow-hidden flex flex-col justify-center px-3"), all[s.page].view(c)),
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
