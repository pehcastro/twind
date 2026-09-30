package scene

import (
	"image"
	"testing"
)

func TestTurnCoversTheTurnedCorners(t *testing.T) {
	for _, c := range []struct {
		turn float64
		want image.Rectangle
	}{
		{0, image.Rect(20, 20, 80, 60)},
		{0.125, image.Rect(14, 4, 86, 76)},
		{0.25, image.Rect(30, 10, 70, 70)},
		{0.5, image.Rect(20, 20, 80, 60)},
	} {
		b := box(2, 1, 6, 2, paint(24, 24, 27, 255))
		b.Turn = c.turn
		if got := record(page(b)).Layers[0].Boxes[1].Visual; !c.want.In(got) || !got.In(c.want.Inset(-1)) {
			t.Errorf("a 60x40 px box at turn %v covers %v, want %v give or take a pixel", c.turn, got, c.want)
		}
	}
}
