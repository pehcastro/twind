package ui

import (
	"slices"
	"strings"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

const pageSteps = 10

type Slider struct {
	control
	Value, Min, Max, Step int
	OnChange              func(int)
}

func NewSlider(rt *twi.Runtime) *Slider {
	return &Slider{control: control{rt: rt}, Max: 100, Step: 1}
}

func (s *Slider) Node(options ...twi.NodeOption) twi.Node {
	percent := 0
	if s.Max > s.Min {
		percent = (min(max(s.Value, s.Min), s.Max) - s.Min) * 100 / (s.Max - s.Min)
	}
	keys := s.behave(func(k input.KeyEvent) bool {
		v := s.Value
		switch k.Key {
		case input.KeyArrowRight, input.KeyArrowUp:
			v += s.Step
		case input.KeyArrowLeft, input.KeyArrowDown:
			v -= s.Step
		case input.KeyPageUp:
			v += pageSteps * s.Step
		case input.KeyPageDown:
			v -= pageSteps * s.Step
		case input.KeyHome:
			v = s.Min
		case input.KeyEnd:
			v = s.Max
		default:
			return false
		}
		if v = min(max(v, s.Min), s.Max); v != s.Value {
			s.Value = v
			notify(s.OnChange, v)
		}
		return true
	})
	return part("relative flex flex-row w-full h-1 items-center rounded-full bg-muted select-none", slices.Concat(keys, []twi.NodeOption{
		part("h-1 rounded-full bg-primary "+strings.Fields(indicatorWidths)[percent], nil),
		part("w-2 h-1 shrink-0 rounded-full bg-white "+s.ring(primaryRing, onItem), []twi.NodeOption{s.dataActive(true)}),
	}, options))
}
