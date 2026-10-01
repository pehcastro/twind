package present

import (
	"bytes"
	"hash/maphash"
	"image"
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/terminal"
)

func TestTurnAnglesKeepTheirOwnRows(t *testing.T) {
	box := raster.Box{Rect: raster.Rect{X: 10, Y: 15, W: 30, H: 20}, Radii: [4]float64{4, 4, 4, 4}}
	turned := func(turn float64) []raster.Op {
		return []raster.Op{
			{Kind: raster.Shadow, Box: box, Color: color.RGBA{A: 90}, Shadow: raster.BoxShadow{Y: 3, Blur: 6}, Turn: turn},
			{Kind: raster.Fill, Box: box, Color: color.RGBA{R: 20, G: 120, B: 200, A: 255}, Turn: turn},
			{Kind: raster.Border, Box: box, Color: color.RGBA{R: 200, G: 200, B: 200, A: 255}, Width: 1, Turn: turn},
		}
	}
	if _, ok := shareKey(nil, turned(0.1)[1:2], image.Rect(0, 0, 50, 50)); ok {
		t.Error("a turned box offers its pixels to a job that matches only its upright geometry")
	}
	s, _ := screen(terminal.GraphicsSixel)
	s.cache, s.shapes, s.seed = map[uint64]*cached{}, map[string]*cached{}, maphash.MakeSeed()
	angles := []float64{0.1, 0.2}
	looks := make([]*cached, len(angles))
	for i, turn := range angles {
		looks[i] = s.look(&scene.Box{Visual: image.Rect(0, 0, 50, 50), Ops: turned(turn), Look: uint64(i + 1)})
	}
	if looks[0] == looks[1] {
		t.Fatal("a box at turn 0.1 and at turn 0.2 share one raster")
	}
	s.rasterise(0, nil, nil)
	for i, turn := range angles {
		want := image.NewRGBA(image.Rect(0, 0, 50, 50))
		new(raster.Raster).Draw(want, turned(turn), want.Rect)
		got := image.NewRGBA(want.Rect)
		paintLook(got, got.Rect, looks[i], image.Point{})
		for y := range 50 {
			if !bytes.Equal(got.Pix[y*got.Stride:][:got.Stride], want.Pix[y*want.Stride:][:want.Stride]) {
				t.Fatalf("turn %v: row %d differs from a full raster", turn, y)
			}
		}
	}
}
