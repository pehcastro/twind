package chart

import (
	_ "embed"
	"encoding/binary"
	"image"

	konst "github.com/pehcastro/twind/internal/konst/chart"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/text"
	"github.com/pehcastro/twind/twi/theme"
)

//go:embed world.bin
var worldData []byte

type country struct {
	code, name string
	rings      [][]point
	lo, hi     point
}

type atlas struct {
	w, h      float32
	countries []country
}

func decode(data []byte) *atlas {
	next := func() uint64 {
		v, n := binary.Uvarint(data)
		data = data[n:]
		return v
	}
	delta := func() float32 {
		v, n := binary.Varint(data)
		data = data[n:]
		return float32(v)
	}
	a := &atlas{w: float32(next()), h: float32(next())}
	a.countries = make([]country, next())
	var at point
	for i := range a.countries {
		k := &a.countries[i]
		k.code, data = string(data[:konst.CodeBytes]), data[konst.CodeBytes:]
		size := next()
		k.name, data = string(data[:size]), data[size:]
		k.rings = make([][]point, next())
		k.lo, k.hi = point{a.w, a.h}, point{}
		for j := range k.rings {
			ring := make([]point, next())
			for n := range ring {
				at = point{at.x + delta(), at.y + delta()}
				ring[n] = at
				k.lo, k.hi = point{min(k.lo.x, at.x), min(k.lo.y, at.y)}, point{max(k.hi.x, at.x), max(k.hi.y, at.y)}
			}
			k.rings[j] = ring
		}
	}
	return a
}

func (c *Chart) atlas() *atlas {
	if c.world == nil {
		c.world = decode(worldData)
	}
	return c.world
}

func (a *atlas) fit(w, h float32) (scale float32, offset point) {
	scale = min(w/a.w, h/a.h)
	return scale, point{(w - a.w*scale) / 2, (h - a.h*scale) / 2}
}

func (a *atlas) locate(w, h float32, q point) int {
	scale, offset := a.fit(w, h)
	g := point{(q.x - offset.x) / scale, (q.y - offset.y) / scale}
	for i, k := range a.countries {
		if g.x < k.lo.x || g.x > k.hi.x || g.y < k.lo.y || g.y > k.hi.y {
			continue
		}
		inside := false
		for _, ring := range k.rings {
			e := ring[len(ring)-1]
			for _, b := range ring {
				if (b.y > g.y) != (e.y > g.y) && g.x < b.x+(g.y-b.y)*(e.x-b.x)/(e.y-b.y) {
					inside = !inside
				}
				e = b
			}
		}
		if inside {
			return i + 1
		}
	}
	return 0
}

func (c *Chart) regions(p plot) []int {
	a := c.atlas()
	if len(c.Series) != 1 {
		panic("chart: a map shows one series")
	}
	of := make([]int, len(a.countries))
	for i := range of {
		of[i] = -1
	}
	for j, code := range p.labels {
		found := false
		for i, k := range a.countries {
			if k.code == code {
				of[i], found = j, true
			}
		}
		if !found {
			panic("chart: no country has the code " + text.Sanitize(code, text.RemoveBidi))
		}
	}
	return of
}

func (c *Chart) colour(r *raster, p plot, of []int, m metrics, hover int) {
	a := c.atlas()
	scale, offset := a.fit(float32(r.w), float32(r.h))
	lo, hi := p.extent()
	place := func(k country, closed bool) [][]point {
		out := r.rings[:0]
		for j, ring := range k.rings {
			if j >= len(r.rings) {
				r.rings = append(r.rings, nil)
			}
			pts := r.rings[j][:0]
			for _, g := range ring {
				pts = append(pts, point{offset.x + g.x*scale, offset.y + g.y*scale})
			}
			if closed {
				pts = append(pts, pts[0])
			}
			r.rings[j] = pts
			out = r.rings[:j+1]
		}
		return out
	}
	for i, k := range a.countries {
		r.polygon(place(k, false)...)
		fill := paint{token: theme.Muted, ink: c.quiet(theme.Muted), top: 1, bottom: 1}
		if j := of[i]; j >= 0 {
			f := float32(1)
			if hi > lo {
				f = float32((p.values[0][j] - lo) / (hi - lo))
			}
			l := faint
			switch {
			case f > konst.MapUpper:
				l = solid
			case f > konst.MapLower:
				l = half
			}
			fill = paint{token: c.Series[0].Color, ink: c.ink(0, l), top: konst.MapFloor + (1-konst.MapFloor)*f, dither: c.Dither}
			fill.bottom = fill.top
		}
		if hover == i+1 {
			fill.top, fill.bottom = 1, 1
			if of[i] < 0 {
				fill.token = theme.Border
			}
		}
		r.commit(fill)
	}
	if r.dst == nil {
		return
	}
	for _, k := range a.countries {
		for _, ring := range place(k, true) {
			r.stroke(ring, m.grid/2)
		}
	}
	r.commit(paint{top: 1, bottom: 1, erase: true})
}

func (c *Chart) choropleth(p plot) twi.Node {
	of := c.regions(p)
	w, h := c.Width, c.Height-1-konst.LegendGap
	c.halfs = halves(c.halfs, w, h)
	c.colour(c.halfs, p, of, metrics{grid: 1}, 0)
	hover := c.hover
	area := []twi.NodeOption{twi.Class("relative flex flex-col"),
		twi.OnPointerMove(func(e *twi.Event) {
			at := e.Offset()
			q := point{float32(at.X) + 0.5, (float32(at.Y) + 0.5) * konst.CellRows}
			c.point(c.atlas().locate(float32(w), float32(h*konst.CellRows), q), at)
		}),
		twi.OnPointerLeave(func(*twi.Event) { c.point(0, image.Point{}) }),
		surface(c.key(0), func(dst *image.RGBA, cell image.Point) {
			c.canvas = c.pixels(c.canvas, dst, cell)
			c.colour(c.canvas, p, of, pixelMetrics(cell), hover)
		}, c.glyphs(c.halfs, nil, image.Rectangle{}), w),
	}
	if c.hover > 0 {
		k := c.atlas().countries[c.hover-1]
		var rows []entry
		if j := of[c.hover-1]; j >= 0 {
			rows = []entry{{c.Series[0].Color, c.Series[0].Label, number(p.values[0][j])}}
		}
		area = append(area, tooltip(c.rt.Widths(), text.Sanitize(k.name, text.RemoveBidi), rows, beside(c.at.X, c.at.X+1, c.at.Y-1, w)))
	}
	return twi.Element(area...)
}
