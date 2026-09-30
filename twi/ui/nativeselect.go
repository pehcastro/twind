package ui

import (
	"slices"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

type NativeSelect struct {
	control
	Options  []string
	Value    string
	OnChange func(string)
}

func NewNativeSelect(rt *twi.Runtime) *NativeSelect { return &NativeSelect{control: control{rt: rt}} }

func (s *NativeSelect) Node(options ...twi.NodeOption) twi.Node {
	keys := s.behave(func(k input.KeyEvent) bool {
		last := len(s.Options) - 1
		i := slices.Index(s.Options, s.Value)
		switch d := arrow(k); {
		case last < 0:
			return false
		case d != 0:
			i = min(max(i+d, 0), last)
		case k.Key == input.KeyHome:
			i = 0
		case k.Key == input.KeyEnd:
			i = last
		default:
			return false
		}
		if s.Options[i] != s.Value {
			s.Value = s.Options[i]
			notify(s.OnChange, s.Value)
		}
		return true
	})
	return part("flex flex-row h-1 items-center gap-1 px-1 rounded-md dark:bg-input/30 "+s.ring(inputRing, true), slices.Concat(keys, []twi.NodeOption{
		part("grow", []twi.NodeOption{twi.Text(s.Value)}),
		part("text-muted-foreground opacity-50", []twi.NodeOption{twi.Text("⌄")}),
	}, options))
}
