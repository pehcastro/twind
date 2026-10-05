package ui

import (
	"slices"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
)

type NativeSelect struct {
	anchored
	Options  []string
	Value    string
	OnChange func(string)
	active   int
}

func NewNativeSelect(rt *twi.Runtime) *NativeSelect {
	s := &NativeSelect{anchored: newAnchored(rt, Bottom, Start)}
	s.anchorWidth = true
	return s
}

func (s *NativeSelect) Node(options ...twi.NodeOption) twi.Node {
	label := func(i int) string { return s.Options[i] }
	keys := s.behave(func(k input.KeyEvent) bool {
		last := len(s.Options) - 1
		i := slices.Index(s.Options, s.Value)
		switch d := arrow(k); {
		case last < 0:
			return false
		case press(k) || k.Key == input.KeyF4 || k.Modifiers == input.ModAlt && (k.Key == input.KeyArrowDown || k.Key == input.KeyArrowUp):
			s.open()
			return true
		case d != 0:
			i = min(max(i+d, 0), last)
		case k.Key == input.KeyHome:
			i = 0
		case k.Key == input.KeyEnd:
			i = last
		case typed(k):
			i = typeahead(last+1, i, k.Rune, label)
		default:
			return false
		}
		s.change(s.Options[i])
		return true
	})
	items := make([]twi.NodeOption, len(s.Options))
	for i, o := range s.Options {
		items[i] = s.option(o, i, &s.active, o == s.Value, func() { s.choose(o) }, nil)
	}
	trigger := part(fade+"flex flex-row h-1 items-center gap-1 px-1 rounded-md dark:bg-input/30 "+s.ring(inputRing, onSelf), slices.Concat(keys, []twi.NodeOption{
		s.toggle(s.open),
		part("grow", []twi.NodeOption{twi.Text(s.Value)}),
		icon("⌄", "text-muted-foreground opacity-50"),
	}, options))
	return s.anchored.Node(trigger, s.list(func(k input.KeyEvent) bool {
		return s.listKey(k, &s.active, len(s.Options), label, func(i int) { s.choose(s.Options[i]) })
	}, items))
}

func (s *NativeSelect) open() {
	s.active = max(slices.Index(s.Options, s.Value), 0)
	s.set(true)
}

func (s *NativeSelect) choose(value string) {
	s.change(value)
	s.set(false)
}

func (s *NativeSelect) change(value string) {
	if s.Value != value {
		s.Value = value
		notify(s.OnChange, value)
	}
}
