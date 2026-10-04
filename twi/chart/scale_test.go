package chart

import (
	"math"
	"slices"
	"testing"

	"github.com/twind-dev/twind/twi/theme"
)

func TestBarTicksFollowRecharts(t *testing.T) {
	for _, c := range []struct {
		lo, hi float64
		want   []float64
	}{
		{0, 305, []float64{0, 80, 160, 240, 320}},
		{0, 300, []float64{0, 75, 150, 225, 300}},
		{0, 0.3, []float64{0, 0.075, 0.15, 0.225, 0.3}},
		{0, 1, []float64{0, 0.25, 0.5, 0.75, 1}},
		{0, 7, []float64{0, 2, 4, 6, 8}},
		{0, 9.5, []float64{0, 3, 6, 9, 12}},
		{0, 1234, []float64{0, 350, 700, 1050, 1400}},
		{0, 0, []float64{0, 1, 2, 3, 4}},
		{-40, 120, []float64{-40, 0, 40, 80, 120}},
		{-50, 0, []float64{-60, -45, -30, -15, 0}},
		{0, 99999, []float64{0, 25000, 50000, 75000, 100000}},
		{0, 0.07, []float64{0, 0.02, 0.04, 0.06, 0.08}},
		{0, 530, []float64{0, 150, 300, 450, 600}},
		{0, 17, []float64{0, 5, 10, 15, 20}},
		{0, 0.14, []float64{0, 0.035, 0.07, 0.105, 0.14}},
		{0, 0.28, []float64{0, 0.07, 0.14, 0.21, 0.28}},
	} {
		if got := niceTicks(c.lo, c.hi); !slices.Equal(got, c.want) {
			t.Errorf("ticks of [%v, %v] = %v, recharts-scale 0.4.5 gives %v", c.lo, c.hi, got, c.want)
		}
	}
}

func TestBarLabelsPrintDecimalsAndThousands(t *testing.T) {
	for v, want := range map[float64]string{0.225: "0.225", 1400: "1,400", 100000: "100,000", -60: "-60", 1234567.5: "1,234,567.5", 0: "0"} {
		if got := number(v); got != want {
			t.Errorf("number(%v) = %q, want %q", v, got, want)
		}
	}
}

func TestBarStackSitsOnTheSeriesBelow(t *testing.T) {
	c := &Chart{Kind: Bar, Stacked: true, Labels: []string{"a", "b"}, Series: []Series{
		{Label: "x", Color: theme.Chart1, Values: []float64{1, 2}},
		{Label: "y", Color: theme.Chart2, Values: []float64{3, -1}},
		{Label: "z", Color: theme.Chart3, Values: []float64{4, 5}},
	}}
	p := c.model()
	wantBase := [][]float64{{0, 0}, {1, 2}, {4, 1}}
	wantTop := [][]float64{{1, 2}, {4, 1}, {8, 6}}
	for s := range wantTop {
		if !slices.Equal(p.base[s], wantBase[s]) || !slices.Equal(p.top[s], wantTop[s]) {
			t.Errorf("series %d: base %v top %v, want base %v top %v", s, p.base[s], p.top[s], wantBase[s], wantTop[s])
		}
	}
	if want := []float64{0, 2, 4, 6, 8}; !slices.Equal(p.ticks, want) {
		t.Errorf("stacked ticks %v, want %v from the tallest stack 8, not the largest value 5", p.ticks, want)
	}
	c.Stacked = false
	if p := c.model(); !slices.Equal(p.ticks, []float64{-2, 0, 2, 4, 6}) || !slices.Equal(p.base[2], []float64{0, 0}) {
		t.Errorf("grouped: ticks %v base %v, want [-2 0 2 4 6] and bars from zero", p.ticks, p.base[2])
	}
}

func TestLineMapsTicksToTheCanvas(t *testing.T) {
	c := &Chart{Kind: Line, Labels: []string{"a", "b", "c", "d"}, Series: []Series{{Label: "x", Color: theme.Chart1, Values: []float64{0, 305, 160, 80}}}}
	m := metrics{top: 5, bottom: 105}
	pts := c.model().series(0, 400, m, nil)
	want := []point{{50, 105}, {150, 5 + 100*15.0/320}, {250, 55}, {350, 80}}
	for i := range want {
		if math.Abs(float64(pts[i].x-want[i].x)) > 1e-3 || math.Abs(float64(pts[i].y-want[i].y)) > 1e-3 {
			t.Errorf("point %d at %v, want %v: band centres across, top tick 320 at the margin, zero at the bottom", i, pts[i], want[i])
		}
	}
}

func TestCanvasTicksSitOnTheirLabelRows(t *testing.T) {
	p := (&Chart{Kind: Line, Labels: []string{"a"}, Series: []Series{{Color: theme.Chart1, Values: []float64{305}}}}).model()
	for _, h := range []int{5, 9, 10, 14, 22} {
		pixels := p.rows(metrics{}, h, 20)
		for _, tick := range p.ticks {
			row := p.row(tick, h)
			if got, want := p.y(tick, pixels), float32(row)*20+10; got != want {
				t.Errorf("%d rows: tick %v at pixel %v, want %v, the middle of its label row %d", h, tick, got, want, row)
			}
		}
		if bottom := p.row(0, h); bottom != h-1 {
			t.Errorf("%d rows: zero labelled on row %d, want the last row", h, bottom)
		}
		if gap := p.row(p.ticks[3], h) - p.row(p.ticks[4], h); gap != p.row(0, h)-p.row(p.ticks[1], h) {
			t.Errorf("%d rows: ticks %d rows apart at the top and %d at the bottom, want even", h, gap, p.row(0, h)-p.row(p.ticks[1], h))
		}
	}
}

func TestBarBandsDoNotOverlap(t *testing.T) {
	c := &Chart{Kind: Bar, Labels: make([]string, 6), Series: []Series{
		{Color: theme.Chart1, Values: []float64{1, 2, 3, 4, 5, 6}},
		{Color: theme.Chart2, Values: []float64{6, 5, 4, 3, 2, 1}},
	}}
	p := c.model()
	var last float32
	for i := range 6 {
		for s := range 2 {
			x0, x1 := p.bar(i, s, 600, metrics{gap: 5})
			if x0 < last || x1 <= x0 || x0 < float32(i)*100+10 || x1 > float32(i+1)*100-10 {
				t.Errorf("bar %d/%d spans %v..%v: outside its band's 10%% gaps or over the bar before (ends %v)", i, s, x0, x1, last)
			}
			if s == 1 && x0-last != 5 {
				t.Errorf("bar %d: gap %v between grouped bars, want 5", i, x0-last)
			}
			last = x1
		}
	}
}
