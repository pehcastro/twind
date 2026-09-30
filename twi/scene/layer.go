package scene

import (
	"cmp"
	"image"
	"slices"

	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/raster"
)

type Frame struct {
	Layers []Layer
	cell   image.Point
	screen layout.Rect
	ops    []raster.Op
	runs   []run
}

type Layer struct {
	Parent  int
	Origin  image.Point
	Opacity float64
	Ops     []raster.Op
	Visual  image.Rectangle
	key     uint64
	hash    uint64
	runs    []run
	opsFrom int
	runFrom int
}

type run struct {
	key, hash uint64
	visual    image.Rectangle
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
	f.cell, f.screen = cell, root.Clip
	f.Layers, f.ops, f.runs = f.Layers[:0], f.ops[:0], f.runs[:0]
	f.promote(stack(root, hashSeed), -1)
	opsTo, runTo := len(f.ops), len(f.runs)
	for i := len(f.Layers) - 1; i >= 0; i-- {
		l := &f.Layers[i]
		l.Ops, l.runs = f.ops[l.opsFrom:opsTo:opsTo], f.runs[l.runFrom:runTo:runTo]
		opsTo, runTo = l.opsFrom, l.runFrom
		l.hash = l.key
		for _, r := range l.runs {
			l.hash = mix(mix(l.hash, r.key), r.hash)
			l.Visual = l.Visual.Union(r.visual.Add(l.Origin))
		}
	}
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

func (f *Frame) promote(ctx *context, parent int) {
	c := &chunk{first: len(f.Layers)}
	f.Layers = append(f.Layers, Layer{
		Parent: parent, Origin: f.pixels(ctx.node.Bounds).Min, Opacity: ctx.node.Opacity,
		key: mix(mix(ctx.key, 0), 0), opsFrom: len(f.ops), runFrom: len(f.runs),
	})
	f.paint(ctx, c)
}

func (f *Frame) paint(ctx *context, c *chunk) {
	f.draw(ctx.flow[0], c)
	for _, s := range ctx.below {
		f.child(s, c)
	}
	for _, e := range ctx.flow[1:] {
		if e.scroll != nil {
			f.child(e.scroll, c)
			continue
		}
		f.draw(e, c)
	}
	for _, s := range ctx.level {
		f.child(s, c)
	}
	for _, s := range ctx.above {
		f.child(s, c)
	}
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
			key: mix(f.Layers[c.first].key, uint64(c.count)), opsFrom: start, runFrom: len(f.runs),
		})
	}
	f.runs = append(f.runs, run{key: e.key, hash: hash(f.ops[start:]), visual: visual})
}
