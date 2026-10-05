package raster

import "testing"

func TestCornerButtonGroup(t *testing.T) {
	const r = 7.5
	img := render(200, 40,
		Op{Kind: Fill, Box: Box{Rect: Rect{10, 10, 60, 20}, Radii: [4]float64{r, 0, 0, r}}, Color: black},
		Op{Kind: Fill, Box: Box{Rect: Rect{70, 10, 60, 20}}, Color: black},
		Op{Kind: Fill, Box: Box{Rect: Rect{130, 10, 60, 20}, Radii: [4]float64{0, r, r, 0}}, Color: black},
	)
	for _, p := range []struct {
		x, y int
		want float64
	}{
		{10, 10, 0}, {10, 29, 0}, {189, 10, 0}, {189, 29, 0},
		{69, 10, 1}, {70, 10, 1}, {69, 29, 1}, {129, 29, 1}, {130, 10, 1}, {10, 20, 1}, {189, 20, 1},
	} {
		if got := darkness(img, p.x, p.y); got != p.want {
			t.Errorf("(%d,%d): coverage %.3f, want %v: outer corners round, the seams square", p.x, p.y, got, p.want)
		}
	}
}

func TestCornerFullOnOneSide(t *testing.T) {
	img := render(140, 40, Op{Kind: Fill, Box: Box{Rect: Rect{20, 10, 100, 20}, Radii: [4]float64{1 << 20, 0, 0, 1 << 20}}, Color: black})
	for _, p := range []struct {
		x, y int
		want float64
	}{{20, 10, 0}, {20, 29, 0}, {21, 20, 1}, {32, 10, 1}, {119, 10, 1}, {119, 29, 1}} {
		if got := darkness(img, p.x, p.y); got != p.want {
			t.Errorf("rounded-l-full (%d,%d): coverage %.3f, want %v: a half-height curve on the left, square on the right", p.x, p.y, got, p.want)
		}
	}
}

func TestCornerTopClip(t *testing.T) {
	img := render(140, 60,
		Op{Kind: Clip, Box: Box{Rect: Rect{20, 10, 100, 40}, Radii: [4]float64{10, 10, 0, 0}}},
		Op{Kind: Fill, Box: Box{Rect: Rect{0, 0, 140, 60}}, Color: black},
		Op{Kind: Pop},
	)
	for _, p := range []struct {
		x, y int
		want float64
	}{{20, 10, 0}, {119, 10, 0}, {20, 49, 1}, {119, 49, 1}, {70, 10, 1}, {19, 30, 0}, {70, 50, 0}} {
		if got := darkness(img, p.x, p.y); got != p.want {
			t.Errorf("rounded-t clip (%d,%d): coverage %.3f, want %v", p.x, p.y, got, p.want)
		}
	}
}
