package runtime

import (
	"slices"

	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/events"
	"github.com/twind-dev/twind/twi/style"
)

type Node struct {
	Key                  string
	At                   []int
	Focusable, Disabled  bool
	Scope, AutoFocus     bool
	KeyDown, Focus, Blur []events.Listener[*Elem]
	Click, Enter, Leave  []events.Listener[*Elem]
	PointerDownOutside   []func()
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
	case events.Click:
		return e.node.Click
	case events.PointerEnter:
		return e.node.Enter
	case events.PointerLeave:
		return e.node.Leave
	case events.KeyUp, events.PointerDown, events.PointerUp, events.PointerOver, events.PointerOut:
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

func mark(n render.Node, path []int, at, along style.State) render.Node {
	state := style.NodeState{}
	if n.State != nil {
		state = *n.State
	}
	if len(path) == 0 {
		state.States |= at
	} else {
		state.States |= along
		if path[0] < len(n.Children) {
			n.Children = slices.Clone(n.Children)
			n.Children[path[0]] = mark(n.Children[path[0]], path[1:], at, along)
		}
	}
	n.State = &state
	return n
}

func (d *document) at(path []int) *Elem {
	e := d.root
	if path == nil || e == nil {
		return nil
	}
	for {
		i := slices.IndexFunc(e.children, func(c *Elem) bool {
			return len(c.node.At) <= len(path) && slices.Equal(c.node.At, path[:len(c.node.At)])
		})
		if i < 0 {
			return e
		}
		path, e = path[len(e.children[i].node.At):], e.children[i]
	}
}

func (e *Elem) holds(n *Elem) bool {
	for ; n != nil; n = n.parent {
		if n == e {
			return true
		}
	}
	return false
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
