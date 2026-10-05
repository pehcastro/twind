package scene

import (
	"image"

	konst "github.com/pehcastro/twind/internal/konst/scene"
	"github.com/pehcastro/twind/twi/layout"
	"github.com/pehcastro/twind/twi/raster"
	"github.com/pehcastro/twind/twi/style"
)

type Frame struct {
	Layers []Layer
	cell   image.Point
	screen layout.Rect
	root   *Node
	ops    []raster.Op
	boxes  []Box
	looks  looks
	stamps stamps
	scratch
	stacker
}

type Walker struct{ stacker }

type stacker struct {
	contexts []context
	entries  []entry
	top      chain
	painted  bool
	probing  bool
	probe    image.Point
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
	if cell != f.cell {
		f.stamps.reset()
	}
	f.cell, f.screen, f.root, f.painted = cell, root.Clip, root, true
	f.Layers, f.ops, f.boxes = f.Layers[:0], f.ops[:0], f.boxes[:0]
	grown := cap(f.ops)
	f.promote(f.page(root), -1, false)
	if cap(f.ops) != grown {
		from := 0
		for i := range f.boxes {
			b := &f.boxes[i]
			end := from + len(b.Ops)
			b.Ops, from = f.ops[from:end:end], end
		}
	}
	opsTo, boxTo := len(f.ops), len(f.boxes)
	for i := len(f.Layers) - 1; i >= 0; i-- {
		l := &f.Layers[i]
		l.Ops, l.Boxes, l.Visual = f.ops[l.opsFrom:opsTo:opsTo], f.boxes[l.boxFrom:boxTo:boxTo], l.Visual.Intersect(l.Clip)
		opsTo, boxTo = l.opsFrom, l.boxFrom
	}
}

func (w *Walker) Hit(root *Node, x, y int, hits func(*Node) bool) (top *Node) {
	w.probing, w.probe = true, image.Pt(x, y)
	w.Walk(root, func(n *Node) {
		if hits(n) {
			top = n
		}
	}, func(_ *Node, inside func()) { inside() })
	w.probing = false
	return top
}

func (w *Walker) Walk(root *Node, draw func(*Node), group func(n *Node, inside func())) {
	ctx := w.page(root)
	group(root, func() { w.walk(ctx, draw, group) })
}

func (w *Walker) walk(ctx int32, draw func(*Node), group func(n *Node, inside func())) {
	w.visit(ctx, func(run []entry) {
		for _, e := range run {
			draw(e.node)
		}
	}, func(c int32) { group(w.contexts[c].node, func() { w.walk(c, draw, group) }) })
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
		if s.probing && !c.Holds(s.probe.X, s.probe.Y) {
			continue
		}
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
			if !s.painted || paints(c) {
				s.push(c, ck, round)
			}
		default:
			s.collect(ctx, c, ck, round)
		}
	}
}

func (s *stacker) visit(c int32, run func([]entry), child func(int32)) {
	ctx := s.contexts[c]
	each := func(l chain) {
		for at := l.head; at != 0; at = s.contexts[at].next {
			child(at)
		}
	}
	flow := s.entries[ctx.first:ctx.end]
	run(flow[:1])
	each(ctx.below)
	from := 1
	for at := 1; at < len(flow); {
		opens := flow[at].opens
		if opens == 0 {
			at++
			continue
		}
		run(flow[from:at])
		if s.contexts[opens].scroll {
			child(opens)
		}
		at = int(s.contexts[opens].end - ctx.first)
		from = at
	}
	run(flow[from:])
	each(ctx.level)
	each(ctx.above)
	each(ctx.top)
}

func (f *Frame) promote(ctx int32, parent int, scroll bool) {
	n, key := f.contexts[ctx].node, mix(mix(f.contexts[ctx].key, 0), 0)
	l := Layer{
		Parent: parent, Origin: f.pixels(n.Bounds).Min, Opacity: n.Opacity, Clip: f.pixels(f.screen),
		key: key, hash: key, opsFrom: len(f.ops), boxFrom: len(f.boxes), scroller: scroll,
	}
	if scroll {
		l.Origin, l.Clip = f.pixels(n.ScrollContent).Min, f.surface(n.Padding, n.Halves.Padding).Intersect(f.surface(n.Clip, n.Halves.Clip))
	}
	c := &chunk{first: len(f.Layers)}
	f.Layers = append(f.Layers, l)
	if !scroll {
		n = nil
	}
	f.paint(ctx, c, n)
}

func (f *Frame) paint(ctx int32, c *chunk, skip *Node) {
	f.visit(ctx, func(run []entry) {
		for i := range run {
			if run[i].node != skip {
				f.draw(&run[i], c)
			}
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
		f.draw(&f.entries[f.contexts[ctx].first], c)
		f.promote(ctx, c.first, true)
		c.closed = true
		l, start := &f.Layers[c.first], len(f.ops)
		visual, look, ok := f.thumb(n, l.Origin, l.Clip)
		f.commit(mix(f.contexts[ctx].key, 0), visual, look, ok, start, c)
	default:
		f.paint(ctx, c, nil)
	}
}

func (f *Frame) draw(e *entry, c *chunk) {
	l, start := &f.Layers[c.first], len(f.ops)
	visual, look, ok := f.record(e.node, e.round, l.Origin, l.Clip)
	f.commit(e.key, visual, look, ok, start, c)
}

func (f *Frame) commit(key uint64, visual image.Rectangle, look uint64, ok bool, start int, c *chunk) {
	if !ok {
		f.ops = f.ops[:start]
		return
	}
	if c.closed {
		c.closed = false
		c.count++
		first := &f.Layers[c.first]
		sub := mix(first.key, uint64(c.count))
		f.Layers = append(f.Layers, Layer{
			Parent: c.first, Origin: first.Origin, Opacity: 1, Clip: first.Clip,
			key: sub, hash: sub, opsFrom: start, boxFrom: len(f.boxes),
		})
	}
	end, hash := len(f.ops), mix(look, uint64(visual.Min.X)<<32|uint64(uint32(visual.Min.Y)))
	f.boxes = append(f.boxes, Box{Visual: visual, Ops: f.ops[start:end:end], Look: look, key: key, hash: hash})
	l := &f.Layers[len(f.Layers)-1]
	l.hash = mix(mix(l.hash, key), hash)
	l.Visual = l.Visual.Union(visual.Add(l.Origin))
}
