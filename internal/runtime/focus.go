package runtime

import (
	"image"
	"slices"

	"github.com/pehcastro/twind/internal/events"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/style"
)

type Scope uint8

const (
	NoScope Scope = iota
	ModalScope
	NonModalScope
)

type Node struct {
	Key                              string
	At                               []int
	Focusable, Disabled, AutoFocus   bool
	TopLayer                         bool
	Scope                            Scope
	Measure                          *Ref
	Listeners                        []events.Listener[*Elem]
	PointerDownOutside, FocusOutside []func()
	Scroll                           []func(image.Point)
	Paste                            func(string)
	Hotkey                           func(input.KeyEvent) bool
	Width                            func(int)
	Children                         []Node
}

type Elem struct {
	node        Node
	parent      *Elem
	children    []*Elem
	frame       uint64
	offset      image.Point
	offsetKnown bool
}

type document struct {
	root    *Elem
	scene   *scene.Node
	frame   uint64
	scopes  []*Elem
	opened  []*Elem
	loose   []*Elem
	entered []entered
	autos   []*Elem
	layers  []*Elem
	refs    []*Elem
	scrolls []*Elem
	hotkeys []*Elem
	widths  []*Elem
	focused *Elem
	expect  events.Type
	exact   bool
	heard   bool
}

type entered struct{ scope, previous *Elem }

func (d *document) Root() *Elem                  { return d.root }
func (d *document) Parent(e *Elem) (*Elem, bool) { return e.parent, e.parent != nil }
func (d *document) Children(e *Elem) []*Elem     { return e.children }
func (d *document) Focusable(e *Elem) bool       { return e.frame == d.frame && e.node.Focusable }
func (d *document) TabIndex(*Elem) int           { return 0 }
func (d *document) Disabled(e *Elem) bool        { return e.node.Disabled }
func (d *document) gone(e *Elem) bool            { return e.frame != d.frame }

func (d *document) Listeners(e *Elem) []events.Listener[*Elem] {
	d.heard = d.heard || slices.ContainsFunc(e.node.Listeners, func(l events.Listener[*Elem]) bool { return !d.exact || l.Type == d.expect })
	return e.node.Listeners
}

func (d *document) Origin(e *Elem) image.Point {
	at := d.sceneOf(e).Bounds
	return image.Pt(at.X, at.Y)
}

func (e *Elem) clickable() bool {
	return slices.ContainsFunc(e.node.Listeners, func(l events.Listener[*Elem]) bool { return l.Type == events.Click })
}

func (d *document) update(root Node, focus *events.FocusManager[*Elem]) {
	d.frame++
	d.scopes, d.loose, d.autos, d.layers, d.refs, d.scrolls = d.scopes[:0], d.loose[:0], d.autos[:0], d.layers[:0], d.refs[:0], d.scrolls[:0]
	d.hotkeys, d.widths = d.hotkeys[:0], d.widths[:0]
	if d.root == nil {
		d.root = &Elem{}
	}
	d.attach(d.root, root)
	for len(d.opened) > 0 && slices.ContainsFunc(d.opened, d.gone) {
		focus.Close(d)
		d.opened = d.opened[:len(d.opened)-1]
	}
	for i := len(d.entered) - 1; i >= 0; i-- {
		e := d.entered[i]
		if !d.gone(e.scope) {
			continue
		}
		d.entered = slices.Delete(d.entered, i, i+1)
		if current, ok := focus.Current(); e.previous != nil && (!ok || e.scope.holds(current)) {
			focus.Set(d, e.previous)
		}
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
	for _, s := range d.loose {
		if slices.ContainsFunc(d.entered, func(e entered) bool { return e.scope == s }) {
			continue
		}
		e := entered{scope: s}
		if current, ok := focus.Current(); ok {
			e.previous = current
		}
		d.entered = append(d.entered, e)
		if first := d.first(s); first != nil {
			focus.Set(d, first)
		}
	}
	if _, ok := focus.Current(); !ok && len(d.autos) > 0 {
		focus.Set(d, d.autos[0])
	}
	d.moved(focus)
}

func (r *Runtime) Focus(key string) bool {
	looper, stopped := r.looping()
	switch looper {
	case "":
		return false
	case goroutine():
		return r.focusKey(key)
	}
	took := make(chan bool, 1)
	r.Dispatch(func() { took <- r.focusKey(key) })
	select {
	case ok := <-took:
		return ok
	case <-stopped:
		return false
	}
}

func (r *Runtime) focusKey(key string) bool {
	e := r.doc.root.keyed(key)
	if e == nil || !r.focus.Set(&r.doc, e) {
		return false
	}
	r.pointed = true
	r.refocused()
	return true
}

func (e *Elem) keyed(key string) *Elem {
	if e == nil || e.node.Key == key {
		return e
	}
	for _, c := range e.children {
		if found := c.keyed(key); found != nil {
			return found
		}
	}
	return nil
}

func (d *document) first(e *Elem) *Elem {
	if d.Focusable(e) && !d.Disabled(e) {
		return e
	}
	for _, c := range e.children {
		if f := d.first(c); f != nil {
			return f
		}
	}
	return nil
}

func (d *document) moved(focus *events.FocusManager[*Elem]) {
	current, ok := focus.Current()
	if !ok || current == d.focused {
		return
	}
	d.focused = current
	for _, e := range d.entered {
		if !e.scope.holds(current) {
			for _, f := range e.scope.node.FocusOutside {
				f()
			}
		}
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
	switch n.Scope {
	case NoScope:
	case ModalScope:
		d.scopes = append(d.scopes, e)
	case NonModalScope:
		d.loose = append(d.loose, e)
	default:
		panic("runtime: unknown focus scope")
	}
	if fresh && n.AutoFocus {
		d.autos = append(d.autos, e)
	}
	if n.TopLayer {
		d.layers = append(d.layers, e)
	}
	if n.Measure != nil {
		d.refs = append(d.refs, e)
	}
	if len(n.Scroll) > 0 {
		d.scrolls = append(d.scrolls, e)
	}
	if n.Hotkey != nil {
		d.hotkeys = append(d.hotkeys, e)
	}
	if n.Width != nil {
		d.widths = append(d.widths, e)
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
