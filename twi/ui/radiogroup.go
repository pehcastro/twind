package ui

import (
	"slices"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

type RadioGroup struct {
	control
	Value        string
	OnChange     func(string)
	items, built []string
}

func NewRadioGroup(rt *twi.Runtime) *RadioGroup { return &RadioGroup{control: control{rt: rt}} }

func (g *RadioGroup) Item(value string, options ...twi.NodeOption) twi.Node {
	active := value == g.Value || g.Value == "" && len(g.built) == 0
	g.built = append(g.built, value)
	var dot []twi.NodeOption
	if value == g.Value {
		dot = []twi.NodeOption{twi.Text("●")}
	}
	circle := part("flex flex-row w-2 h-1 shrink-0 items-center justify-center rounded-full text-primary dark:bg-input/30 "+g.ring(inputRing, onItem), append(dot, g.dataActive(active)))
	return part("flex flex-row items-center gap-3", append([]twi.NodeOption{circle, g.click(func() { g.choose(value) })}, options...))
}

func (g *RadioGroup) Node(options ...twi.NodeOption) twi.Node {
	g.items, g.built = g.built, nil
	return part("flex flex-col gap-1", append(g.behave(g.key), options...))
}

func (g *RadioGroup) key(k input.KeyEvent) bool {
	n := len(g.items)
	i := max(slices.Index(g.items, g.Value), 0)
	switch d := arrow(k); {
	case n == 0:
		return false
	case d != 0:
		i = (i + d + n) % n
	case !space(k):
		return false
	}
	g.choose(g.items[i])
	return true
}

func (g *RadioGroup) choose(value string) {
	if g.Value != value {
		g.Value = value
		notify(g.OnChange, value)
	}
}
