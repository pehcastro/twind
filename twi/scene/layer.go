package scene

import (
	"cmp"
	"image"
	"slices"

	konst "github.com/twind-dev/twind/internal/konst/scene"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/raster"
)

type Frame struct {
	Layers []Layer
	cell   image.Point
	screen layout.Rect
	root   *Node
	ops    []raster.Op
	boxes  []Box
}

type Layer struct {
	Parent  int
	Origin  image.Point
	Opacity float64
	Ops     []raster.Op
	Boxes   []Box
	Visual  image.Rectangle
	key     uint64
	hash    uint64
	opsFrom int
	boxFrom int
}

type Box struct {
	Visual    image.Rectangle
	Ops       []raster.Op
	Look      uint64
	key, hash uint64
	opsFrom   int
}

type context struct {
	node                *Node
	key                 uint64
	flow                []entry
	below, level, above []*context
}

type entry struct {
	node   *Node
	key    uint64
	scroll *context
}

type chunk struct {
	first, count int
	closed       bool
}

func (f *Frame) Record(root *Node, cell image.Point) {
	f.cell, f.screen, f.root = cell, root.Clip, root
	f.Layers, f.ops, f.boxes = f.Layers[:0], f.ops[:0], f.boxes[:0]
	f.promote(stack(root, konst.HashSeed), -1)
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
		for _, b := range l.Boxes {
			l.hash = mix(mix(l.hash, b.key), b.hash)
			l.Visual = l.Visual.Union(b.Visual.Add(l.Origin))
		}
	}
}

func Walk(root *Node, draw func(*Node), group func(n *Node, inside func())) {
	var walk func(*context)
	walk = func(ctx *context) {
		ctx.visit(func(e entry) { draw(e.node) }, func(c *context) { group(c.node, func() { walk(c) }) })
	}
	group(root, func() { walk(stack(root, konst.HashSeed)) })
}

func stack(n *Node, key uint64) *context {
	ctx := &context{node: n, key: key}
	ctx.collect(n, key, ctx)
	byZ := func(a, b *context) int { return cmp.Compare(a.node.ZIndex, b.node.ZIndex) }
	slices.SortStableFunc(ctx.below, byZ)
	slices.SortStableFunc(ctx.above, byZ)
	return ctx
}

func (ctx *context) collect(n *Node, key uint64, into *context) {
	into.flow = append(into.flow, entry{node: n, key: key})
	for i := range n.Children {
		c, ck := &n.Children[i], mix(key, uint64(i)+1)
		positioned := c.Position != layout.PositionStatic
		switch {
		case c.Opacity < 1 || c.Position == layout.PositionFixed || positioned && c.ZIndex != 0:
			switch child := stack(c, ck); {
			case c.ZIndex < 0:
				ctx.below = append(ctx.below, child)
			case c.ZIndex > 0:
				ctx.above = append(ctx.above, child)
			default:
				ctx.level = append(ctx.level, child)
			}
		case positioned:
			own := &context{node: c, key: ck}
			ctx.level = append(ctx.level, own)
			ctx.collect(c, ck, own)
		case c.Scroll:
			own := &context{node: c, key: ck}
			into.flow = append(into.flow, entry{scroll: own})
			ctx.collect(c, ck, own)
		default:
			ctx.collect(c, ck, into)
		}
	}
}

func (ctx *context) visit(draw func(entry), child func(*context)) {
	draw(ctx.flow[0])
	for _, s := range ctx.below {
		child(s)
	}
	for _, e := range ctx.flow[1:] {
		if e.scroll != nil {
			child(e.scroll)
			continue
		}
		draw(e)
	}
	for _, s := range ctx.level {
		child(s)
	}
	for _, s := range ctx.above {
		child(s)
	}
}

func (f *Frame) promote(ctx *context, parent int) {
	c := &chunk{first: len(f.Layers)}
	f.Layers = append(f.Layers, Layer{
		Parent: parent, Origin: f.pixels(ctx.node.Bounds).Min, Opacity: ctx.node.Opacity,
		key: mix(mix(ctx.key, 0), 0), opsFrom: len(f.ops), boxFrom: len(f.boxes),
	})
	f.paint(ctx, c)
}

func (f *Frame) paint(ctx *context, c *chunk) {
	ctx.visit(func(e entry) { f.draw(e, c) }, func(s *context) { f.child(s, c) })
}

func (f *Frame) child(ctx *context, c *chunk) {
	n := ctx.node
	if n.Opacity == 1 && n.Position != layout.PositionFixed && !n.Scroll {
		f.paint(ctx, c)
		return
	}
	f.promote(ctx, c.first)
	c.closed = true
}

func (f *Frame) draw(e entry, c *chunk) {
	origin, start := f.Layers[c.first].Origin, len(f.ops)
	visual, ok := f.record(e.node, origin)
	if !ok {
		f.ops = f.ops[:start]
		return
	}
	if c.closed {
		c.closed = false
		c.count++
		f.Layers = append(f.Layers, Layer{
			Parent: c.first, Origin: origin, Opacity: 1,
			key: mix(f.Layers[c.first].key, uint64(c.count)), opsFrom: start, boxFrom: len(f.boxes),
		})
	}
	look := mix(hash(f.ops[start:], visual.Min), uint64(visual.Dx())<<32|uint64(visual.Dy()))
	at := uint64(visual.Min.X)<<32 | uint64(uint32(visual.Min.Y))
	f.boxes = append(f.boxes, Box{Visual: visual, Look: look, key: e.key, hash: mix(look, at), opsFrom: start})
}
