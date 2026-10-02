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
	if ev.Action == input.MouseMove {
		r.pointer.at, r.pointer.seen, r.pointer.moved = ev, true, true
		return
	}
	if r.pointer.moved {
		r.move()
	}
	r.pointer.at, r.pointer.seen = ev, true
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

func (r *Runtime) move() {
	r.hover()
	if target := r.captured(); target != nil {
		r.send(target, events.PointerMove)
	}
}

func (r *Runtime) captured() *Elem {
	if down := r.pointer.down; down != nil && !r.doc.gone(down) {
		return down
	}
	return r.pointer.over
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
	r.dirty = r.dirty || !r.ringless
	r.pointed, r.ringless = true, true
	target := r.pointer.over
	r.pointer.down = target
	if ev.Button == input.MouseLeft {
		r.dirty = r.restyles(r.pointer.pressed, r.pointer.hovered, style.StateActive) || r.dirty
		r.pointer.pressed = r.pointer.hovered
		r.sel.count(ev, r.cfg.Clock.Now())
	}
	prevented := target != nil && r.send(target, events.PointerDown).DefaultPrevented()
	if ev.Button == input.MouseLeft {
		r.pick(ev, prevented || target != nil && target.clickable())
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
	over, down := r.pointer.over, r.pointer.down
	if target := r.captured(); target != nil {
		r.send(target, events.PointerUp)
	}
	r.pointer.down = nil
	if ev.Button != input.MouseLeft {
		return
	}
	r.sel.dragging, r.sel.active = false, r.sel.shown
	r.dirty = r.restyles(r.pointer.pressed, nil, style.StateActive) || r.dirty
	r.pointer.pressed = nil
	if over != nil && over == down && !r.doc.Disabled(over) {
		r.send(over, events.Click)
	}
}

func (r *Runtime) send(target *Elem, t events.Type) *events.Event[*Elem] {
	e := &events.Event[*Elem]{Type: t, Mouse: r.pointer.at}
	r.doc.expect, r.doc.exact = t, true
	events.Dispatch(&r.doc, target, e)
	r.doc.exact = false
	return e
}

func (r *Runtime) hit(x, y int) []int {
	top := r.walker.Hit(&r.scene, x, y, func(n *scene.Node) bool {
		return contains(n.Bounds, x, y) && contains(n.Clip, x, y) && n.Visibility == style.Visible && n.PointerEvents != style.PointerNone
	})
	if top == nil {
		return nil
	}
	var find func(n *scene.Node, path []int) []int
	find = func(n *scene.Node, path []int) []int {
		if n == top {
			return slices.Clone(path)
		}
		for i := range n.Children {
			if !n.Children[i].Holds(x, y) {
				continue
			}
			if found := find(&n.Children[i], append(path, i)); found != nil {
				return found
			}
		}
		return nil
	}
	return find(&r.scene, []int{})
}
