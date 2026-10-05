package scene

import (
	"image"
	"math"
	"slices"

	konst "github.com/pehcastro/twind/internal/konst/scene"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/raster"
	"github.com/pehcastro/twind/twi/style"
)

type Move struct {
	Layer    int
	From, To image.Point
	Was      image.Rectangle
}

type Scroll struct {
	Layer int
	By    image.Point
}

type Damage struct {
	Rects   []image.Rectangle
	Layers  []int
	Moves   []Move
	Scrolls []Scroll
}

type scratch struct {
	damage        Damage
	layers, boxes map[uint64]int
	whole         []bool
}

func Diff(prev, next *Frame) Damage {
	s := &next.scratch
	if s.layers == nil {
		s.layers, s.boxes = map[uint64]int{}, map[uint64]int{}
	}
	d, index := Damage{Rects: s.damage.Rects[:0], Layers: s.damage.Layers[:0], Moves: s.damage.Moves[:0], Scrolls: s.damage.Scrolls[:0]}, s.layers
	clear(index)
	for i, l := range prev.Layers {
		index[l.key] = i
	}
	parent := func(f *Frame, l *Layer) uint64 {
		if l.Parent < 0 {
			return 0
		}
		return f.Layers[l.Parent].key
	}
	latest := -1
	s.whole = slices.Grow(s.whole[:0], len(next.Layers))[:len(next.Layers)]
	whole := s.whole
	clear(whole)
	for i := range next.Layers {
		l := &next.Layers[i]
		j, ok := index[l.key]
		if !ok {
			d.add(l.Visual, i)
			continue
		}
		delete(index, l.key)
		p := &prev.Layers[j]
		if j < latest || p.Opacity != l.Opacity || p.Clip != l.Clip || parent(prev, p) != parent(next, l) || l.Parent >= 0 && whole[l.Parent] {
			whole[i] = true
			d.add(p.Visual, i)
			d.add(l.Visual, i)
			continue
		}
		latest = j
		switch {
		case p.Origin == l.Origin:
		case l.scroller:
			d.Scrolls = append(d.Scrolls, Scroll{Layer: i, By: l.Origin.Sub(p.Origin)})
		default:
			d.Moves = append(d.Moves, Move{Layer: i, From: p.Origin, To: l.Origin, Was: p.Visual})
		}
		if p.hash != l.hash {
			d.boxes(p, l, i, s.boxes)
		}
	}
	for _, p := range prev.Layers {
		if _, gone := index[p.key]; gone {
			d.add(p.Visual, -1)
		}
	}
	s.damage = d
	return d
}

func (d *Damage) boxes(p, l *Layer, at int, index map[uint64]int) {
	same := 0
	for ; same < min(len(p.Boxes), len(l.Boxes)) && p.Boxes[same].key == l.Boxes[same].key; same++ {
		if p.Boxes[same].hash != l.Boxes[same].hash {
			d.add(p.Boxes[same].Visual.Add(l.Origin), at)
			d.add(l.Boxes[same].Visual.Add(l.Origin), at)
		}
	}
	olds, news := p.Boxes[same:], l.Boxes[same:]
	clear(index)
	for j, b := range olds {
		index[b.key] = j
	}
	latest := -1
	for _, b := range news {
		j, ok := index[b.key]
		if !ok {
			d.add(b.Visual.Add(l.Origin), at)
			continue
		}
		delete(index, b.key)
		if j < latest || olds[j].hash != b.hash {
			d.add(olds[j].Visual.Add(l.Origin), at)
			d.add(b.Visual.Add(l.Origin), at)
		}
		latest = max(latest, j)
	}
	for _, b := range olds {
		if _, gone := index[b.key]; gone {
			d.add(b.Visual.Add(l.Origin), at)
		}
	}
}

func (d *Damage) add(r image.Rectangle, layer int) {
	if n := len(d.Rects); r.Empty() || n > 0 && d.Rects[n-1] == r && d.Layers[n-1] == layer {
		return
	}
	d.Rects, d.Layers = append(d.Rects, r), append(d.Layers, layer)
}

type looks struct {
	slots [konst.LookSlots]memo
	ops   []raster.Op
}

type memo struct {
	look     uint64
	at, size image.Point
	from, to int
}

func (m *looks) look(ops []raster.Op, visual image.Rectangle) uint64 {
	colors := uint64(len(ops))
	for i := range ops {
		colors = colors<<8 ^ packed(ops[i].Color)
	}
	slot := &m.slots[mix(colors, math.Float64bits(ops[0].Box.W)^math.Float64bits(ops[0].Box.H)>>1)%konst.LookSlots]
	if slot.to-slot.from == len(ops) && slot.size == visual.Size() && same(m.ops[slot.from:slot.to], slot.at, ops, visual.Min) {
		return slot.look
	}
	look := mix(hash(ops, visual.Min), uint64(visual.Dx())<<32|uint64(visual.Dy()))
	if len(m.ops)+len(ops) > konst.LookOps {
		m.ops, m.slots = m.ops[:0], [konst.LookSlots]memo{}
	}
	*slot = memo{look: look, at: visual.Min, size: visual.Size(), from: len(m.ops), to: len(m.ops) + len(ops)}
	m.ops = append(m.ops, ops...)
	return look
}

type stamps struct {
	slots   [konst.StampSlots]stamp
	ops     []raster.Op
	shadows []style.Shadow
}

