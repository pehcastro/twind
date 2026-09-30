package chart

import (
	"math"
	"strconv"
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/chart"
)

type plot struct {
	kind      Kind
	stacked   bool
	points    int
	base, top [][]float64
	ticks     []float64
}

func (c *Chart) model() plot {
	p := plot{kind: c.Kind, stacked: c.Stacked, points: len(c.Labels)}
	lo, hi := 0.0, 0.0
	for s, series := range c.Series {
		if len(series.Values) != p.points {
			panic("chart: series " + strconv.Itoa(s) + " has " + strconv.Itoa(len(series.Values)) + " values for " + strconv.Itoa(p.points) + " labels")
		}
		base, top := make([]float64, p.points), make([]float64, p.points)
		for i, v := range series.Values {
			if c.Stacked && s > 0 {
				base[i] = p.top[s-1][i]
			}
			top[i] = base[i] + v
			lo, hi = min(lo, base[i], top[i]), max(hi, base[i], top[i])
		}
		p.base, p.top = append(p.base, base), append(p.top, top)
	}
	p.ticks = niceTicks(lo, hi)
	return p
}

func niceTicks(lo, hi float64) []float64 {
	ticks := make([]float64, 0, konst.TickCount)
	if lo == hi {
		for i := range konst.TickCount {
			ticks = append(ticks, float64(i))
		}
		return ticks
	}
	rough := (hi - lo) / (konst.TickCount - 1)
	digits := int(math.Floor(math.Log10(rough))) + 1
	mantissa := konst.StepMantissa
	if digits == 1 {
		mantissa = konst.FirstDigitMantissa
	}
	exp := digits - konst.StepShift
	for k := ceil(rough / scaled(mantissa, exp)); ; k++ {
		m := k * mantissa
		step := scaled(m, exp)
		below, up := ceil(-lo/step), ceil(hi/step)
		spare := konst.TickCount - (below + up + 1)
		if spare < 0 {
			continue
		}
		if hi > 0 {
			up += spare
		} else {
			below += spare
		}
		for j := -below; j <= up; j++ {
			ticks = append(ticks, scaled(j*m, exp))
		}
		return ticks
	}
}

func scaled(m, exp int) float64 {
	if exp < 0 {
		return float64(m) / math.Pow10(-exp)
	}
	return float64(m) * math.Pow10(exp)
}

func ceil(v float64) int { return int(math.Ceil(v - math.Abs(v)*konst.CeilSlack)) }

func number(v float64) string {
	s := strconv.FormatFloat(math.Abs(v), 'f', -1, 64)
	whole, frac, _ := strings.Cut(s, ".")
	var out strings.Builder
	if v < 0 {
		out.WriteByte('-')
	}
	for i, d := range whole {
		if i > 0 && (len(whole)-i)%konst.ThousandsEvery == 0 {
			out.WriteByte(',')
		}
		out.WriteRune(d)
	}
	if frac != "" {
		out.WriteString("." + frac)
	}
	return out.String()
}

type metrics struct {
	margin, half, dot, active, radius, gap, grid float32
}

func (p plot) y(v float64, h float32, m metrics) float32 {
	top, bottom := p.ticks[len(p.ticks)-1], p.ticks[0]
	return m.margin + float32((top-v)/(top-bottom))*(h-m.margin)
}

func (p plot) series(s int, w, h float32, m metrics, out []point) []point {
	out = out[:0]
	for i, v := range p.top[s] {
		out = append(out, point{(float32(i) + 0.5) * w / float32(p.points), p.y(v, h, m)})
	}
	return out
}

func (p plot) bar(i, s int, w float32, m metrics) (x0, x1 float32) {
	band := w / float32(p.points)
	n, slot := float32(len(p.top)), float32(s)
	if p.stacked {
		n, slot = 1, 0
	}
	width := max(float32(math.Floor(float64((band*(1-2*konst.CategoryGap)-(n-1)*m.gap)/n))), 1)
	x0 = round((float32(i)+0.5)*band-(n*width+(n-1)*m.gap)/2) + slot*(width+m.gap)
	return x0, x0 + width
}

func round(v float32) float32 { return float32(math.Round(float64(v))) }
