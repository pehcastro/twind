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
		r.scrollFirst(r.scrollers(r.pointer.hovered), func(*scene.Node) (int, int) { return dx, dy })
	}
}

func (r *Runtime) scrollers(path []int) []scroller {
	var around []scroller
	n := &r.scene
	for i := 0; ; i++ {
		if n.Scroll {
			around = append(around, scroller{path[:i], n})
		}
		if i == len(path) || path[i] >= len(n.Children) {
			break
		}
		n = &n.Children[path[i]]
	}
	slices.Reverse(around)
	return around
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
	around := r.scrollers(path)
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
