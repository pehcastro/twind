package ui

import (
	"slices"

	"github.com/twind-dev/twind/twi"
)

type Switch struct {
	control
	Checked  bool
	OnChange func(bool)
}

func NewSwitch(rt *twi.Runtime) *Switch { return &Switch{control: control{rt: rt}} }

func (s *Switch) Node(options ...twi.NodeOption) twi.Node {
	track, thumb := "justify-start bg-input dark:bg-input/80", "bg-background dark:bg-foreground"
	if s.Checked {
		track, thumb = "justify-end bg-primary", "bg-background dark:bg-primary-foreground"
	}
	keys := s.behave(flip(&s.Checked, s.OnChange, press))
	return part("flex flex-row w-4 h-1 shrink-0 items-center rounded-full "+track+" "+s.ring("", true),
		slices.Concat(keys, []twi.NodeOption{part("w-2 h-1 rounded-full "+thumb, nil)}, options))
}
