package twi

import (
	"slices"
	"strings"

	"github.com/twind-dev/twind/twi/edit"
	"github.com/twind-dev/twind/twi/runtime"
	"github.com/twind-dev/twind/twi/text"
)

type Input struct {
	edit.Buffer
	Placeholder                                   string
	CursorClass, SelectionClass, PlaceholderClass string
	owner                                         *runtime.Runtime
	focused                                       bool
}

func NewInput(rt *Runtime) *Input { return &Input{owner: rt.Runtime} }

func (in *Input) Node(options ...NodeOption) Node {
	in.Widths = in.owner.Widths()
	value := in.Value()
	start, end := in.Selection()
	unbroken := func(s string) Node { return Text(strings.ReplaceAll(s, " ", " ")) }
	parts := []NodeOption{unbroken(value)}
	switch {
	case !in.focused:
	case start == end:
		cursor := " "
		for g := range text.Graphemes(value[start:]) {
			cursor = g
			break
		}
		end = min(start+len(cursor), len(value))
		parts = []NodeOption{unbroken(value[:start]), Element(Class(in.CursorClass), unbroken(cursor)), unbroken(value[end:])}
	default:
		parts = []NodeOption{unbroken(value[:start]), Element(Class(in.SelectionClass), unbroken(value[start:end])), unbroken(value[end:])}
	}
	if value == "" {
		parts = append(parts, Element(Class(in.PlaceholderClass), Text(in.Placeholder)))
	}
	return Element(slices.Concat([]NodeOption{
		Focusable(),
		OnKeyDown(func(e *Event) {
			if in.Apply(e.Key) {
				e.PreventDefault()
				e.StopPropagation()
				in.owner.Invalidate()
			}
		}),
		OnFocus(func() { in.focus(true) }),
		OnBlur(func() { in.focus(false) }),
	}, options, parts)...)
}

func (in *Input) focus(focused bool) {
	in.focused = focused
	in.owner.Invalidate()
}
