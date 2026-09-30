package ui

import (
	"slices"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

func toggle(c *control, v Variant, s Size, on bool, at ringAt) string {
	classes := fade + "flex flex-row shrink-0 items-center justify-center gap-1 h-1 rounded-md font-medium select-none [&_svg]:shrink-0 [&_svg]:pointer-events-none " +
		pick("toggle", s, map[Size]string{SizeDefault: "min-w-4 px-1", SizeSM: "min-w-3 px-1", SizeLG: "min-w-5 px-1"})
	if on {
		classes += " bg-accent text-accent-foreground"
	}
	return classes + " " + c.ring(pick("toggle", v, map[Variant]string{Default: "", Outline: inputRing}), at)
}

type Toggle struct {
	control
	Variant  Variant
	Size     Size
	Pressed  bool
	OnChange func(bool)
}

func NewToggle(rt *twi.Runtime) *Toggle { return &Toggle{control: control{rt: rt}} }

func (t *Toggle) Node(options ...twi.NodeOption) twi.Node {
	return part(toggle(&t.control, t.Variant, t.Size, t.Pressed, onSelf), slices.Concat(t.pressable(press, flip(&t.Pressed, t.OnChange)), options))
}

type ToggleGroup struct {
	control
	Variant      Variant
	Size         Size
	Multiple     bool
	Value        []string
	OnChange     func([]string)
	active       int
	items, built []string
}

func NewToggleGroup(rt *twi.Runtime) *ToggleGroup { return &ToggleGroup{control: control{rt: rt}} }

func (g *ToggleGroup) Item(value string, options ...twi.NodeOption) twi.Node {
	at := len(g.built)
	g.built = append(g.built, value)
	pick := g.click(func() {
		g.active = at
		g.flip(value)
	})
	return part(toggle(&g.control, g.Variant, g.Size, slices.Contains(g.Value, value), onItem), append([]twi.NodeOption{g.dataActive(at == g.active), pick}, options...))
}

func (g *ToggleGroup) Node(options ...twi.NodeOption) twi.Node {
	g.items, g.built = g.built, nil
	g.active = min(g.active, max(len(g.items)-1, 0))
	return part(fade+"flex flex-row items-center gap-1 rounded-md", append(g.behave(g.key), options...))
}

func (g *ToggleGroup) key(k input.KeyEvent) bool {
	n := len(g.items)
	switch d := arrow(k); {
	case n == 0:
		return false
	case d != 0:
		g.active = (g.active + d + n) % n
	case k.Key == input.KeyHome:
		g.active = 0
	case k.Key == input.KeyEnd:
		g.active = n - 1
	case !press(k):
		return false
	default:
		g.flip(g.items[g.active])
	}
	return true
}

func (g *ToggleGroup) flip(v string) {
	switch {
	case slices.Contains(g.Value, v):
		g.Value = slices.DeleteFunc(slices.Clone(g.Value), func(s string) bool { return s == v })
	case g.Multiple:
		g.Value = append(slices.Clone(g.Value), v)
	default:
		g.Value = []string{v}
	}
	notify(g.OnChange, g.Value)
}