type stamp struct {
	size                  image.Point
	visual                image.Rectangle
	border                Border
	background            color.Color
	look                  uint64
	from, to              int
	shadows, insets, ends int
}

func (s *stamps) reset() {
	s.slots, s.ops, s.shadows = [konst.StampSlots]stamp{}, s.ops[:0], s.shadows[:0]
}

func (s *stamps) find(n *Node, size image.Point) *stamp {
	h := mix(uint64(size.X)<<32|uint64(uint32(size.Y)), packed(n.Background.RGBA)<<32|packed(n.Border.Color.RGBA))
	h = mix(h, uint64(n.Border.Radius)<<16|uint64(n.Border.Style)<<8|uint64(len(n.Shadows))<<4|uint64(len(n.InsetShadows)))
	return &s.slots[h%konst.StampSlots]
}

func (st *stamp) holds(s *stamps, n *Node, size image.Point, visual image.Rectangle) bool {
	return st.size == size && st.background == n.Background && st.border == n.Border && st.visual == visual &&
		slices.Equal(s.shadows[st.shadows:st.insets], n.Shadows) && slices.Equal(s.shadows[st.insets:st.ends], n.InsetShadows)
}

func (s *stamps) keep(st *stamp, n *Node, ops []raster.Op, bounds, visual image.Rectangle, look uint64) {
	if len(s.ops)+len(ops) > konst.StampOps || len(s.shadows)+len(n.Shadows)+len(n.InsetShadows) > konst.StampShadows {
		s.reset()
	}
	*st = stamp{size: bounds.Size(), visual: visual.Sub(bounds.Min), border: n.Border, background: n.Background, look: look, from: len(s.ops), shadows: len(s.shadows)}
	x, y := float64(bounds.Min.X), float64(bounds.Min.Y)
	for _, op := range ops {
		op.Box.X -= x
		op.Box.Y -= y
		s.ops = append(s.ops, op)
	}
	s.shadows = append(s.shadows, n.Shadows...)
	st.insets = len(s.shadows)
	s.shadows = append(s.shadows, n.InsetShadows...)
	st.to, st.ends = len(s.ops), len(s.shadows)
}

func same(a []raster.Op, aAt image.Point, b []raster.Op, bAt image.Point) bool {
	ne := func(u, v float64) uint64 { return math.Float64bits(u) ^ math.Float64bits(v) }
	ax, ay, bx, by := float64(aAt.X), float64(aAt.Y), float64(bAt.X), float64(bAt.Y)
	for i := range a {
		p, q := &a[i], &b[i]
		r, s := &p.Box.Radii, &q.Box.Radii
		u, v := &p.Shadow, &q.Shadow
		if ne(p.Box.X-ax, q.Box.X-bx)|ne(p.Box.Y-ay, q.Box.Y-by)|ne(p.Box.W, q.Box.W)|ne(p.Box.H, q.Box.H)|
			ne(r[0], s[0])|ne(r[1], s[1])|ne(r[2], s[2])|ne(r[3], s[3])|
			ne(p.Angle, q.Angle)|ne(p.Width, q.Width)|ne(p.Opacity, q.Opacity)|ne(p.Turn, q.Turn)|ne(p.Pivot.X, q.Pivot.X)|ne(p.Pivot.Y, q.Pivot.Y)|
			ne(u.X, v.X)|ne(u.Y, v.Y)|ne(u.Blur, v.Blur)|ne(u.Spread, v.Spread)|
			uint64(p.Kind^q.Kind)|uint64(p.Dash^q.Dash)|packed(p.Color)^packed(q.Color) != 0 || u.Inset != v.Inset || p.Pixels != q.Pixels ||
			!slices.EqualFunc(p.Stops, q.Stops, func(m, n raster.Stop) bool { return m.Color == n.Color && ne(m.At, n.At) == 0 }) {
			return false
		}
	}
	return true
}

func hash(ops []raster.Op, at image.Point) uint64 {
	h := uint64(konst.HashSeed)
	x, y := float64(at.X), float64(at.Y)
	for i := range ops {
		op := &ops[i]
		b, s := &op.Box, &op.Shadow
		for _, v := range [...]float64{
			float64(op.Kind), b.X - x, b.Y - y, b.W, b.H, b.Radii[0], b.Radii[1], b.Radii[2], b.Radii[3],
			op.Angle, op.Width, op.Opacity, s.X, s.Y, s.Blur, s.Spread, float64(op.Dash),
		} {
			h = mix(h, math.Float64bits(v))
		}
		inset := uint64(0)
		if s.Inset {
			inset = 1
		}
		h = mix(mix(h, packed(op.Color)), inset)
		if op.Kind == raster.Canvas {
			h = mix(h, op.Pixels.Key)
		}
		if op.Turn != 0 {
			h = mix(mix(mix(h, math.Float64bits(op.Turn)), math.Float64bits(op.Pivot.X)), math.Float64bits(op.Pivot.Y))
		}
		for _, stop := range op.Stops {
			h = mix(mix(h, packed(stop.Color)), math.Float64bits(stop.At))
		}
	}
	return h
}

func packed(c color.RGBA) uint64 {
	return uint64(c.R)<<24 | uint64(c.G)<<16 | uint64(c.B)<<8 | uint64(c.A)
}

func mix(h, v uint64) uint64 {
	h = (h ^ v) * konst.HashPrime
	return h ^ h>>konst.HashShift
}
