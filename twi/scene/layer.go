package scene

import (
	"image"

	konst "github.com/twind-dev/twind/internal/konst/scene"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/style"
)

type Frame struct {
	Layers []Layer
	cell   image.Point
	screen layout.Rect
	root   *Node
	ops    []raster.Op
	boxes  []Box
	looks  looks
	stacker
}

type Walker struct{ stacker }

type stacker struct {
	contexts []context
	entries  []entry
	top      chain
}

type chain struct{ head, tail int32 }

type Layer struct {
	Parent   int
	Origin   image.Point
	Opacity  float64
	Ops      []raster.Op
	Boxes    []Box
	Visual   image.Rectangle
	Clip     image.Rectangle
	key      uint64
	hash     uint64
	opsFrom  int
	boxFrom  int
	scroller bool
}

type Box struct {
	Visual    image.Rectangle
	Ops       []raster.Op
	Look      uint64
	key, hash uint64
	opsFrom   int
}

type context struct {
	node                     *Node
	key                      uint64
	first, end               int32
	below, level, above, top chain
	next                     int32
	scroll                   bool
}

type entry struct {
	node         *Node
	key          uint64
	round, opens int32
}

type chunk struct {
	first, count int
	closed       bool
}

func (f *Frame) Record(root *Node, cell image.Point) {
	f.cell, f.screen, f.root = cell, root.Clip, root
	f.Layers, f.ops, f.boxes = f.Layers[:0], f.ops[:0], f.boxes[:0]
	f.promote(f.page(root), -1, false)
	opsTo, boxTo := len(f.ops), len(f.boxes)
	for i := len(f.Layers) - 1; i >= 0; i-- {
		l := &f.Layers[i]
		l.Ops, l.Boxes = f.ops[l.opsFrom:opsTo:opsTo], f.boxes[l.boxFrom:boxTo:boxTo]
		for j := len(l.Boxes) - 1; j >= 0; j-- {
			b := &l.Boxes[j]
			b.Ops = f.ops[b.opsFrom:opsTo:opsTo]
			opsTo = b.opsFrom
		}
		opsTo, boxTo = l.opsFrom, l.boxFrom
		l.hash = l.key
		for j := range l.Boxes {
			b := &l.Boxes[j]
			l.hash = mix(mix(l.hash, b.key), b.hash)
			l.Visual = l.Visual.Union(b.Visual.Add(l.Origin))
		}
		l.Visual = l.Visual.Intersect(l.Clip)
	}
}

func Walk(root *Node, draw func(*Node), group func(n *Node, inside func())) {
	new(Walker).Walk(root, draw, group)
}

func (w *Walker) Walk(root *Node, draw func(*Node), group func(n *Node, inside func())) {
	ctx := w.page(root)
	group(root, func() { w.walk(ctx, draw, group) })
}

func (w *Walker) walk(ctx int32, draw func(*Node), group func(n *Node, inside func())) {
	w.visit(ctx, func(e entry) { draw(e.node) }, func(c int32) { group(w.contexts[c].node, func() { w.walk(c, draw, group) }) })
}

func (s *stacker) page(root *Node) int32 {
	if s.contexts == nil {
		nodes, opened := census(root)
		s.contexts, s.entries = make([]context, 1, opened+2), make([]entry, 1, nodes+1)
	}
	s.contexts, s.entries, s.top = s.contexts[:1], s.entries[:1], chain{}
	ctx := s.stack(root, konst.HashSeed, 0)
	s.contexts[ctx].top = s.top
	return ctx
}

func census(n *Node) (nodes, opened int) {
	nodes = 1
	for i := range n.Children {
		c := &n.Children[i]
		if c.TopLayer > 0 || c.Opacity < 1 || c.Position != layout.PositionStatic || c.Scroll {
			opened++
		}
		more, open := census(c)
		nodes, opened = nodes+more, opened+open
	}
	return nodes, opened
}

func (s *stacker) open(n *Node, key uint64, scroll bool) int32 {
	s.contexts = append(s.contexts, context{node: n, key: key, first: int32(len(s.entries)), scroll: scroll})
	return int32(len(s.contexts) - 1)
}

func (s *stacker) close(ctx int32) {
	s.entries[s.contexts[ctx].first].opens = ctx
	s.contexts[ctx].end = int32(len(s.entries))
}

func (s *stacker) stack(n *Node, key uint64, round int32) int32 {
	ctx := s.open(n, key, false)
	s.collect(ctx, n, key, round)
	s.close(ctx)
	return ctx
}

func (s *stacker) push(n *Node, key uint64, round int32) int32 {
	s.entries = append(s.entries, entry{node: n, key: key, round: round})
	return int32(len(s.entries) - 1)
}

func (s *stacker) insert(l *chain, c int32, rank func(*Node) int) {
	r := rank(s.contexts[c].node)
	prev, at := int32(0), l.head
	if l.tail != 0 && rank(s.contexts[l.tail].node) <= r {
		prev, at = l.tail, 0
	}
	for at != 0 && rank(s.contexts[at].node) <= r {
		prev, at = at, s.contexts[at].next
	}
	s.contexts[c].next = at
	if prev == 0 {
		l.head = c
	} else {
		s.contexts[prev].next = c
	}
	if at == 0 {
		l.tail = c
	}
}

