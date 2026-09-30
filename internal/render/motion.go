package render

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/motion"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
)

func (t *Tree) Wake() (time.Duration, bool) {
	if t.presenting {
		return t.now, true
	}
	return t.motion.Wake()
}

func moves(st *style.ComputedStyle) bool {
	return st.Transition.Duration+st.Transition.Delay > 0 || st.Animation.Keyframes != style.KeyframesNone
}

func (t *Tree) animate(s *styledBox, prev, next *style.ComputedStyle) {
	moving := t.motion.Frame(s.key, prev, next, t.now, &t.overlay)
	if moving {
		if s.shown == nil {
			s.shown = new(style.ComputedStyle)
		}
		*s.shown = *next
		t.motion.Overlay(&t.overlay, s.shown)
	}
	s.painted = s.painted && !moving && !s.animated
	s.animated = moving
}

func (t *Tree) key() motion.Key {
	if n := len(t.free); n > 0 {
		k := t.free[n-1]
		t.free = t.free[:n-1]
		return k
	}
	t.keys++
	return t.keys - 1
}

func (t *Tree) release(s *styledBox) {
	t.motion.Drop(s.key)
	t.free = append(t.free, s.key)
	s.animated, s.painted, s.exiting = false, false, nil
	s.box.Children = s.box.Children[:len(s.children)]
	for _, c := range s.children {
		t.release(c)
	}
}

func (t *Tree) leave(parent, gone *styledBox) {
	t.release(gone)
	t.relayout = true
	if gone.exit != nil && gone.exit.Duration > 0 && !t.motion.Reduced {
		gone.born = t.now
		parent.exiting = append(parent.exiting, gone)
	}
}

func (t *Tree) exits(s *styledBox) {
	kept := s.exiting[:0]
	for _, e := range s.exiting {
		s.painted = false
		if age := t.now - e.born; age < e.exit.Duration && !t.motion.Reduced {
			e.lift, e.painted, t.presenting = e.exit.Exit(age), false, true
			kept = append(kept, e)
			continue
		}
		t.relayout = true
	}
	clear(s.exiting[len(kept):])
	s.exiting = kept
}

func shift(l style.Length, size int, extra float64) int {
	switch l.Unit {
	case style.Cells:
		extra += l.Value
	case style.Percent:
		extra += l.Value * float64(size) / 100
	case style.Auto, style.None, style.FitContent:
	default:
		panic(fmt.Sprintf("render: unknown translate unit %d", l.Unit))
	}
	return int(math.Round(extra))
}

func transform(n *scene.Node, dx, dy int, scale float64) {
	b := n.Bounds
	cx, cy := float64(b.X)+float64(b.W)/2, float64(b.Y)+float64(b.H)/2
	at := func(x, y int) (int, int) {
		return int(math.Round(cx+(float64(x)-cx)*scale)) + dx, int(math.Round(cy+(float64(y)-cy)*scale)) + dy
	}
	whole := func(r layout.Rect) layout.Rect {
		x0, y0 := at(r.X, r.Y)
		x1, y1 := at(r.X+r.W, r.Y+r.H)
		return layout.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
	}
	n.ScrollContent.X, n.ScrollContent.Y = at(n.ScrollContent.X, n.ScrollContent.Y)
	n.Bounds, n.Padding, n.Content = whole(n.Bounds), whole(n.Padding), whole(n.Content)
	n.Children = carried(n.Children, at, n.Clip)
}

func carried(nodes []scene.Node, at func(x, y int) (int, int), clip layout.Rect) []scene.Node {
	nodes = slices.Clone(nodes)
	for i := range nodes {
		c := &nodes[i]
		x, y := at(c.Bounds.X, c.Bounds.Y)
		dx, dy := x-c.Bounds.X, y-c.Bounds.Y
		for _, r := range [...]*layout.Rect{&c.Bounds, &c.Padding, &c.Content, &c.ScrollContent, &c.Clip} {
			r.X, r.Y = r.X+dx, r.Y+dy
		}
		c.Clip = overlap(c.Clip, clip)
		c.Children = carried(c.Children, at, clip)
	}
	return nodes
}
