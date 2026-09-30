package ui

import (
	"slices"
	"strings"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/edit"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/text"
)

const (
	cursorClass      = "bg-foreground text-background"
	selectionClass   = "bg-primary text-primary-foreground"
	placeholderClass = "text-muted-foreground"
)

type Input struct {
	*twi.Input
	control
}

func NewInput(rt *twi.Runtime) *Input {
	in := &Input{Input: twi.NewInput(rt), control: control{rt: rt}}
	in.CursorClass, in.SelectionClass, in.PlaceholderClass = cursorClass, selectionClass, placeholderClass
	return in
}

func (in *Input) Node(options ...twi.NodeOption) twi.Node {
	return in.field("w-full rounded-md dark:bg-input/30 "+in.ring(inputRing, onSelf), options)
}

func (in *Input) field(classes string, options []twi.NodeOption) twi.Node {
	return in.Input.Node(append(in.behave(nil), merged(fade+"flex flex-row h-1 min-w-0 px-1 overflow-hidden "+classes, options)...)...)
}

type Textarea struct {
	edit.Buffer
	control
	Placeholder string
}

func NewTextarea(rt *twi.Runtime) *Textarea {
	return &Textarea{Buffer: edit.Buffer{Mode: edit.MultiLine}, control: control{rt: rt}}
}

func (t *Textarea) Node(options ...twi.NodeOption) twi.Node {
	return t.field("w-full rounded-md dark:bg-input/30 "+t.ring(inputRing, onSelf), options)
}

func (t *Textarea) field(classes string, options []twi.NodeOption) twi.Node {
	keys := t.behave(func(k input.KeyEvent) bool {
		if k.Key == input.KeyEnter && k.Modifiers == 0 {
			t.Insert("\n")
			return true
		}
		return t.Apply(k)
	})
	var rows []twi.NodeOption
	from := 0
	for line := range strings.SplitSeq(t.Value(), "\n") {
		rows = append(rows, t.row(line, from))
		from += len(line) + 1
	}
	return part(fade+"flex flex-col min-h-4 px-1 overflow-hidden "+classes, slices.Concat(keys, rows, options))
}

func (t *Textarea) row(line string, from int) twi.Node {
	plain := func(s string) twi.Node { return twi.Text(strings.ReplaceAll(s, " ", " ")) }
	start, end := t.Selection()
	start, end = start-from, end-from
	parts := []twi.NodeOption{plain(line)}
	switch {
	case !t.focused:
	case start == end && start >= 0 && start <= len(line):
		cursor := " "
		for g := range text.Graphemes(line[start:]) {
			cursor = g
			break
		}
		parts = []twi.NodeOption{plain(line[:start]), part(cursorClass, []twi.NodeOption{plain(cursor)}), plain(line[min(start+len(cursor), len(line)):])}
	case start != end && start < len(line) && end > 0:
		lo, hi := max(start, 0), min(end, len(line))
		parts = []twi.NodeOption{plain(line[:lo]), part(selectionClass, []twi.NodeOption{plain(line[lo:hi])}), plain(line[hi:])}
	}
	if t.Value() == "" {
		parts = append(parts, part(placeholderClass, []twi.NodeOption{twi.Text(t.Placeholder)}))
	}
	return part("flex flex-row h-1", parts)
}

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

func (in *Input) Group(addons ...Addon) twi.Node {
	return group(&in.control, in.field("grow", nil), addons)
}

func (t *Textarea) Group(addons ...Addon) twi.Node {
	return group(&t.control, t.field("grow", nil), addons)
}

func group(c *control, field twi.Node, addons []Addon, options ...twi.NodeOption) twi.Node {
	var at [BlockEnd + 1][]twi.NodeOption
	for _, a := range addons {
		at[a.align] = append(at[a.align], a.node)
	}
	middle := part("flex flex-row items-center w-full", slices.Concat(at[InlineStart], []twi.NodeOption{field}, at[InlineEnd]))
	return part("flex flex-col w-full min-w-0 rounded-md dark:bg-input/30 "+c.ring(inputRing, onGroup), slices.Concat(options, at[BlockStart], []twi.NodeOption{middle}, at[BlockEnd]))
}
