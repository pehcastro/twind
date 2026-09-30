package chart

import (
	"bytes"
	"image"
	"image/png"
	"math"
	"os"
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/theme"
)

var visitors = []float64{186, 305, 237, 73, 209, 214, 190, 250, 160, 280, 120, 230}

func months() []string {
	return []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
}

func TestLineMaskMatchesResvg(t *testing.T) {
	raw, err := os.ReadFile("testdata/line-480x200.png")
	if err != nil {
		t.Fatal(err)
	}
	ref, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	c := &Chart{Kind: Line, Labels: months(), Series: []Series{{Label: "Desktop", Color: theme.Chart1, Values: visitors}}}
	const w, h = 480, 200
	m := pixelMetrics(image.Point{10, 20})
	r := reuse(nil, w, h)
	p := c.model()
	r.stroke(natural(nil, p.series(0, w, h, m, nil)), m.half)
	var sum, worst float64
	for y := range h {
		for x := range w {
			_, _, _, a := ref.At(x, y).RGBA()
			d := math.Abs(float64(min(r.cover[y*w+x], 1))*255 - float64(a>>8))
			sum += d
			worst = max(worst, d)
		}
	}
	mean := sum / (w * h)
	t.Logf("line mask against resvg 2.6.2 (d3 curveNatural, recharts-scale ticks): mean %.3f/255, worst %.0f/255", mean, worst)
	if mean > 0.5 || worst > 96 {
		t.Errorf("line mask differs from the reference: mean %.3f/255 (limit 0.5), worst %.0f/255 (limit 96)", mean, worst)
	}
}

func TestLineJoinsCoverTheUnionOnce(t *testing.T) {
	r := reuse(nil, 20, 20)
	r.stroke([]point{{4, 10}, {10, 10}, {10.0001, 10}, {18, 10}}, 2)
	for x := range 20 {
		if v := r.cover[10*20+x]; v > 1.0001 {
			t.Fatalf("pixel %d,10 covered %v at a join: coverage must be of the union", x, v)
		}
	}
	if v := r.cover[10*20+10]; v < 0.999 {
		t.Errorf("pixel 10,10 on the line covered %v, want 1", v)
	}
	if v := r.cover[10*20+1]; v != 0 {
		t.Errorf("pixel 1,10 beyond the round cap reaching x=2 covered %v", v)
	}
	if v := r.cover[10*20+2]; v <= 0.9 {
		t.Errorf("pixel 2,10 inside the round cap covered %v", v)
	}
}

func TestAreaFillsBetweenTheCurveAndTheSeriesBelow(t *testing.T) {
	c := &Chart{Kind: Area, Stacked: true, Labels: []string{"a", "b"}, Series: []Series{
		{Color: theme.Chart1, Values: []float64{2, 2}},
		{Color: theme.Chart2, Values: []float64{2, 2}},
	}}
	dst := image.NewRGBA(image.Rect(0, 0, 40, 40))
	c.Draw(dst, image.Point{8, 16}, palette)
	ink := dst.RGBAAt
	if o := ink(20, 35); o.R == 0 || o.B != 0 {
		t.Errorf("below the first curve %v: want chart-1 (red) fill", o)
	}
	if o := ink(20, 15); o.B == 0 || o.R != 0 {
		t.Errorf("between the curves %v: want chart-2 (blue) fill down to chart-1, not to zero", o)
	}
	if o := ink(20, 2); o.A != 0 {
		t.Errorf("above the top curve %v: want nothing", o)
	}
	if o := ink(5, 35); o.A != 0 {
		t.Errorf("left of the first point %v: the area starts at the first band centre", o)
	}
	if a, b := ink(20, 8).A, ink(20, 20).A; a <= b {
		t.Errorf("gradient alpha %d near the top of chart-2's area, %d near its bottom: want it falling downward", a, b)
	}
}

func TestBarStackRoundsOnlyTheOuterEnds(t *testing.T) {
	c := &Chart{Kind: Bar, Stacked: true, Labels: []string{"a"}, Series: []Series{
		{Color: theme.Chart1, Values: []float64{2}},
		{Color: theme.Chart2, Values: []float64{2}},
	}}
	dst := image.NewRGBA(image.Rect(0, 0, 100, 200))
	c.Draw(dst, image.Point{10, 20}, palette)
	left, right := 10, 89
	if a := dst.RGBAAt(left, 199).A; a == math.MaxUint8 {
		t.Errorf("bottom corner of the stack's first series is square (alpha %d): want rounded", a)
	}
	if a := dst.RGBAAt(right, 7).A; a == math.MaxUint8 {
		t.Errorf("top corner of the stack's last series is square (alpha %d): want rounded", a)
	}
	for _, y := range []int{100, 106} {
		for _, x := range []int{left, right} {
			if a := dst.RGBAAt(x, y).A; a != math.MaxUint8 {
				t.Errorf("corner %d,%d where the two series meet has alpha %d: inner ends stay square", x, y, a)
			}
		}
	}
}

func BenchmarkDraw(b *testing.B) {
	for _, kind := range []Kind{Bar, Line, Area} {
		c := threeSeries(kind, kind == Area)(nil)
		dst := image.NewRGBA(image.Rect(0, 0, 900, 440))
		b.Run([]string{Bar: "bar", Line: "line", Area: "stacked-area"}[kind]+"/pixels-900x440", func(b *testing.B) {
			for b.Loop() {
				c.Draw(dst, image.Point{10, 20}, palette)
			}
		})
		b.Run([]string{Bar: "bar", Line: "line", Area: "stacked-area"}[kind]+"/cells-90x22", func(b *testing.B) {
			for b.Loop() {
				c.cells(c.model(), 90, 22)
			}
		})
	}
}

func palette(t theme.Token) color.RGBA {
	switch t {
	case theme.Chart1:
		return color.RGBA{R: 255, A: 255}
	case theme.Chart2:
		return color.RGBA{B: 255, A: 255}
	case theme.Chart3:
		return color.RGBA{G: 255, A: 255}
	}
	return color.RGBA{}
}
