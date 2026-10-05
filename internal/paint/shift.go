package paint

import (
	"github.com/pehcastro/twind/internal/buffer"
	konst "github.com/pehcastro/twind/internal/konst/paint"
	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/twi/style"
)

type view struct {
	node           int32
	at             layout.Rect
	offset, across int
}

func (p *Painter) view(n *scene.Node, i int32, screen layout.Rect) view {
	v := view{node: i, offset: n.ScrollContent.Y - n.Padding.Y, across: n.ScrollContent.X - n.Padding.X}
	for j := range n.Children {
		if n.Children[j].Clip != n.Children[0].Clip {
			return v
		}
	}
	if len(n.Children) > 0 {
		v.at = overlap(n.Children[0].Clip, screen)
	}
	return v
}

func (p *Painter) Moved() layout.Rect { return p.moved }

func (p *Painter) shift(buf *buffer.Buffer) {
	p.moved = layout.Rect{}
	if p.reshaped || buf != p.buf || buf.Width() != p.width || buf.Height() != p.height || len(p.rel) == 0 || len(p.wasRel) != len(p.rel) || len(p.views) != len(p.wasViews) {
		return
	}
	moved := -1
	for i, v := range p.views {
		switch was := p.wasViews[i]; {
		case v.node != was.node || v.at != was.at || v.across != was.across:
			return
		case v.offset == was.offset:
		case moved >= 0:
			return
		default:
			moved = i
		}
	}
	if moved < 0 {
		return
	}
	v, by := p.views[moved], p.views[moved].offset-p.wasViews[moved].offset
	a := v.at
	if a.W == 0 || max(by, -by) >= a.H {
		return
	}
	lo, hi := a.X/konst.DamageColumns, (a.X+a.W+konst.DamageColumns-1)/konst.DamageColumns
	region := layout.Rect{X: lo * konst.DamageColumns, Y: a.Y, W: min(hi*konst.DamageColumns, buf.Width()) - lo*konst.DamageColumns, H: a.H}
	for _, o := range p.ops {
		if o.step != drawStep {
			return
		}
	}
	for y := a.Y; y < a.Y+a.H; y++ {
		if p.wasWide[y] {
			return
		}
	}
	inside := int(v.node) + size(p.nodes[v.node])
	for i, n := range p.nodes {
		if i > int(v.node) && i < inside {
			continue
		}
		if o := overlap(p.areas[i], region); o.W > 0 && o.H > 0 && !p.still(n, region) {
			return
		}
	}
	p.moved = a
	for k := range a.H {
		dst := a.Y + k
		if by > 0 {
			dst = a.Y + a.H - 1 - k
		}
		if src := dst - by; src >= a.Y && src < a.Y+a.H {
			copy(buf.Row(dst)[a.X:a.X+a.W], buf.Row(src)[a.X:a.X+a.W])
		}
	}
	for y := a.Y; y < a.Y+a.H; y++ {
		src := y - by
		for t := lo; t < hi; t++ {
			i := y*p.columns + t
			if p.last[i] = ^p.tiles[i]; src >= a.Y && src < a.Y+a.H && p.rel[i] == p.wasRel[src*p.columns+t] {
				p.last[i] = p.tiles[i]
			}
		}
	}
}

func (p *Painter) still(n *scene.Node, r layout.Rect) bool {
	spans := func(a layout.Rect) bool { return a.Y <= r.Y && a.Y+a.H >= r.Y+r.H }
	misses := func(top, height int) bool { return height <= 0 || top+height <= r.Y || top >= r.Y+r.H }
	body := n.Bounds
	switch {
	case n.Gradient.Kind != style.GradientNone, len(n.InsetShadows) > 0, n.Halves != (scene.Halves{}), n.Opacity < 1:
		return false
	case !spans(overlap(body, n.Clip)):
		return false
	case n.Border.Top && !misses(body.Y, 1), n.Border.Bottom && !misses(body.Y+body.H-1, 1):
		return false
	case !misses(n.Content.Y, min(len(n.Lines(p.Widths)), n.Content.H)):
		return false
	}
	pad := overlap(n.Padding, n.Clip)
	return spans(pad) || misses(pad.Y, pad.H)
}
