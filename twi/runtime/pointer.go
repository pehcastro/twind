package runtime

import (
	"fmt"
	"slices"

	"github.com/twind-dev/twind/twi/events"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
)

type pointer struct {
	at               input.MouseEvent
	seen, moved      bool
	hovered, pressed []int
	over, down       *Elem
}

func (r *Runtime) point(ev input.MouseEvent) {
	r.pointer.at, r.pointer.seen = ev, true
	if ev.Action == input.MouseMove {
		r.pointer.moved = true
		return
	}
	r.hover()
	switch ev.Action {
	case input.MouseScroll:
		r.wheel(ev)
	case input.MousePress:
		r.press(ev)
	case input.MouseRelease:
		r.release(ev)
	default:
		panic(fmt.Sprintf("runtime: unknown mouse action %d", ev.Action))
	}
}

func (r *Runtime) hover() {
	r.pointer.moved = false
	r.extend()
	path := r.hit(r.pointer.at.X, r.pointer.at.Y)
	r.dirty = r.restyles(r.pointer.hovered, path, style.StateHover) || r.dirty
	r.pointer.hovered = path
	prev, target := r.pointer.over, r.doc.at(path)
	if target == prev {
		return
	}
	r.pointer.over = target
	if prev != nil && !r.doc.gone(prev) {
		r.send(prev, events.PointerOut)
	}
	for e := prev; e != nil && !e.holds(target); e = e.parent {
		if !r.doc.gone(e) {
			r.send(e, events.PointerLeave)
		}
	}
	if target == nil {
		return
	}
	r.send(target, events.PointerOver)
	var entering []*Elem
	for e := target; e != nil && !e.holds(prev); e = e.parent {
		entering = append(entering, e)
	}
	for _, e := range slices.Backward(entering) {
		r.send(e, events.PointerEnter)
	}
}

func (r *Runtime) press(ev input.MouseEvent) {
	r.pointed = true
	target := r.pointer.over
	if ev.Button == input.MouseLeft {
		r.dirty = r.restyles(r.pointer.pressed, r.pointer.hovered, style.StateActive) || r.dirty
		r.pointer.pressed, r.pointer.down = r.pointer.hovered, target
	}
	prevented := target != nil && r.send(target, events.PointerDown).DefaultPrevented()
	if ev.Button == input.MouseLeft {
		r.pick(ev, prevented || target != nil && (target.node.Focusable || len(target.node.Click) > 0))
	}
	if target == nil {
		return
	}
	if !prevented {
		for e := target; e != nil; e = e.parent {
			if r.doc.Focusable(e) && !r.doc.Disabled(e) {
				r.focus.Set(&r.doc, e)
				break
			}
		}
	}
	var outside []func()
	var walk func(*Elem)
	walk = func(e *Elem) {
		if !e.holds(target) {
			outside = append(outside, e.node.PointerDownOutside...)
		}
		for _, c := range e.children {
			walk(c)
		}
	}
	walk(r.doc.root)
	for _, f := range outside {
		f()
	}
}

func (r *Runtime) release(ev input.MouseEvent) {
	target := r.pointer.over
	if target != nil {
		r.send(target, events.PointerUp)
	}
	if ev.Button != input.MouseLeft {
		return
	}
	r.sel.dragging, r.sel.active = false, r.sel.shown
	r.dirty = r.restyles(r.pointer.pressed, nil, style.StateActive) || r.dirty
	down := r.pointer.down
	r.pointer.pressed, r.pointer.down = nil, nil
	if target != nil && target == down && !r.doc.Disabled(target) {
		r.send(target, events.Click)
	}
}

func (r *Runtime) send(target *Elem, t events.Type) *events.Event[*Elem] {
	e := &events.Event[*Elem]{Type: t, Mouse: r.pointer.at}
	events.Dispatch(&r.doc, target, e)
	return e
}

func (r *Runtime) hit(x, y int) []int {
	var top *scene.Node
	scene.Walk(&r.scene, func(n *scene.Node) {
		if contains(n.Bounds, x, y) && contains(n.Clip, x, y) {
			top = n
		}
	}, func(_ *scene.Node, inside func()) { inside() })
	if top == nil {
		return nil
	}
	var find func(n *scene.Node, path []int) []int
	find = func(n *scene.Node, path []int) []int {
		if n == top {
			return slices.Clone(path)
		}
		for i := range n.Children {
			if found := find(&n.Children[i], append(path, i)); found != nil {
				return found
			}
		}
		return nil
	}
	return find(&r.scene, []int{})
}
