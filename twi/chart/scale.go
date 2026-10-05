package chart

import (
	"math"
	"strconv"
	"strings"

	konst "github.com/pehcastro/twind/internal/konst/chart"
)

type plot struct {
	kind                      Kind
	stacked, horizontal, step bool
	lead, points              int
	labels                    []string
	values, base, top         [][]float64
	ticks                     []float64
}

func (c *Chart) model() plot { return c.window(c.From, c.To) }

func (c *Chart) window(from, to int) plot {
	if to == 0 {
		to = len(c.Labels)
	}
	if from < 0 || to > len(c.Labels) || from >= to {
		panic("chart: window " + strconv.Itoa(from) + " to " + strconv.Itoa(to) + " outside " + strconv.Itoa(len(c.Labels)) + " labels")
	}
	p := plot{kind: c.Kind, stacked: c.Stacked, horizontal: c.Horizontal && c.Kind == Bar, step: c.Step, points: to - from, labels: c.Labels[from:to]}
	lo, hi := 0.0, 0.0
	for s, series := range c.Series {
		if len(series.Values) != len(c.Labels) {
			panic("chart: series " + strconv.Itoa(s) + " has " + strconv.Itoa(len(series.Values)) + " values for " + strconv.Itoa(len(c.Labels)) + " labels")
		}
		values := series.Values[from:to]
		base, top := make([]float64, p.points), make([]float64, p.points)
		for i, v := range values {
			if v < 0 && c.Kind >= Pie {
				panic("chart: a pie, radial, radar or map value is not negative")
			}
			if c.Stacked && s > 0 {
				base[i] = p.top[s-1][i]
			}
			top[i] = base[i] + v
			lo, hi = min(lo, base[i], top[i]), max(hi, base[i], top[i])
		}
		p.values, p.base, p.top = append(p.values, values), append(p.base, base), append(p.top, top)
	}
	p.ticks = niceTicks(lo, hi)
	return p
}

func (p plot) extent() (lo, hi float64) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, v := range p.values[0] {
		lo, hi = min(lo, v), max(hi, v)
	}
	return lo, hi
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
	top, bottom, half, active, radius, gap, grid, dot float32
}

func (p plot) y(v float64, m metrics) float32 {
	top, bottom := p.ticks[len(p.ticks)-1], p.ticks[0]
	return m.top + float32((top-v)/(top-bottom))*(m.bottom-m.top)
}

func (p plot) rows(m metrics, h int, unit float32) metrics {
	span := p.snap(h - 1 - p.lead)
	m.top, m.bottom = (float32(h-1-span)+0.5)*unit, (float32(h)-0.5)*unit
	return m
}

func (p *plot) fit(h int) int {
	for span := p.snap(h - 1); ; span = p.snap(span - 1) {
		p.lead = 0
		m := p.halves(span + 1)
		for s, top := range p.top {
			for _, v := range top {
				if v >= 0 && (!p.stacked || s == len(p.top)-1) && int(p.y(v, m))/konst.CellRows == 0 {
					p.lead = 1
				}
			}
		}
		if span+1+p.lead <= h {
			return span + 1 + p.lead
		}
	}
}

func (p plot) snap(rows int) int {
	last := len(p.ticks) - 1
	if step := rows / last; step > 0 {
		return step * last
	}
	return rows
}

func (p plot) halves(h int) metrics {
	return p.rows(metrics{half: 0.5, gap: 1, grid: 1, dot: 1}, h, konst.CellRows)
}

func (p plot) row(v float64, h int) int { return int(p.y(v, p.halves(h))) / konst.CellRows }

func (p plot) series(s int, w float32, m metrics, out []point) []point {
	out = out[:0]
	for i, v := range p.top[s] {
		out = append(out, point{(float32(i) + 0.5) * w / float32(p.points), p.y(v, m)})
	}
	return out
}

func (p plot) bar(i, s int, length float32, m metrics) (lo, hi float32) {
	band := length / float32(p.points)
	n, slot := float32(len(p.top)), float32(s)
	if p.stacked {
		n, slot = 1, 0
	}
	width := max(float32(math.Floor(float64((band*(1-2*konst.CategoryGap)-(n-1)*m.gap)/n))), 1)
	lo = round((float32(i)+0.5)*band-(n*width+(n-1)*m.gap)/2) + slot*(width+m.gap)
	return lo, lo + width
}

func round(v float32) float32 { return float32(math.Round(float64(v))) }

func span(i, n, length int) (lo, hi int) { return i * length / n, (i + 1) * length / n }

func under(at, n, length int) int {
	for i := range n {
		if _, hi := span(i, n, length); at < hi {
			return i + 1
		}
	}
	return 0
}
