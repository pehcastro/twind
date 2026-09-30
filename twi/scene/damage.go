package scene

import (
	"image"
	"math"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/raster"
)

const (
	hashSeed  = 0xcbf29ce484222325
	hashPrime = 0x9e3779b97f4a7c15
	hashShift = 29
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
	for i := range next.Layers {
		l := &next.Layers[i]
		j, ok := index[l.key]
		if !ok {
			d.add(l.Visual)
			continue
		}
		delete(index, l.key)
		p := &prev.Layers[j]
		if j < latest || p.Opacity != l.Opacity || parent(prev, p) != parent(next, l) {
			d.add(p.Visual)
			d.add(l.Visual)
			continue
		}
		latest = j
		if p.Origin != l.Origin {
			d.Moves = append(d.Moves, Move{Layer: i, From: p.Origin, To: l.Origin})
		}
		if p.hash != l.hash {
			d.runs(p, l)
		}
	}
	for _, p := range prev.Layers {
		if _, gone := index[p.key]; gone {
			d.add(p.Visual)
		}
	}
	return d
}

func (d *Damage) runs(p, l *Layer) {
	same := 0
	for ; same < min(len(p.runs), len(l.runs)) && p.runs[same].key == l.runs[same].key; same++ {
		if p.runs[same].hash != l.runs[same].hash {
			d.add(p.runs[same].visual.Add(l.Origin))
			d.add(l.runs[same].visual.Add(l.Origin))
		}
	}
	olds, news := p.runs[same:], l.runs[same:]
	index := make(map[uint64]int, len(olds))
	for j, r := range olds {
		index[r.key] = j
	}
	latest := -1
	for _, r := range news {
		j, ok := index[r.key]
		if !ok {
			d.add(r.visual.Add(l.Origin))
			continue
		}
		delete(index, r.key)
		if j < latest || olds[j].hash != r.hash {
			d.add(olds[j].visual.Add(l.Origin))
			d.add(r.visual.Add(l.Origin))
		}
		latest = max(latest, j)
	}
	for _, r := range olds {
		if _, gone := index[r.key]; gone {
			d.add(r.visual.Add(l.Origin))
		}
	}
}

func (d *Damage) add(r image.Rectangle) {
	if r.Empty() || len(d.Rects) > 0 && d.Rects[len(d.Rects)-1] == r {
		return
	}
	d.Rects = append(d.Rects, r)
}

func hash(ops []raster.Op) uint64 {
	h := uint64(hashSeed)
	for _, op := range ops {
		b, s := op.Box, op.Shadow
		for _, v := range [...]float64{
			float64(op.Kind), b.X, b.Y, b.W, b.H, b.Radii[0], b.Radii[1], b.Radii[2], b.Radii[3],
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
	h = (h ^ v) * hashPrime
	return h ^ h>>hashShift
}
