package raster

import (
	"image"
	"math"
	"testing"
)

func spun(turn float64, ops ...Op) []Op {
	for i := range ops {
		ops[i].Turn = turn
	}
	return ops
}

func turnable(turn float64) map[string][]Op {
	box := Box{Rect: Rect{X: 20, Y: 35, W: 60, H: 30}, Radii: [4]float64{12, 4, 0, 8}}
	return map[string][]Op{
		"fill and border": spun(turn,
			Op{Kind: Fill, Box: box, Color: hex(0x18181b, 255)},
			Op{Kind: Border, Box: box, Width: 2, Color: hex(0x3b82f6, 255)}),
		"gradient and dashed border": spun(turn,
			Op{Kind: Fill, Box: box, Angle: 90, Stops: []Stop{{Color: hex(0x4f46e5, 255), At: 0}, {Color: hex(0xdb2777, 255), At: 1}}},
			Op{Kind: Border, Box: box, Width: 2, Color: black, Dash: Dashed}),
		"partial border about the box centre": spun(turn,
			Op{Kind: Fill, Box: box, Color: hex(0xe4e4e7, 255)},
			Op{Kind: Fill, Box: Box{Rect: Rect{X: 20, Y: 35, W: 60, H: 2}}, Color: black, Pivot: Point{Y: 14}}),
		"shadows": spun(turn,
			Op{Kind: Shadow, Box: box, Color: hex(0, 90), Shadow: BoxShadow{X: 3, Y: 8, Blur: 10, Spread: -2}},
			Op{Kind: Shadow, Box: box, Color: hex(0, 255), Shadow: BoxShadow{Spread: 3}},
			Op{Kind: Fill, Box: box, Color: white},
			Op{Kind: Shadow, Box: box.Inset(2), Color: hex(0, 60), Shadow: BoxShadow{Y: 4, Blur: 6, Inset: true}}),
	}
}

func TestTurnQuarterIsTheBoxRotated(t *testing.T) {
	for name, upright := range turnable(0) {
		turned := turnable(0.25)[name]
		a, b := render(100, 100, upright...), render(100, 100, turned...)
		bad := 0
		for i := range 100 {
			for j := range 100 {
				if got, want := at(b, i, j), at(a, j, 99-i); !near(got, want, 1) {
					if bad++; bad <= 3 {
						t.Errorf("%s at a quarter turn, pixel (%d,%d): %v, the upright box rotated %v", name, i, j, got, want)
					}
				}
			}
		}
		if bad > 3 {
			t.Errorf("%s: %d pixels differ from the upright box rotated", name, bad)
		}
	}
}

func TestTurnEighthMatchesAReference(t *testing.T) {
	box := Box{Rect: Rect{X: 20, Y: 35, W: 60, H: 30}, Radii: [4]float64{12, 12, 12, 12}}
	img := render(100, 100, Op{Kind: Fill, Box: box, Color: black, Turn: 0.125})
	const samples = 16
	cx, cy, c := 50.0, 50.0, math.Sqrt2/2
	inside := func(x, y float64) bool {
		u, v := cx+(x-cx)*c+(y-cy)*c, cy-(x-cx)*c+(y-cy)*c
		qx, qy := math.Abs(u-cx)-30+12, math.Abs(v-cy)-15+12
		return math.Hypot(max(qx, 0), max(qy, 0))+min(max(qx, qy), 0) <= 12
	}
	var worst, sum, ink, want float64
	edge, wrong := 0, 0
	for y := range 100 {
		for x := range 100 {
			hits := 0
			for k := range samples * samples {
				if inside(float64(x)+(float64(k%samples)+0.5)/samples, float64(y)+(float64(k/samples)+0.5)/samples) {
					hits++
				}
			}
			ref, got := float64(hits)/(samples*samples), darkness(img, x, y)
			ink, want = ink+got, want+ref
			if ref > 0 && ref < 1 {
				worst, sum, edge = max(worst, math.Abs(got-ref)), sum+math.Abs(got-ref), edge+1
			}
			if (ref == 0 || ref == 1) && math.Abs(got-ref) > 0.02 {
				if wrong++; wrong <= 3 {
					t.Errorf("pixel (%d,%d) at an eighth turn: %.3f, reference %.0f", x, y, got, ref)
				}
			}
		}
	}
	if wrong > 3 {
		t.Errorf("%d pixels wholly inside or outside the reference are not", wrong)
	}
	if worst > 0.1 || sum/float64(edge) > 0.03 || math.Abs(ink-want) > 0.005*want {
		t.Errorf("an eighth turn against a 16x16 supersampled reference: worst edge error %.3f, mean %.4f over %d edge pixels, ink %.1f want %.1f", worst, sum/float64(edge), edge, ink, want)
	}
}

func BenchmarkTurnCard(b *testing.B) {
	bounds := image.Rect(0, 0, 480, 230)
	img := image.NewRGBA(bounds)
	ops := append([]Op{{Kind: Fill, Box: radius(0, 0, 480, 230, 0), Color: hex(0xfafafa, 255)}},
		spun(0.02, card(radius(20, 20, 440, 168, 12), white, hex(0xe4e4e7, 255), tailwind("md"))...)...)
	var r Raster
	r.Draw(img, ops, bounds)
	b.ReportAllocs()
	for b.Loop() {
		r.Draw(img, ops, bounds)
	}
}

func TestTurnTilesMatchWhole(t *testing.T) {
	bounds := image.Rect(0, 0, 100, 100)
	var ops []Op
	for _, name := range []string{"shadows", "fill and border", "gradient and dashed border"} {
		ops = append(ops, turnable(0.1)[name]...)
	}
	ops = append([]Op{{Kind: Fill, Box: radius(0, 0, 100, 100, 0), Color: white}}, ops...)
	var r Raster
	whole := image.NewRGBA(bounds)
	r.Draw(whole, ops, bounds)
	for ty := 0; ty < 100; ty += 16 {
		for tx := 0; tx < 100; tx += 16 {
			tile := image.Rect(tx, ty, tx+16, ty+16)
			part := image.NewRGBA(bounds)
			r.Draw(part, ops, tile)
			for y := tile.Min.Y; y < min(tile.Max.Y, 100); y++ {
				for x := tile.Min.X; x < min(tile.Max.X, 100); x++ {
					if got, want := at(part, x, y), at(whole, x, y); got != want {
						t.Fatalf("tile %v at (%d,%d): %v, whole raster %v", tile, x, y, got, want)
					}
				}
			}
		}
	}
}
