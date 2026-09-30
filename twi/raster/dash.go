package raster

import (
	"fmt"
	"image"
	"math"

	konst "github.com/twind-dev/twind/internal/konst/raster"
)

type Dash uint8

const (
	Solid Dash = iota
	Dashed
	Dotted
)

type pattern struct{ dash, period, phase float64 }

func dashes(d Dash, width float64) (dash, gap float64) {
	switch {
	case d == Dotted:
		return konst.DotPerWidth * width, konst.DotGapPerWidth * width
	case d == Dashed && width >= konst.DashThickWidth:
		return konst.DashThick * width, konst.DashThickGap * width
	case d == Dashed:
		return konst.DashThin * width, konst.DashThinGap * width
	}
	panic(fmt.Sprintf("raster: no pattern for dash %d", d))
}

func loop(dash, gap, length float64) pattern {
	scale := length / (max(math.Round(length/(dash+gap)), 1) * (dash + gap))
	return pattern{dash: dash * scale, period: (dash + gap) * scale}
}

func (p pattern) cover(s float64) float32 {
	ink := func(t float64) float64 {
		n := math.Floor(t / p.period)
		return n*p.dash + min(t-n*p.period, p.dash)
	}
	return float32(ink(s+0.5+p.phase) - ink(s-0.5+p.phase))
}

func (b Box) perimeter() float64 {
	return 2*(b.W+b.H) + (math.Pi/2-2)*(b.Radii[0]+b.Radii[1]+b.Radii[2]+b.Radii[3])
}

func (b Box) along(px, py float64) float64 {
	frames := [4][3]float64{
		{px - b.X, py - b.Y, b.W},
		{py - b.Y, b.X + b.W - px, b.H},
		{b.X + b.W - px, b.Y + b.H - py, b.W},
		{b.Y + b.H - py, px - b.X, b.H},
	}
	best, at, offset := math.Inf(1), 0.0, 0.0
	for k, f := range frames {
		u, v, length := f[0], f[1], f[2]
		start, end := b.Radii[k], b.Radii[(k+1)%4]
		straight := length - start - end
		t := min(max(u-start, 0), straight)
		d, s := math.Hypot(u-start-t, v), t
		if end > 0 && u > length-end {
			dx, dy := u-(length-end), v-end
			d, s = math.Abs(math.Hypot(dx, dy)-end), straight+end*min(max(math.Atan2(dx, -dy), 0), math.Pi/2)
		}
		if d < best {
			best, at = d, offset+s
		}
		offset += straight + end*math.Pi/2
	}
	return at
}

func (r *Raster) dashedBorder(op Op, outer, inner *outline, area image.Rectangle) {
	line := outer.box.inset(op.Width / 2)
	dash, gap := dashes(op.Dash, op.Width)
	p := loop(dash, gap, line.perimeter())
	p.phase = p.dash / 2
	paint, img := premul(op.Color), r.top().img
	for y := area.Min.Y; y < area.Max.Y; y++ {
		fy := float64(y) + 0.5
		lo, hi := outer.touched(y, area)
		holeLo, holeHi := inner.full(y, area)
		for x := lo; x < hi; x++ {
			if x >= holeLo && x < holeHi {
				x = holeHi - 1
				continue
			}
			fx := float64(x) + 0.5
			if cov := max(outer.cover(x, y)-inner.cover(x, y), 0); cov > 0 {
				i := img.PixOffset(x, y)
				blend(img.Pix[i:i+4], paint, cov*p.cover(line.along(fx, fy)))
			}
		}
	}
}
