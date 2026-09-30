package raster

import (
	"image"
	"testing"
)

func TestClipEdge(t *testing.T) {
	ink := hex(0x1478c8, 255)
	for name, op := range map[string]Op{
		"fill":   {Kind: Fill, Box: radius(3, 0, 40, 50, 0), Color: ink},
		"border": {Kind: Border, Box: radius(3, 0, 40, 50, 0), Width: 2, Color: ink},
	} {
		img := image.NewRGBA(image.Rect(0, 0, 50, 50))
		new(Raster).Draw(img, []Op{{Kind: Clip, Box: radius(0, 17, 50, 21, 0)}, op, {Kind: Pop}}, img.Rect)
		for y := range 50 {
			for x := range 50 {
				want := ink
				inner := op.Kind == Border && x >= 5 && x < 41
				if y < 17 || y > 37 || x < 3 || x >= 43 || inner {
					want = hex(0, 0)
				}
				if got := at(img, x, y); got != want {
					t.Fatalf("%s at (%d,%d): %v, want %v", name, x, y, got, want)
				}
			}
		}
	}
}

func TestGradientRamp(t *testing.T) {
	stops := []Stop{{Color: hex(0x0ea5e9, 255)}, {Color: hex(0xf43f5e, 255), At: 1}}
	op := Op{Kind: Fill, Box: radius(0, 0, 64, 4, 0), Angle: 90, Stops: stops}
	var shared Raster
	reused := image.NewRGBA(image.Rect(0, 0, 64, 4))
	shared.Draw(reused, []Op{op}, reused.Rect)
	stops[1].Color = hex(0x22c55e, 255)
	shared.Draw(reused, []Op{op}, reused.Rect)
	fresh := image.NewRGBA(reused.Rect)
	new(Raster).Draw(fresh, []Op{op}, fresh.Rect)
	for x := range 64 {
		if got, want := at(reused, x, 2), at(fresh, x, 2); got != want {
			t.Fatalf("column %d: %v from a reused raster, %v from a fresh one", x, got, want)
		}
	}
}

func BenchmarkGradientPill(b *testing.B) {
	pill := surfacesPage[3]
	img := image.NewRGBA(image.Rectangle{Max: pill.size})
	var r Raster
	r.Draw(img, pill.ops, img.Rect)
	b.ReportAllocs()
	for b.Loop() {
		r.Draw(img, pill.ops, img.Rect)
	}
}
