package runtime

import (
	"image"
	"slices"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
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
		around, _ := r.scrollers(r.pointer.hovered)
		r.scrollFirst(around, func(*scene.Node) (int, int) { return dx, dy })
	}
}

func (r *Runtime) scrollers(path []int) (innermostFirst []scroller, target *scene.Node) {
	n := &r.scene
	for i := 0; ; i++ {
		if n.Scroll {
			innermostFirst = append(innermostFirst, scroller{path[:i], n})
		}
		if i == len(path) {
			target = n
			break
		}
		if path[i] >= len(n.Children) {
			break
		}
		n = &n.Children[path[i]]
	}
	slices.Reverse(innermostFirst)
	return innermostFirst, target
}

func (r *Runtime) ScrollIntoView(key string) {
	r.intoView, r.dirty = key, true
}

func (r *Runtime) scrollIntoView() bool {
	e := r.doc.root.keyed(r.intoView)
	r.intoView = ""
	if e == nil {
		return false
	}
	path := e.path()
	around, target := r.scrollers(path)
	if target == nil {
		return false
	}
	at, moved, aligned := target.Bounds, false, false
	for _, s := range around {
		if len(s.path) == len(path) {
			continue
		}
		view, content := s.node.Padding, s.node.ScrollContent
		x, y := view.X-content.X, view.Y-content.Y
		nx, ny := x+render.Reveal(at.X, at.W, view.X, view.W), y+render.Reveal(at.Y, at.H, view.Y, view.H)
		if !aligned {
			ny, aligned = y+at.Y-view.Y, true
		}
		nx, ny = max(min(nx, content.W-view.W), 0), max(min(ny, content.H-view.H), 0)
		moved = r.tree.ScrollTo(s.path, nx, ny) || moved
		at.X, at.Y = at.X-(nx-x), at.Y-(ny-y)
	}
	return moved
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
	current, focused := r.focus.Current()
	if !focused || ev.Modifiers != 0 || ev.Release {
		return
	}
	path := current.path()
	around, _ := r.scrollers(path)
	arrows := len(around) > 0 && len(around[0].path) == len(path)
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
		if !arrows {
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
		delta = func(*scene.Node) (int, int) { return dx, dy }
	default:
		return
	}
	r.scrollFirst(around, delta)
}

func (r *Runtime) scrollFirst(innermostFirst []scroller, delta func(*scene.Node) (int, int)) {
	for _, s := range innermostFirst {
		dx, dy := delta(s.node)
		if r.tree.ScrollBy(s.path, dx, dy) {
			r.dirty = true
			return
		}
	}
}

func (e *Elem) path() []int {
	if e.parent == nil {
		return nil
	}
	return append(e.parent.path(), e.node.At...)
}
