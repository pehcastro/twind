package runtime

import (
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	pkonst "github.com/twind-dev/twind/internal/konst/paint"
	konst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/paint"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/text"
)

type spot struct{ node, from, to int }

func (a spot) before(b spot) bool { return a.node < b.node || a.node == b.node && a.from < b.from }

type glyph struct {
	paint.Glyph
	spot
}

type selectable struct {
	path       []int
	raw, clean string
	clip       layout.Rect
}

type selection struct {
	active, shown, dragging bool
	root                    []int
	bounds                  layout.Rect
	anchor, focus           spot
	press                   input.MouseEvent
	pressed                 time.Time
	clicks                  int
	texts                   []selectable
	glyphs                  []glyph
}

func (s *selection) clear() {
	s.active, s.shown, s.dragging = false, false, false
}

func (s *selection) span() (lo, hi spot) {
	if s.focus.before(s.anchor) {
		return s.focus, s.anchor
	}
	return s.anchor, s.focus
}

func (r *Runtime) pick(ev input.MouseEvent, refused bool) {
	s := &r.sel
	now := r.cfg.Clock.Now()
	near := max(ev.X-s.press.X, s.press.X-ev.X, ev.Y-s.press.Y, s.press.Y-ev.Y) <= konst.MultiClickSlack
	if near && now.Sub(s.pressed) <= konst.MultiClick {
		s.clicks = min(s.clicks+1, konst.LineClicks)
	} else {
		s.clicks = 1
	}
	s.press, s.pressed = ev, now
	r.dirty = r.dirty || s.shown
	s.clear()
	path := r.pointer.hovered
	if refused || path == nil {
		return
	}
	root, all := 0, false
	_, _, st, ok := r.descend(path, func(depth int, n *scene.Node, st style.ComputedStyle) {
		if all {
			return
		}
		b := n.Border
		if depth == 0 || b.Top || b.Right || b.Bottom || b.Left || n.HidesOverflow || n.TopLayer > 0 {
			root = depth
		}
		if st.UserSelect == style.SelectAll {
			root, all = depth, true
		}
	})
	if !ok || st.UserSelect == style.SelectNone {
		return
	}
	s.active, s.root = true, append(s.root[:0], path[:root]...)
	r.flow()
	if len(s.glyphs) == 0 {
		s.clear()
		return
	}
	at := s.near(ev.X, ev.Y)
	switch {
	case all:
		s.expand(at, func(a, b glyph) bool { return true })
	case s.clicks == konst.LineClicks:
		s.expand(at, func(a, b glyph) bool {
			return a.node == b.node && !strings.Contains(s.texts[a.node].clean[a.from:b.from], "\n")
		})
	case s.clicks == konst.WordClicks:
		s.expand(at, func(a, b glyph) bool { return a.node == b.node && a.to == b.from && wordy(a) && wordy(b) })
	default:
		s.anchor, s.focus, s.dragging = s.glyphs[at].spot, s.glyphs[at].spot, true
	}
	r.dirty = r.dirty || s.shown
}

func wordy(g glyph) bool {
	c, _ := utf8.DecodeRuneInString(g.Cluster)
	return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_'
}

func (s *selection) expand(at int, joins func(a, b glyph) bool) {
	lo, hi := at, at
	for lo > 0 && joins(s.glyphs[lo-1], s.glyphs[lo]) {
		lo--
	}
	for hi+1 < len(s.glyphs) && joins(s.glyphs[hi], s.glyphs[hi+1]) {
		hi++
	}
	s.anchor, s.focus, s.shown = s.glyphs[lo].spot, s.glyphs[hi].spot, true
}

func (s *selection) near(x, y int) int {
	b := s.bounds
	x, y = min(max(x, b.X), b.X+b.W-1), min(max(y, b.Y), b.Y+b.H-1)
	best, far := 0, [2]int{}
	for i, g := range s.glyphs {
		d := [2]int{max(g.Y-y, y-g.Y), max(g.X-x, x-g.X-g.Width+1, 0)}
		if i == 0 || d[0] < far[0] || d[0] == far[0] && d[1] < far[1] {
			best, far = i, d
		}
	}
	return best
}

func (r *Runtime) extend() {
	s := &r.sel
	if !s.dragging {
		return
	}
	at := r.pointer.at
	focus := s.glyphs[s.near(at.X, at.Y)].spot
	shown := s.shown || at.X != s.press.X || at.Y != s.press.Y
	if focus != s.focus || shown != s.shown {
		s.focus, s.shown, r.dirty = focus, shown, true
	}
}

func (r *Runtime) descend(path []int, visit func(depth int, n *scene.Node, st style.ComputedStyle)) (*scene.Node, render.Node, style.ComputedStyle, bool) {
	n, node := &r.scene, r.nodes
	st := r.computed(style.ComputedStyle{}, node)
	for depth := 0; ; depth++ {
		if visit != nil {
			visit(depth, n, st)
		}
		if depth == len(path) {
			return n, node, st, true
		}
		i := path[depth]
		if i >= len(n.Children) || i >= len(node.Children) {
			return nil, render.Node{}, st, false
		}
		n, node = &n.Children[i], node.Children[i]
		st = r.computed(st, node)
	}
}

