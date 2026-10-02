package ui

import (
	"slices"
	"strings"

	tkonst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/edit"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/text"
)

const (
	cursorClass      = "bg-foreground text-background"
	selectionClass   = "bg-primary text-primary-foreground"
	placeholderClass = "text-muted-foreground"
	fieldBox         = "w-full rounded-md dark:bg-input/30 "
)

type editor struct {
	edit.Buffer
	control
	Placeholder   string
	width, scroll int
	dragging      bool
	listeners     []twi.NodeOption
}

type Input struct{ editor }

type Textarea struct{ editor }

func NewInput(rt *twi.Runtime) *Input {
	return &Input{editor{control: control{rt: rt}}}
}

func NewTextarea(rt *twi.Runtime) *Textarea {
	return &Textarea{editor{Buffer: edit.Buffer{Mode: edit.MultiLine}, control: control{rt: rt}}}
}

func (in *Input) Node(options ...twi.NodeOption) twi.Node {
	return in.field("h-3 "+fieldBox+in.edge(fieldEdge), options)
}

func (t *Textarea) Node(options ...twi.NodeOption) twi.Node {
	return t.field("min-h-6 "+fieldBox+t.edge(fieldEdge), options)
}

func (in *Input) Group(addons ...Addon) twi.Node {
	return group(&in.control, in.field("h-1 grow", nil), addons)
}

func (t *Textarea) Group(addons ...Addon) twi.Node {
	return group(&t.control, t.field("min-h-4 grow", nil), addons)
}

func (e *editor) field(classes string, options []twi.NodeOption) twi.Node {
	e.Widths = e.rt.Widths()
	tag := style.ElementInput
	if e.Mode == edit.MultiLine {
		tag = style.ElementTextarea
	}
	if !e.focused {
		e.scroll = 0
	} else if e.width > 0 {
		_, caret := e.Cursor()
		e.scroll = max(min(e.scroll, caret), caret+1-e.width)
	}
	if e.scroll > 0 {
		widest := 0
		for line := range strings.SplitSeq(e.Value(), "\n") {
			widest = max(widest, e.Widths.Width(line))
		}
		e.scroll = max(min(e.scroll, widest+1-e.width), 0)
	}
	own := e.behave(nil)
	if !e.Disabled {
		if e.listeners == nil {
			e.listeners = []twi.NodeOption{twi.OnKeyDown(e.key), twi.OnPointerDown(e.press), twi.OnPointerMove(e.drag), twi.OnPointerUp(e.release)}
		}
		own = append(own, e.listeners...)
	}
	from := 0
	for line := range strings.SplitSeq(e.Value(), "\n") {
		own = append(own, e.row(line, from))
		from += len(line) + 1
	}
	return part(fade+"flex flex-col min-w-0 px-1 overflow-hidden select-none cursor-text "+classes, slices.Concat(own, options, []twi.NodeOption{twi.Tag(tag)}))
}

func (e *editor) row(line string, from int) twi.Node {
	skip, x := 0, 0
	for g := range text.Graphemes(line) {
		if x >= e.scroll {
			break
		}
		skip, x = skip+len(g), x+e.Widths.Width(g)
	}
	line, from = line[skip:], from+skip
	start, end := e.Selection()
	lo, hi := start-from, end-from
	parts := []twi.NodeOption{twi.Text(line)}
	switch {
	case !e.focused:
	case lo == hi && lo >= 0 && lo <= len(line):
		cursor := " "
		for g := range text.Graphemes(line[lo:]) {
			cursor = g
			break
		}
		parts = []twi.NodeOption{twi.Text(line[:lo]), part(cursorClass, []twi.NodeOption{twi.Text(cursor)}), twi.Text(line[min(lo+len(cursor), len(line)):])}
	case lo != hi && lo < len(line) && hi > 0:
		lo, hi = max(lo, 0), min(hi, len(line))
		parts = []twi.NodeOption{twi.Text(line[:lo]), part(selectionClass, []twi.NodeOption{twi.Text(line[lo:hi])}), twi.Text(line[hi:])}
	}
	if x > e.scroll {
		parts = slices.Insert(parts, 0, twi.NodeOption(twi.Text(" ")))
	}
	if e.Value() == "" {
		parts = append(parts, part(placeholderClass, []twi.NodeOption{twi.Text(e.Placeholder)}))
	}
	return part("flex flex-row h-1 min-w-0 overflow-hidden whitespace-pre", parts)
}

func (e *editor) key(ev *twi.Event) {
	k := ev.Key
	if k.Release {
		return
	}
	e.width = e.rt.ContentBox(ev.Current()).Dx()
	if e.Mode == edit.MultiLine && k.Key == input.KeyEnter && k.Modifiers == 0 {
		e.Insert("\n")
	} else if !e.Apply(k) {
		return
	}
	ev.PreventDefault()
	ev.StopPropagation()
	e.rt.Invalidate()
}

func (e *editor) at(ev *twi.Event) int {
	box := e.rt.ContentBox(ev.Current())
	e.width = box.Dx()
	return e.At(ev.Mouse.Y-box.Min.Y, ev.Mouse.X-box.Min.X+e.scroll)
}

func (e *editor) press(ev *twi.Event) {
	if ev.Mouse.Button != input.MouseLeft {
		return
	}
	unit := edit.Grapheme
	switch e.rt.Clicks() {
	case tkonst.WordClicks:
		unit = edit.Word
	case tkonst.LineClicks:
		unit = edit.Line
	}
	e.Press(e.at(ev), unit, unit == edit.Grapheme && ev.Mouse.Modifiers&input.ModShift != 0)
	e.dragging = true
	e.rt.Invalidate()
}

func (e *editor) drag(ev *twi.Event) {
	if e.dragging {
		e.Drag(e.at(ev))
		e.rt.Invalidate()
	}
}

func (e *editor) release(*twi.Event) { e.dragging = false }

type Align uint8

const (
	InlineStart Align = iota
	InlineEnd
	BlockStart
	BlockEnd
)

type Addon struct {
	align Align
	node  twi.Node
}

func InputGroupAddon(a Align, children ...twi.NodeOption) Addon {
	return Addon{a, part("flex flex-row items-center gap-1 font-medium text-muted-foreground select-none "+pick("input group addon", a, map[Align]string{
		InlineStart: "pl-1",
		InlineEnd:   "pr-1",
		BlockStart:  "w-full px-1",
		BlockEnd:    "w-full px-1",
	}), children)}
}

func InputGroupText(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center gap-1 text-muted-foreground", children)
}

func InputGroupButton(children ...twi.NodeOption) twi.Node {
	return Button(Ghost, SizeXS, children...)
}

func group(c *control, field twi.Node, addons []Addon, options ...twi.NodeOption) twi.Node {
	var at [BlockEnd + 1][]twi.NodeOption
	for _, a := range addons {
		at[a.align] = append(at[a.align], a.node)
	}
	middle := part("flex flex-row items-center w-full", slices.Concat(at[InlineStart], []twi.NodeOption{field}, at[InlineEnd]))
	return part("flex flex-col w-full min-w-0 rounded-md dark:bg-input/30 "+c.edge(groupEdge), slices.Concat(options, at[BlockStart], []twi.NodeOption{middle}, at[BlockEnd]))
}