func zIndex(n *Node) int { return n.ZIndex }

func topLayer(n *Node) int { return n.TopLayer }

func treeOrder(*Node) int { return 0 }

func (s *stacker) collect(ctx int32, n *Node, key uint64, round int32) {
	self := s.push(n, key, round)
	if n.HidesOverflow && n.Border.Radius != style.RadiusNone {
		round = self
	}
	for i := range n.Children {
		c, ck := &n.Children[i], mix(key, uint64(i)+1)
		positioned := c.Position != layout.PositionStatic
		switch {
		case c.TopLayer > 0:
			child := s.stack(c, ck, 0)
			s.insert(&s.top, child, topLayer)
		case c.Opacity < 1 || c.Position == layout.PositionFixed || positioned && c.ZIndex != 0:
			masks := round
			if c.Position == layout.PositionFixed {
				masks = 0
			}
			child := s.stack(c, ck, masks)
			switch parent := &s.contexts[ctx]; {
			case c.ZIndex < 0:
				s.insert(&parent.below, child, zIndex)
			case c.ZIndex > 0:
				s.insert(&parent.above, child, zIndex)
			default:
				s.insert(&parent.level, child, treeOrder)
			}
		case positioned:
			own := s.open(c, ck, false)
			s.insert(&s.contexts[ctx].level, own, treeOrder)
			s.collect(ctx, c, ck, round)
			s.close(own)
		case c.Scroll:
			own := s.open(c, ck, true)
			s.collect(ctx, c, ck, round)
			s.close(own)
		case len(c.Children) == 0:
			s.push(c, ck, round)
		default:
			s.collect(ctx, c, ck, round)
		}
	}
}

func (s *stacker) visit(c int32, draw func(entry), child func(int32)) {
	ctx := s.contexts[c]
	each := func(l chain) {
		for at := l.head; at != 0; at = s.contexts[at].next {
			child(at)
		}
	}
	flow := s.entries[ctx.first:ctx.end]
	draw(flow[0])
	each(ctx.below)
	for at := 1; at < len(flow); {
		e := flow[at]
		if e.opens == 0 {
			draw(e)
			at++
			continue
		}
		if s.contexts[e.opens].scroll {
			child(e.opens)
		}
		at = int(s.contexts[e.opens].end - ctx.first)
	}
	each(ctx.level)
	each(ctx.above)
	each(ctx.top)
}

func (f *Frame) promote(ctx int32, parent int, scroll bool) {
	n := f.contexts[ctx].node
	l := Layer{
		Parent: parent, Origin: f.pixels(n.Bounds).Min, Opacity: n.Opacity, Clip: f.pixels(f.screen),
		key: mix(mix(f.contexts[ctx].key, 0), 0), opsFrom: len(f.ops), boxFrom: len(f.boxes), scroller: scroll,
	}
	if scroll {
		l.Origin, l.Clip = f.pixels(n.ScrollContent).Min, f.pixels(n.Padding).Intersect(f.pixels(n.Clip))
	}
	c := &chunk{first: len(f.Layers)}
	f.Layers = append(f.Layers, l)
	f.visit(ctx, func(e entry) {
		if !scroll || e.node != n {
			f.draw(e, c)
		}
	}, func(s int32) { f.child(s, c) })
}

func (f *Frame) child(ctx int32, c *chunk) {
	n := f.contexts[ctx].node
	switch {
	case n.Opacity < 1 || n.Position == layout.PositionFixed:
		f.promote(ctx, c.first, false)
		c.closed = true
	case n.Scroll:
		f.draw(f.entries[f.contexts[ctx].first], c)
		f.promote(ctx, c.first, true)
		c.closed = true
		l, start := &f.Layers[c.first], len(f.ops)
		visual, ok := f.thumb(n, l.Origin, l.Clip)
		f.commit(mix(f.contexts[ctx].key, 0), visual, ok, start, c)
	default:
		f.visit(ctx, func(e entry) { f.draw(e, c) }, func(s int32) { f.child(s, c) })
	}
}

func (f *Frame) draw(e entry, c *chunk) {
	l, start := &f.Layers[c.first], len(f.ops)
	visual, ok := f.record(e.node, e.round, l.Origin, l.Clip)
	f.commit(e.key, visual, ok, start, c)
}

func (f *Frame) commit(key uint64, visual image.Rectangle, ok bool, start int, c *chunk) {
	if !ok {
		f.ops = f.ops[:start]
		return
	}
	if c.closed {
		c.closed = false
		c.count++
		first := &f.Layers[c.first]
		f.Layers = append(f.Layers, Layer{
			Parent: c.first, Origin: first.Origin, Opacity: 1, Clip: first.Clip,
			key: mix(first.key, uint64(c.count)), opsFrom: start, boxFrom: len(f.boxes),
		})
	}
	f.boxes = append(f.boxes, Box{})
	b := &f.boxes[len(f.boxes)-1]
	b.Visual, b.Look, b.key, b.opsFrom = visual, f.looks.look(f.ops[start:], visual), key, start
	b.hash = mix(b.Look, uint64(visual.Min.X)<<32|uint64(uint32(visual.Min.Y)))
}
