package paint

import (
	"cmp"
	"slices"

	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
)

type layer struct {
	z                   int
	opacity             float64
	flow                []*scene.Node
	below, level, above []*layer
}

func stack(root *scene.Node) *layer {
	l := &layer{z: root.ZIndex, opacity: root.Opacity}
	l.collect(root, l)
	byZ := func(a, b *layer) int { return cmp.Compare(a.z, b.z) }
	slices.SortStableFunc(l.below, byZ)
	slices.SortStableFunc(l.above, byZ)
	return l
}

func (ctx *layer) collect(n *scene.Node, into *layer) {
	into.flow = append(into.flow, n)
	for i := range n.Children {
		c := &n.Children[i]
		positioned := c.Position != layout.PositionStatic
		switch {
		case c.Opacity < 1 || c.Position == layout.PositionFixed || positioned && c.ZIndex != 0:
			switch child := stack(c); {
			case c.ZIndex < 0:
				ctx.below = append(ctx.below, child)
			case c.ZIndex > 0:
				ctx.above = append(ctx.above, child)
			default:
				ctx.level = append(ctx.level, child)
			}
		case positioned:
			own := &layer{opacity: 1}
			ctx.level = append(ctx.level, own)
			ctx.collect(c, own)
		default:
			ctx.collect(c, into)
		}
	}
}

func (l *layer) paint(buf *buffer.Buffer) {
	if l.opacity <= 0 {
		return
	}
	target := buf
	if l.opacity < 1 {
		target = buffer.New(buf.Width(), buf.Height())
		target.Fill(buffer.Rect{W: buf.Width(), H: buf.Height()}, buffer.Cell{Grapheme: " ", Bg: color.Color{Kind: color.Literal}})
	}
	draw(target, l.flow[0])
	for _, b := range l.below {
		b.paint(target)
	}
	for _, n := range l.flow[1:] {
		draw(target, n)
	}
	for _, c := range l.level {
		c.paint(target)
	}
	for _, a := range l.above {
		a.paint(target)
	}
	if target != buf {
		fade(buf, target, l.opacity)
	}
}
