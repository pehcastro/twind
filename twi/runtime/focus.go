package runtime

import (
	"slices"

	"github.com/twind-dev/twind/twi/events"
)

type Node struct {
	Key                  string
	Focusable, Disabled  bool
	Scope, AutoFocus     bool
	KeyDown, Focus, Blur []events.Listener[*Elem]
	Children             []Node
}

type Elem struct {
	node     Node
	parent   *Elem
	children []*Elem
	frame    uint64
}

type document struct {
	root   *Elem
	frame  uint64
	scopes []*Elem
	opened []*Elem
	autos  []*Elem
}

func (d *document) Root() *Elem                  { return d.root }
func (d *document) Parent(e *Elem) (*Elem, bool) { return e.parent, e.parent != nil }
func (d *document) Children(e *Elem) []*Elem     { return e.children }
func (d *document) Focusable(e *Elem) bool       { return e.frame == d.frame && e.node.Focusable }
func (d *document) TabIndex(*Elem) int           { return 0 }
func (d *document) Disabled(e *Elem) bool        { return e.node.Disabled }
func (d *document) gone(e *Elem) bool            { return e.frame != d.frame }
func (d *document) Listeners(e *Elem, t events.Type) []events.Listener[*Elem] {
	switch t {
	case events.KeyDown:
		return e.node.KeyDown
	case events.Focus:
		return e.node.Focus
	case events.Blur:
		return e.node.Blur
	case events.KeyUp, events.MouseDown, events.MouseUp, events.MouseMove, events.Click:
		return nil
	}
	panic("runtime: unknown event type")
}

func (d *document) update(root Node, focus *events.FocusManager[*Elem]) {
	d.frame++
	d.scopes, d.autos = d.scopes[:0], d.autos[:0]
	if d.root == nil {
		d.root = &Elem{}
	}
	d.attach(d.root, root)
	for len(d.opened) > 0 && slices.ContainsFunc(d.opened, d.gone) {
		focus.Close(d)
		d.opened = d.opened[:len(d.opened)-1]
	}
	if current, ok := focus.Current(); ok && d.gone(current) {
		*focus, d.opened = events.FocusManager[*Elem]{}, nil
	}
	for _, s := range d.scopes {
		if !slices.Contains(d.opened, s) {
			focus.Open(d, s)
			d.opened = append(d.opened, s)
		}
	}
	if _, ok := focus.Current(); !ok && len(d.autos) > 0 {
		focus.Set(d, d.autos[0])
	}
}

func (d *document) attach(e *Elem, n Node) {
	fresh := e.frame == 0
	e.node, e.frame = n, d.frame
	if n.Scope {
		d.scopes = append(d.scopes, e)
	}
	if fresh && n.AutoFocus {
		d.autos = append(d.autos, e)
	}
	kept := len(e.children) == len(n.Children)
	for i := 0; kept && i < len(n.Children); i++ {
		kept = e.children[i].node.Key == n.Children[i].Key
	}
	if !kept {
		old := e.children
		e.children = make([]*Elem, len(n.Children))
		for i, c := range n.Children {
			at := i
			if c.Key != "" {
				at = slices.IndexFunc(old, func(o *Elem) bool { return o != nil && o.node.Key == c.Key })
			}
			if at >= 0 && at < len(old) && old[at] != nil && old[at].node.Key == c.Key {
				e.children[i], old[at] = old[at], nil
			} else {
				e.children[i] = &Elem{parent: e}
			}
		}
	}
	for i, c := range n.Children {
		d.attach(e.children[i], c)
	}
}
