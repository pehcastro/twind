package scene

import (
	"image"
	"math"
	"slices"

	konst "github.com/twind-dev/twind/internal/konst/scene"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/raster"
)

type Move struct {
	Layer    int
	From, To image.Point
}

type Scroll struct {
	Layer int
	By    image.Point
}

type Damage struct {
	Rects   []image.Rectangle
	Moves   []Move
	Scrolls []Scroll
}

func Diff(prev, next *Frame) Damage {
	var d Damage
	index := make(map[uint64]int, len(prev.Layers))
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
	whole := make([]bool, len(next.Layers))
	for i := range next.Layers {
		l := &next.Layers[i]
		j, ok := index[l.key]
		if !ok {
			d.add(l.Visual)
			continue
		}
		delete(index, l.key)
		p := &prev.Layers[j]
		if j < latest || p.Opacity != l.Opacity || p.Clip != l.Clip || parent(prev, p) != parent(next, l) || l.Parent >= 0 && whole[l.Parent] {
			whole[i] = true
			d.add(p.Visual)
			d.add(l.Visual)
			continue
		}
		latest = j
		switch {
		case p.Origin == l.Origin:
		case l.scroller:
			d.Scrolls = append(d.Scrolls, Scroll{Layer: i, By: l.Origin.Sub(p.Origin)})
		default:
			d.Moves = append(d.Moves, Move{Layer: i, From: p.Origin, To: l.Origin})
		}
		if p.hash != l.hash {
			d.boxes(p, l)
		}
	}
	for _, p := range prev.Layers {
		if _, gone := index[p.key]; gone {
			d.add(p.Visual)
		}
	}
	return d
}

func (d *Damage) boxes(p, l *Layer) {
	same := 0
	for ; same < min(len(p.Boxes), len(l.Boxes)) && p.Boxes[same].key == l.Boxes[same].key; same++ {
		if p.Boxes[same].hash != l.Boxes[same].hash {
			d.add(p.Boxes[same].Visual.Add(l.Origin))
			d.add(l.Boxes[same].Visual.Add(l.Origin))
		}
	}
	olds, news := p.Boxes[same:], l.Boxes[same:]
	index := make(map[uint64]int, len(olds))
	for j, b := range olds {
		index[b.key] = j
	}
	latest := -1
	for _, b := range news {
		j, ok := index[b.key]
		if !ok {
			d.add(b.Visual.Add(l.Origin))
			continue
		}
		delete(index, b.key)
		if j < latest || olds[j].hash != b.hash {
			d.add(olds[j].Visual.Add(l.Origin))
			d.add(b.Visual.Add(l.Origin))
		}
		latest = max(latest, j)
	}
	for _, b := range olds {
		if _, gone := index[b.key]; gone {
			d.add(b.Visual.Add(l.Origin))
		}
	}
}

func (d *Damage) add(r image.Rectangle) {
	if r.Empty() || len(d.Rects) > 0 && d.Rects[len(d.Rects)-1] == r {
		return
	}
	d.Rects = append(d.Rects, r)
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

func same(a []raster.Op, aAt image.Point, b []raster.Op, bAt image.Point) bool {
	ne := func(u, v float64) uint64 { return math.Float64bits(u) ^ math.Float64bits(v) }
	ax, ay, bx, by := float64(aAt.X), float64(aAt.Y), float64(bAt.X), float64(bAt.Y)
	for i := range a {
		p, q := &a[i], &b[i]
		r, s := &p.Box.Radii, &q.Box.Radii
		u, v := &p.Shadow, &q.Shadow
		if ne(p.Box.X-ax, q.Box.X-bx)|ne(p.Box.Y-ay, q.Box.Y-by)|ne(p.Box.W, q.Box.W)|ne(p.Box.H, q.Box.H)|
			ne(r[0], s[0])|ne(r[1], s[1])|ne(r[2], s[2])|ne(r[3], s[3])|
			ne(p.Angle, q.Angle)|ne(p.Width, q.Width)|ne(p.Opacity, q.Opacity)|
			ne(u.X, v.X)|ne(u.Y, v.Y)|ne(u.Blur, v.Blur)|ne(u.Spread, v.Spread)|
			uint64(p.Kind^q.Kind)|uint64(p.Dash^q.Dash)|packed(p.Color)^packed(q.Color) != 0 || u.Inset != v.Inset ||
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
