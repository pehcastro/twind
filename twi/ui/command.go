package ui

import (
	"slices"
	"strconv"
	"unicode"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/ui"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

type CommandItem struct {
	value    string
	children []twi.NodeOption
}

type CommandGroup struct {
	heading   string
	items     []CommandItem
	separator bool
}

type Command struct {
	Empty              string
	OnSelect           func(string)
	rt                 *twi.Runtime
	input              *twi.Input
	selected, searched string
	visible            []string
	scored, rows       []commandRow
	spans              [][2]int
	offset             int
	chosen             func()
}

func NewCommand(rt *twi.Runtime) *Command {
	in := twi.NewInput(rt)
	in.CursorClass, in.SelectionClass, in.PlaceholderClass = cursorClass, selectionClass, placeholderClass
	return &Command{Empty: "No results found.", rt: rt, input: in}
}

func (c *Command) Search(s string) {
	c.input.Apply(input.KeyEvent{Key: input.KeyRune, Rune: 'a', Modifiers: input.ModCtrl})
	c.input.Insert(s)
	c.rt.Invalidate()
}

func (c *Command) Node(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col w-full overflow-hidden rounded-md bg-popover text-popover-foreground", append([]twi.NodeOption{keyDown(c.rt, c.key)}, children...))
}

func (c *Command) Input(placeholder string) twi.Node {
	c.input.Placeholder = placeholder
	return part("flex flex-row shrink-0 items-center gap-1 border-b px-1", []twi.NodeOption{
		icon("⌕", "shrink-0 opacity-50"),
		c.input.Node(twi.Class("flex flex-row grow h-1 min-w-0 overflow-hidden")),
	})
}

func (c *Command) Item(value string, children ...twi.NodeOption) CommandItem {
	icons := 0
	for _, o := range children {
		if _, ok := o.(ItemIcon); ok {
			icons++
		}
	}
	switch icons {
	case len(children):
		return CommandItem{value, slotted(nil, children, twi.Text(value))}
	case 0:
		return CommandItem{value, children}
	}
	return CommandItem{value, slotted(nil, children)}
}

func (c *Command) Group(heading string, items ...CommandItem) CommandGroup {
	return CommandGroup{heading: heading, items: items}
}

func (c *Command) Separator() CommandGroup { return CommandGroup{separator: true} }

func CommandShortcut(children ...twi.NodeOption) twi.Node {
	return part("grow pl-2 text-right text-muted-foreground", children)
}

type rowKind uint8

const (
	itemRow rowKind = iota
	headingRow
	separatorRow
)

type commandRow struct {
	kind    rowKind
	heading string
	item    *CommandItem
	score   int
}

func (c *Command) List(groups ...CommandGroup) twi.Node {
	search := c.input.Value()
	c.scored, c.spans = c.scored[:0], c.spans[:0]
	for _, g := range groups {
		from := len(c.scored)
		switch {
		case g.separator && search == "":
			c.scored = append(c.scored, commandRow{kind: separatorRow})
		case !g.separator:
			if g.heading != "" {
				c.scored = append(c.scored, commandRow{kind: headingRow, heading: g.heading})
			}
			items := len(c.scored)
			for i := range g.items {
				if s := score(g.items[i].value, search); s > 0 {
					c.scored = append(c.scored, commandRow{kind: itemRow, item: &g.items[i], score: s})
				}
			}
			if len(c.scored) == items {
				c.scored = c.scored[:from]
				continue
			}
			slices.SortStableFunc(c.scored[items:], func(a, b commandRow) int { return b.score - a.score })
			c.scored[from].score = c.scored[items].score
		default:
			continue
		}
		c.spans = append(c.spans, [2]int{from, len(c.scored)})
	}
	if search != "" {
		slices.SortStableFunc(c.spans, func(a, b [2]int) int { return c.scored[b[0]].score - c.scored[a[0]].score })
	}
	c.rows = c.rows[:0]
	for _, s := range c.spans {
		c.rows = append(c.rows, c.scored[s[0]:s[1]]...)
	}
	rows := c.rows
	c.visible = c.visible[:0]
	for _, r := range rows {
		if r.kind == itemRow {
			c.visible = append(c.visible, r.item.value)
		}
	}
	if len(c.visible) == 0 {
		c.selected, c.searched = "", search
		return part("py-1 text-center", []twi.NodeOption{twi.Text(c.Empty)})
	}
	if search != c.searched || !slices.Contains(c.visible, c.selected) {
		c.selected, c.searched, c.offset = c.visible[0], search, 0
	}
	at := slices.IndexFunc(rows, func(r commandRow) bool { return r.kind == itemRow && r.item.value == c.selected })
	if at < c.offset {
		c.offset = at
		if at > 0 && rows[at-1].kind == headingRow {
			c.offset--
		}
	}
	c.offset = min(max(c.offset, at-konst.CommandRows+1), max(len(rows)-konst.CommandRows, 0))
	shown := rows[c.offset:min(c.offset+konst.CommandRows, len(rows))]
	window := make([]twi.NodeOption, 0, len(shown))
	previous := separatorRow
	for _, r := range shown {
		window = append(window, c.row(r, previous))
		previous = r.kind
	}
	return part("flex flex-col shrink-0 gap-1 px-1 pt-1", window)
}

func (c *Command) row(r commandRow, previous rowKind) twi.Node {
	switch r.kind {
	case itemRow:
	case headingRow:
		classes := "px-2 font-medium text-muted-foreground"
		if previous == itemRow {
			classes += " mt-1"
		}
		return part(classes, []twi.NodeOption{twi.Text(r.heading)})
	case separatorRow:
		return part(separator, nil)
	default:
		panic("ui: unknown command row")
	}
	value := r.item.value
	classes := "relative flex flex-row items-center gap-1 rounded-sm px-2 select-none [&_svg]:shrink-0 [&_svg]:pointer-events-none"
	selected := value == c.selected
	if selected {
		classes += " bg-accent text-accent-foreground"
	}
	return part(classes, append([]twi.NodeOption{
		twi.Data("selected", strconv.FormatBool(selected)),
		twi.OnPointerEnter(func() {
			if c.selected != value {
				c.selected = value
				c.rt.Invalidate()
			}
		}),
		twi.OnClick(func(*twi.Event) { c.choose(value) }),
	}, r.item.children...))
}

func (c *Command) key(k input.KeyEvent) bool {
	i := slices.Index(c.visible, c.selected)
	switch {
	case len(c.visible) == 0:
		return false
	case k.Key == input.KeyArrowDown:
		c.selected = c.visible[min(i+1, len(c.visible)-1)]
	case k.Key == input.KeyArrowUp:
		c.selected = c.visible[max(i-1, 0)]
	case k.Key == input.KeyEnter && k.Modifiers == 0:
		c.choose(c.selected)
	default:
		return false
	}
	return true
}

func (c *Command) choose(value string) {
	notify(c.OnSelect, value)
	if c.chosen != nil {
		c.chosen()
	}
	c.rt.Invalidate()
}

func score(value, search string) int {
	if search == "" {
		return 1
	}
	at := 0
	for i := range value {
		if foldPrefix(value[i:], search) {
			return 2*konst.MatchBase - min(at, konst.MatchBase-1)
		}
		at++
	}
	first, last, next := -1, 0, search
	at = 0
	for _, r := range value {
		want, size := utf8.DecodeRuneInString(next)
		if size == 0 {
			break
		}
		if unicode.ToLower(r) == unicode.ToLower(want) {
			if first < 0 {
				first = at
			}
			last, next = at, next[size:]
		}
		at++
	}
	if next != "" {
		return 0
	}
	return max(konst.MatchBase-(last-first), 1)
}

func foldPrefix(s, prefix string) bool {
	for _, p := range prefix {
		r, size := utf8.DecodeRuneInString(s)
		if size == 0 || unicode.ToLower(r) != unicode.ToLower(p) {
			return false
		}
		s = s[size:]
	}
	return true
}

type CommandDialog struct {
	*Dialog
	*Command
	Hotkey  rune
	wasOpen bool
}

func NewCommandDialog(rt *twi.Runtime) *CommandDialog {
	d := &CommandDialog{Dialog: newDialog(rt, palette, Bottom), Command: NewCommand(rt), Hotkey: 'k'}
	d.chosen = func() { d.set(false) }
	return d
}

func (d *CommandDialog) Node(children ...twi.NodeOption) twi.Node {
	if d.wasOpen && !d.Open {
		d.Search("")
	}
	d.wasOpen = d.Open
	hotkey := twi.OnKey(func(k input.KeyEvent) {
		if !k.Release && k.Key == input.KeyRune && k.Rune == d.Hotkey && k.Modifiers == input.ModCtrl {
			d.set(!d.Open)
		}
	})
	return part("absolute", []twi.NodeOption{hotkey, d.Content(d.Command.Node(children...))})
}
