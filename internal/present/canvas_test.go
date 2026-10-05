package present

import (
	"bytes"
	"hash/maphash"
	"image"
	"testing"

	"github.com/pehcastro/twind/internal/raster"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi/color"
)

func TestCanvasKeysKeepTheirOwnRows(t *testing.T) {
	painted := func(key uint64) []raster.Op {
		img := image.NewRGBA(image.Rect(0, 0, 30, 20))
		for y := range 20 {
			for x := range 30 {
				if (x+y+int(key))%7 == 0 {
					copy(img.Pix[img.PixOffset(x, y):], []uint8{0, 200, 100, 255})
				}
			}
		}
		return []raster.Op{
			{Kind: raster.Fill, Box: raster.Box{Rect: raster.Rect{W: 50, H: 50}}, Color: color.RGBA{R: 20, G: 20, B: 24, A: 255}},
			{Kind: raster.Canvas, Box: raster.Box{Rect: raster.Rect{X: 10, Y: 15, W: 30, H: 20}}, Pixels: &raster.Pixels{Key: key, Image: img}},
		}
	}
	if _, ok := shareKey(nil, painted(1)[1:], image.Rect(0, 0, 50, 50)); ok {
		t.Error("a canvas offers its pixels to a job that matches only its box")
	}
	s, _ := screen(terminal.GraphicsSixel)
	s.cache, s.shapes, s.seed = map[uint64]*cached{}, map[string]*cached{}, maphash.MakeSeed()
	keys := []uint64{1, 2}
	ops := make([][]raster.Op, len(keys))
	looks := make([]*cached, len(keys))
	for i, key := range keys {
		ops[i] = painted(key)
		looks[i] = s.look(&scene.Box{Visual: image.Rect(0, 0, 50, 50), Ops: ops[i], Look: uint64(i + 1)})
	}
	if looks[0] == looks[1] {
		t.Fatal("two canvases with different keys share one raster")
	}
	s.rasterise(0, nil, nil)
	for i, key := range keys {
		want := image.NewRGBA(image.Rect(0, 0, 50, 50))
		new(raster.Raster).Draw(want, ops[i], want.Rect)
		got := image.NewRGBA(want.Rect)
		paintLook(got, got.Rect, looks[i], image.Point{})
		for y := range 50 {
			if !bytes.Equal(got.Pix[y*got.Stride:][:got.Stride], want.Pix[y*want.Stride:][:want.Stride]) {
				t.Fatalf("key %d: row %d differs from a full raster", key, y)
			}
		}
	}
}
