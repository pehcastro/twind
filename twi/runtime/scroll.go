package runtime

import (
	"image"
	"slices"

	konst "github.com/pehcastro/twind/internal/konst/terminal"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/layout"
	"github.com/pehcastro/twind/twi/scene"
)

type scroller struct {
	path []int
	node *scene.Node
}

func (r *Runtime) wheel(ev input.MouseEvent) {
	var dx, dy int
	switch ev.Button {
	case input.MouseWheelUp:
		dy = -konst.WheelLines
	case input.MouseWheelDown:
		dy = konst.WheelLines
	case input.MouseWheelLeft:
		dx = -konst.WheelLines
	case input.MouseWheelRight:
		dx = konst.WheelLines
	case input.MouseNone, input.MouseLeft, input.MouseMiddle, input.MouseRight:
		return
	default:
		panic("runtime: unknown mouse button")
	}
	if r.pointer.hovered != nil {
		r.scrollFirst(r.scrollers(r.pointer.hovered), func(*scene.Node) (int, int) { return dx, dy })
	}
}

func (r *Runtime) scrollers(path []int) (innermostFirst []scroller) {
	n := &r.scene
	for i := 0; ; i++ {
		if n.Scroll {
			innermostFirst = append(innermostFirst, scroller{path[:i], n})
		}
		if i == len(path) || path[i] >= len(n.Children) {
			break
		}
		n = &n.Children[path[i]]
	}
	slices.Reverse(innermostFirst)
	return innermostFirst
}

func (r *Runtime) ScrollIntoView(key string) {
	r.intoView, r.dirty = key, true
}

func (r *Runtime) scrollIntoView() bool {
	e := r.doc.root.keyed(r.intoView)
	r.intoView = ""
	return e != nil && r.tree.ScrollToAnchor(e.path())
}

type Ref struct {
	bounds image.Rectangle
	frame  uint64
}

func (f *Ref) Bounds() image.Rectangle { return f.bounds }

func (r *Runtime) ContentBox(e *Elem) image.Rectangle {
	c := r.doc.sceneOf(e).Content
	return image.Rect(c.X, c.Y, c.X+c.W, c.Y+c.H)
}

func (r *Runtime) Viewport() image.Rectangle { return image.Rect(0, 0, r.width, r.height) }

func (r *Runtime) measure() bool {
	moved := false
	for _, e := range r.doc.widths {
		e.node.Width(r.doc.sceneOf(e).Content.W)
	}
	for _, e := range r.doc.refs {
		at, ref := r.doc.sceneOf(e).Bounds, e.node.Measure
		box := image.Rect(at.X, at.Y, at.X+at.W, at.Y+at.H)
		moved = moved || box != ref.bounds
		ref.bounds, ref.frame = box, r.doc.frame
	}
	for _, ref := range r.measured {
		if ref.frame != r.doc.frame && ref.bounds != (image.Rectangle{}) {
			ref.bounds, moved = image.Rectangle{}, true
		}
	}
	r.measured = r.measured[:0]
	for _, e := range r.doc.refs {
		r.measured = append(r.measured, e.node.Measure)
	}
	return moved
}

func (d *document) sceneOf(e *Elem) *scene.Node {
	n := d.scene
	for _, i := range e.path() {
		n = &n.Children[i]
	}
	return n
}

func (r *Runtime) scrolled() {
	for _, e := range r.doc.scrolls {
		n, at := r.doc.sceneOf(e), image.Point{}
		if n.Scroll {
			at = image.Pt(n.Padding.X-n.ScrollContent.X, n.Padding.Y-n.ScrollContent.Y)
		}
		moved := e.offsetKnown && at != e.offset
		e.offset, e.offsetKnown = at, true
		if !moved {
			continue
		}
		for _, handle := range e.node.Scroll {
			r.rebuild = true
			handle(at)
		}
	}
}

func contains(r layout.Rect, x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

func (r *Runtime) scrollKey(ev input.KeyEvent) {
	if ev.Modifiers != 0 || ev.Release {
		return
	}
	var path []int
	current, focused := r.focus.Current()
	if focused {
		path = current.path()
	}
	around := r.scrollers(path)
	var delta func(*scene.Node) (int, int)
	switch ev.Key {
	case input.KeyPageUp:
		delta = func(s *scene.Node) (int, int) { return 0, -s.Padding.H }
	case input.KeyPageDown:
		delta = func(s *scene.Node) (int, int) { return 0, s.Padding.H }
	case input.KeyHome:
		delta = func(s *scene.Node) (int, int) { return 0, -s.ScrollContent.H }
	case input.KeyEnd:
		delta = func(s *scene.Node) (int, int) { return 0, s.ScrollContent.H }
	case input.KeyArrowUp, input.KeyArrowDown, input.KeyArrowLeft, input.KeyArrowRight:
		if !focused || len(around) == 0 || len(around[0].path) != len(path) {
			return
		}
		dx, dy := 0, konst.ArrowLines
		switch ev.Key {
		case input.KeyArrowUp:
			dy = -konst.ArrowLines
		case input.KeyArrowLeft:
			dx, dy = -konst.ArrowLines, 0
		case input.KeyArrowRight:
			dx, dy = konst.ArrowLines, 0
		}
		r.scrollFirst(around, func(*scene.Node) (int, int) { return dx, dy })
		return
	default:
		return
	}
	if !r.scrollFirst(around, delta) {
		r.scrollFirst(r.page(), delta)
	}
}

func (r *Runtime) page() []scroller {
	var page []scroller
	area := 0
	var walk func(n *scene.Node, path []int)
	walk = func(n *scene.Node, path []int) {
		if a := n.Padding.W * n.Padding.H; n.Scroll && a > area {
			page, area = []scroller{{slices.Clone(path), n}}, a
		}
		for i := range n.Children {
			walk(&n.Children[i], append(path, i))
		}
	}
	walk(&r.scene, nil)
	return page
}

func (r *Runtime) scrollFirst(innermostFirst []scroller, delta func(*scene.Node) (int, int)) bool {
	for _, s := range innermostFirst {
		dx, dy := delta(s.node)
		if r.tree.ScrollBy(s.path, dx, dy) {
			r.dirty = true
			return true
		}
	}
	return false
}

func (e *Elem) path() []int {
	if e.parent == nil {
		return nil
	}
	return append(e.parent.path(), e.node.At...)
}
