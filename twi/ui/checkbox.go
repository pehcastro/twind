package ui

import (
	"slices"

	"github.com/twind-dev/twind/twi"
)

type Checkbox struct {
	control
	Checked  bool
	OnChange func(bool)
}

func NewCheckbox(rt *twi.Runtime) *Checkbox { return &Checkbox{control: control{rt: rt}} }

func (c *Checkbox) Node(options ...twi.NodeOption) twi.Node {
	box, mark := "dark:bg-input/30 "+c.ring(inputRing, true), []twi.NodeOption(nil)
	if c.Checked {
		box, mark = "bg-primary text-primary-foreground "+c.ring(primaryRing, true), []twi.NodeOption{twi.Text("✓")}
	}
	keys := c.behave(flip(&c.Checked, c.OnChange, space))
	return part("flex flex-row w-2 h-1 shrink-0 items-center justify-center rounded-sm "+box, slices.Concat(keys, mark, options))
}
