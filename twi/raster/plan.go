package raster

import (
	"image"
	"math"
)

type frame struct {
	clip image.Rectangle
	nest int
}

func (r *Raster) plan(ops []Op, tile image.Rectangle) {
	r.first, r.same = tile.Min.Y, resize(r.same, tile.Dy())
	for i := range r.same {
		r.same[i] = i > 0
	}
	for len(r.outlines) < 2*len(ops) {
		r.outlines = append(r.outlines, outline{})
	}
	r.shapes, r.hide, r.hider, r.index = 0, image.Rectangle{}, 0, -1
	r.frames = append(r.frames[:0], frame{clip: tile})
	for i, op := range ops {
		f := r.frames[len(r.frames)-1]
		c := f.clip
		switch op.Kind {
		case Fill:
			b := op.Box.fit()
			lo, hi := b.straight(0)
			if len(op.Stops) > 0 || op.Dash != Solid && b.W < b.H {
				lo, hi = 1, 0
			}
			r.band(b.pixels(0.5).Intersect(c), lo, hi)
			r.halves(b)
			inside := image.Rectangle{image.Pt(int(math.Ceil(b.X))+1, int(math.Ceil(lo-0.5))+1), image.Pt(int(math.Floor(b.X+b.W))-1, int(math.Floor(hi-0.5)))}.Intersect(c)
			if f.nest == 0 && len(op.Stops) == 0 && op.Dash == Solid && op.Color.A == math.MaxUint8 && inside.Dx()*inside.Dy() > r.hide.Dx()*r.hide.Dy() {
				r.hide, r.hider = inside, i
			}
		case Border:
			outer := op.Box.fit()
			lo, hi := outer.straight(0)
			innerLo, innerHi := outer.inset(op.Width).straight(0)
			if op.Dash != Solid || !outer.even() {
				lo, hi = 1, 0
			}
			r.band(outer.pixels(0.5).Intersect(c), max(lo, innerLo), min(hi, innerHi))
		case Shadow:
			g := geometry(op)
			if !g.box.even() {
				g.bandLo, g.bandHi = 1, 0
			}
			r.band(g.area.Intersect(c), g.bandLo, g.bandHi)
		case Opacity:
			r.band(c, float64(c.Min.Y)+0.5, float64(c.Max.Y)-0.5)
			r.frames = append(r.frames, frame{clip: c, nest: f.nest + 1})
		case Clip:
			b := op.Box
			if b.Radii == [4]float64{} {
				r.frames = append(r.frames, frame{clip: c.Intersect(image.Rect(round(b.X), round(b.Y), round(b.X+b.W), round(b.Y+b.H))), nest: f.nest})
				continue
			}
			c = c.Intersect(b.pixels(0))
			lo, hi := b.fit().straight(0)
			r.band(c, lo, hi)
			r.halves(b.fit())
			r.frames = append(r.frames, frame{clip: c, nest: f.nest + 1})
		case Pop:
			r.frames = r.frames[:max(len(r.frames)-1, 1)]
		}
	}
}

func (r *Raster) band(area image.Rectangle, lo, hi float64) {
	if area.Empty() {
		return
	}
	first, last := int(math.Ceil(lo-0.5)), int(math.Floor(hi-0.5))
	if first > last {
		r.vary(area.Min.Y, area.Max.Y+1)
		return
	}
	r.vary(area.Min.Y, max(first+2, area.Min.Y+1))
	r.vary(min(last, area.Max.Y), area.Max.Y+1)
}

func (r *Raster) halves(b Box) {
	if !b.even() {
		mid := int(math.Floor(b.Y + b.H/2))
		r.vary(mid-1, mid+2)
	}
}

func (r *Raster) vary(from, to int) {
	for y := max(from, r.first); y < min(to, r.first+len(r.same)); y++ {
		r.same[y-r.first] = false
	}
}

func (r *Raster) skip(y int) bool { return r.same[y-r.first] }

func (r *Raster) open(y, lo, hi int) [2][2]int {
	h := r.hide
	if r.index >= r.hider || r.depth > 0 || y < h.Min.Y || y >= h.Max.Y || hi <= h.Min.X || lo >= h.Max.X {
		return [2][2]int{{lo, hi}, {hi, hi}}
	}
	return [2][2]int{{lo, max(lo, h.Min.X)}, {min(hi, h.Max.X), hi}}
}

func (b Box) even() bool { return b.Radii[0] == b.Radii[3] && b.Radii[1] == b.Radii[2] }
