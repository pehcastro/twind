package scene

import (
	"image"
	"math"

	konst "github.com/twind-dev/twind/internal/konst/scene"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/raster"
)

type Move struct {
	Layer    int
	From, To image.Point
}

type Damage struct {
	Rects []image.Rectangle
	Moves []Move
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
		if j < latest || p.Opacity != l.Opacity || parent(prev, p) != parent(next, l) || l.Parent >= 0 && whole[l.Parent] {
			whole[i] = true
			d.add(p.Visual)
			d.add(l.Visual)
			continue
		}
		latest = j
		if p.Origin != l.Origin {
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

func hash(ops []raster.Op, at image.Point) uint64 {
	h := uint64(konst.HashSeed)
	x, y := float64(at.X), float64(at.Y)
	for _, op := range ops {
		b, s := op.Box, op.Shadow
		for _, v := range [...]float64{
			float64(op.Kind), b.X - x, b.Y - y, b.W, b.H, b.Radii[0], b.Radii[1], b.Radii[2], b.Radii[3],
			op.Angle, op.Width, op.Opacity, s.X, s.Y, s.Blur, s.Spread,
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
