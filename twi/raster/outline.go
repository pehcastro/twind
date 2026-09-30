package raster

import (
	"image"
	"math"

	konst "github.com/twind-dev/twind/internal/konst/raster"
)

type outline struct {
	box   Box
	first int
	rows  []edges
	covs  []float32
	none  edges
}

type edges struct {
	ready, touches, fills          bool
	touchLo, touchHi, gapLo, gapHi int
	fullLo, fullHi, at             int
}

func (r *Raster) outline(b Box) *outline {
	for i := r.shapes - 1; i >= max(r.shapes-konst.OutlineWindow, 0); i-- {
		if r.outlines[i].box == b {
			return &r.outlines[i]
		}
	}
	o := &r.outlines[r.shapes]
	r.shapes++
	rows := b.pixels(0.5)
	first, last := max(rows.Min.Y, r.first), min(rows.Max.Y, r.first+len(r.same))
	o.box, o.first, o.rows, o.covs = b, first, resize(o.rows, max(last-first, 0)), o.covs[:0]
	clear(o.rows)
	return o
}

func (o *outline) row(y int) *edges {
	i := y - o.first
	if i < 0 || i >= len(o.rows) {
		return &o.none
	}
	e := &o.rows[i]
	if e.ready {
		return e
	}
	b, fy := o.box, float64(y)+0.5
	e.ready = true
	e.touches = fy > b.Y-0.5 && fy < b.Y+b.H+0.5
	e.fills = fy >= b.Y+0.5 && fy <= b.Y+b.H-0.5
	if e.touches {
		left, right := b.extent(fy, 0.5)
		e.touchLo, e.touchHi = int(math.Floor(left-0.5))+1, int(math.Ceil(right-0.5))
	}
	if e.fills {
		left, right := b.extent(fy, -0.5)
		e.fullLo, e.fullHi = int(math.Ceil(left-0.5)), int(math.Floor(right-0.5))+1
	}
	e.gapLo, e.gapHi = e.touchLo, e.touchLo
	if e.fills && e.fullLo < e.fullHi {
		e.gapLo, e.gapHi = min(max(e.fullLo, e.touchLo), e.touchHi), min(max(e.fullHi, e.touchLo), e.touchHi)
	}
	e.at = len(o.covs)
	for x := e.touchLo; x < e.gapLo; x++ {
		o.covs = append(o.covs, b.cover(float64(x)+0.5, fy))
	}
	for x := e.gapHi; x < e.touchHi; x++ {
		o.covs = append(o.covs, b.cover(float64(x)+0.5, fy))
	}
	return e
}

func (o *outline) touched(y int, area image.Rectangle) (int, int) {
	e := o.row(y)
	if !e.touches {
		return area.Min.X, area.Min.X
	}
	return within(e.touchLo, e.touchHi, area)
}

func (o *outline) full(y int, area image.Rectangle) (int, int) {
	e := o.row(y)
	if !e.fills {
		return area.Min.X, area.Min.X
	}
	return within(e.fullLo, e.fullHi, area)
}

func within(lo, hi int, area image.Rectangle) (int, int) {
	lo = min(max(lo, area.Min.X), area.Max.X)
	return lo, max(lo, min(hi, area.Max.X))
}

func (o *outline) cover(x, y int) float32 {
	e := o.row(y)
	switch {
	case x >= e.touchLo && x < e.gapLo:
		return o.covs[e.at+x-e.touchLo]
	case x >= e.gapHi && x < e.touchHi:
		return o.covs[e.at+e.gapLo-e.touchLo+x-e.gapHi]
	}
	return o.box.cover(float64(x)+0.5, float64(y)+0.5)
}
