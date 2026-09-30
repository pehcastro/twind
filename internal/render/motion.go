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
	pose := motion.Pose{Scale: 1}
	if moving {
		if s.shown == nil {
			s.shown = new(style.ComputedStyle)
		}
		*s.shown = *next
		t.motion.Overlay(&t.overlay, s.shown)
		pose = motion.Pose{Scale: t.overlay.Pose.Scale, TranslateX: t.overlay.Pose.TranslateX, TranslateY: t.overlay.Pose.TranslateY}
	}
	s.painted = s.painted && !moving && !s.animated
	s.animated, s.pose = moving, pose
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
	for _, list := range [2][]*styledBox{s.children, s.exiting} {
		for _, c := range list {
			t.release(c)
		}
	}
	s.animated, s.painted, s.exiting = false, false, nil
	s.box.Children = s.box.Children[:len(s.children)]
}

func (t *Tree) exits(s *styledBox) {
	kept := s.exiting[:0]
	for _, e := range s.exiting {
		s.painted = false
		held := t.closing(e)
		if age := t.now - e.born; e.exit != nil && age < e.exit.Total() && !t.motion.Reduced {
			e.lift, e.painted, t.presenting, held = e.exit.Exit(age), false, true, true
		}
		if held {
			kept = append(kept, e)
			continue
		}
		t.release(e)
		t.relayout = true
	}
	clear(s.exiting[len(kept):])
	s.exiting = kept
}

func (t *Tree) closing(s *styledBox) bool {
	held := false
	if s.animated || t.motion.Holds(s.key) {
		switch a := &s.computed.Animation; a.Fill {
		case style.FillNone:
			a.Fill = style.FillForwards
		case style.FillBackwards:
			a.Fill = style.FillBoth
		case style.FillForwards, style.FillBoth:
		}
		t.animate(s, &s.computed, &s.computed)
		held = t.motion.Closing(s.key)
	}
	for _, c := range s.children {
		held = t.closing(c) || held
		s.painted = s.painted && c.painted
	}
	return held
}

func translated(l style.Length, size int) float64 {
	switch l.Unit {
	case style.Cells:
		return l.Value
	case style.Percent:
		return l.Value * float64(size) / 100
	case style.Auto, style.None, style.FitContent:
		return 0
	}
	panic(fmt.Sprintf("render: unknown translate unit %d", l.Unit))
}

func transform(n *scene.Node, dx, dy int, sx, sy float64) {
	b := n.Bounds
	cx, cy := float64(b.X)+float64(b.W)/2, float64(b.Y)+float64(b.H)/2
	at := func(x, y int) (int, int) {
		return int(math.Round(cx+(float64(x)-cx)*sx)) + dx, int(math.Round(cy+(float64(y)-cy)*sy)) + dy
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
