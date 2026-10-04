package scene

import (
	"image"
	"testing"

	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/raster"
)

func canvas(key uint64, img *image.RGBA) Node {
	b := box(2, 1, 8, 4, paint(24, 24, 27, 255))
	b.Content = layout.Rect{X: 3, Y: 2, W: 6, H: 2}
	b.Pixels = &raster.Pixels{Key: key, Image: img}
	return b
}

func TestCanvasRecordsItsPixelsOverTheContentBox(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 60, 40))
	bare := canvas(1, img)
	bare.Background = paint(0, 0, 0, 0)
	ops := record(page(bare)).Layers[0].Boxes[1].Ops
	if len(ops) != 1 || ops[0].Kind != raster.Canvas {
		t.Fatalf("a canvas with no background gave %+v, want one canvas op", ops)
	}
	if got, want := ops[0].Box.Rect, (raster.Rect{X: 30, Y: 40, W: 60, H: 40}); got != want || ops[0].Pixels.Image != img {
		t.Errorf("canvas op over %+v with image %p, want the content box %+v and the node's image", got, ops[0].Pixels.Image, want)
	}
	if d := damage(page(canvas(1, img)), page(canvas(2, image.NewRGBA(img.Rect)))); len(d.Rects) == 0 {
		t.Error("a new key gave no damage")
	}
	if d := damage(page(canvas(1, img)), page(canvas(1, img))); len(d.Rects) != 0 {
		t.Errorf("the same canvas twice gave damage %v", d.Rects)
	}
}
