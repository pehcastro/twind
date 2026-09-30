package runtime

import (
	"slices"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
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
	at, moved := target.Bounds, false
	for _, s := range around {
		if len(s.path) == len(path) {
			continue
		}
		view, content := s.node.Padding, s.node.ScrollContent
		x, y := view.X-content.X, view.Y-content.Y
		nx, ny := x, y+at.Y-view.Y
		switch {
		case at.X < view.X:
			nx += at.X - view.X
		case at.X+at.W > view.X+view.W:
			nx += min(at.X+at.W-view.X-view.W, at.X-view.X)
		}
		nx, ny = max(min(nx, content.W-view.W), 0), max(min(ny, content.H-view.H), 0)
		moved = r.tree.ScrollTo(s.path, nx, ny) || moved
		at.X, at.Y = at.X-(nx-x), at.Y-(ny-y)
	}
	return moved
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