func (r *Runtime) computed(parent style.ComputedStyle, node render.Node) style.ComputedStyle {
	var state style.NodeState
	if node.State != nil {
		state = *node.State
	}
	return r.cfg.Sheet.WithColumns(r.width).ComputeState(parent, node.Classes, state)
}

func (r *Runtime) flow() {
	s := &r.sel
	if !s.active {
		return
	}
	n, node, st, ok := r.descend(s.root, nil)
	if !ok {
		s.clear()
		return
	}
	s.bounds, s.glyphs = n.Bounds, s.glyphs[:0]
	k := 0
	var walk func(n *scene.Node, node render.Node, st style.ComputedStyle, path []int)
	walk = func(n *scene.Node, node render.Node, st style.ComputedStyle, path []int) {
		if node.Text != "" && st.UserSelect != style.SelectNone {
			t := selectable{path: slices.Clone(path), raw: node.Text, clip: n.Clip}
			if k < len(s.texts) && s.texts[k].raw == node.Text {
				t.clean = s.texts[k].clean
			} else {
				t.clean = text.Sanitize(node.Text, text.RemoveBidi)
			}
			if k < len(s.texts) {
				s.texts[k] = t
			} else {
				s.texts = append(s.texts, t)
			}
			pos := 0
			for g := range paint.Placed(n, text.Widths{}, n.Content) {
				for pos < len(t.clean) && !strings.HasPrefix(t.clean[pos:], g.Cluster) && (t.clean[pos] == ' ' || t.clean[pos] == '\n') {
					pos++
				}
				if strings.HasPrefix(t.clean[pos:], g.Cluster) {
					s.glyphs = append(s.glyphs, glyph{g, spot{k, pos, pos + len(g.Cluster)}})
					pos += len(g.Cluster)
				}
			}
			k++
		}
		for i := range min(len(n.Children), len(node.Children)) {
			walk(&n.Children[i], node.Children[i], r.computed(st, node.Children[i]), append(path, i))
		}
	}
	walk(n, node, st, slices.Clone(s.root))
	s.texts = s.texts[:k]
	if s.anchor.node >= k || s.focus.node >= k {
		s.clear()
	}
}

func (s *selection) text() string {
	lo, hi := s.span()
	var b strings.Builder
	var last *glyph
	for i := range s.glyphs {
		g := &s.glyphs[i]
		if g.before(lo) || hi.before(g.spot) {
			continue
		}
		switch {
		case last == nil:
		case last.node == g.node:
			b.WriteString(s.texts[g.node].clean[last.to:g.from])
		case last.Y != g.Y:
			b.WriteByte('\n')
		case g.X > last.X+last.Width:
			b.WriteByte(' ')
		}
		b.WriteString(g.Cluster)
		last = g
	}
	return b.String()
}

func (r *Runtime) copySelection() error {
	if r.cfg.NoClipboard {
		return nil
	}
	_, err := r.out.Write(terminal.Clipboard(r.sel.text()))
	return err
}

func (r *Runtime) highlight(root scene.Node) scene.Node {
	s := &r.sel
	if !s.shown {
		return root
	}
	lo, hi := s.span()
	fill := color.Color{Kind: color.Literal, RGBA: color.RGBA{R: pkonst.SelectionRed, G: pkonst.SelectionGreen, B: pkonst.SelectionBlue, A: pkonst.SelectionAlpha}}
	runs := make([][]scene.Node, len(s.texts))
	for _, g := range s.glyphs {
		if g.before(lo) || hi.before(g.spot) {
			continue
		}
		boxes := runs[g.node]
		if last := len(boxes) - 1; last >= 0 && boxes[last].Bounds.Y == g.Y && boxes[last].Bounds.X+boxes[last].Bounds.W == g.X {
			boxes[last].Bounds.W += g.Width
			boxes[last].Padding, boxes[last].Content = boxes[last].Bounds, boxes[last].Bounds
			continue
		}
		box := layout.Rect{X: g.X, Y: g.Y, W: g.Width, H: 1}
		runs[g.node] = append(boxes, scene.Node{Bounds: box, Padding: box, Content: box, Clip: s.texts[g.node].clip, Opacity: 1, Background: fill})
	}
	for k := len(runs) - 1; k >= 0; k-- {
		if len(runs[k]) > 0 {
			root = underneath(root, s.texts[k].path, runs[k])
		}
	}
	return root
}

func underneath(n scene.Node, path []int, boxes []scene.Node) scene.Node {
	if len(path) == 0 {
		n.Children = slices.Concat(n.Children, boxes)
		return n
	}
	i := path[0]
	if len(path) == 1 {
		n.Children = slices.Concat(n.Children[:i], boxes, n.Children[i:])
		return n
	}
	n.Children = slices.Clone(n.Children)
	n.Children[i] = underneath(n.Children[i], path[1:], boxes)
	return n
}
