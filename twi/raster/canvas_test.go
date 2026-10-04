package raster

import (
	"bytes"
	"image"
	"testing"

	"github.com/twind-dev/twind/twi/color"
)

func TestCanvasCompositesPremultipliedInsideItsClip(t *testing.T) {
	pixels := image.NewRGBA(image.Rect(0, 0, 10, 4))
	copy(pixels.Pix[pixels.PixOffset(0, 0):], []uint8{128, 0, 0, 128})
	copy(pixels.Pix[pixels.PixOffset(6, 0):], []uint8{0, 200, 0, 255})
	copy(pixels.Pix[pixels.PixOffset(9, 3):], []uint8{0, 0, 255, 255})
	ops := []Op{
		{Kind: Fill, Box: Box{Rect: Rect{0, 0, 20, 10}}, Color: color.RGBA{R: 10, G: 20, B: 30, A: 255}},
		{Kind: Clip, Box: Box{Rect: Rect{0, 0, 12, 10}}},
		{Kind: Canvas, Box: Box{Rect: Rect{5, 2, 10, 4}}, Pixels: &Pixels{Key: 1, Image: pixels}},
		{Kind: Pop},
	}
	whole := image.NewRGBA(image.Rect(0, 0, 20, 10))
	new(Raster).Draw(whole, ops, whole.Rect)
	for _, c := range []struct {
		x, y int
		want [4]uint8
		why  string
	}{
		{5, 2, [4]uint8{128 + 5, 10, 15, 255}, "a half red pixel blends premultiplied over the fill"},
		{11, 2, [4]uint8{0, 200, 0, 255}, "an opaque pixel inside the clip replaces the fill"},
		{6, 2, [4]uint8{10, 20, 30, 255}, "a transparent pixel keeps the fill"},
		{11, 3, [4]uint8{10, 20, 30, 255}, "a canvas row is its own, not a copy of the row above"},
		{14, 5, [4]uint8{10, 20, 30, 255}, "a pixel outside the clip is not drawn"},
		{4, 2, [4]uint8{10, 20, 30, 255}, "the canvas starts at its box"},
	} {
		if got := [4]uint8(whole.Pix[whole.PixOffset(c.x, c.y):]); got != c.want {
			t.Errorf("%s: %d,%d is %v, want %v", c.why, c.x, c.y, got, c.want)
		}
	}
	tiled := image.NewRGBA(whole.Rect)
	for _, tile := range []image.Rectangle{image.Rect(0, 0, 8, 3), image.Rect(8, 0, 20, 3), image.Rect(0, 3, 20, 10)} {
		new(Raster).Draw(tiled, ops, tile)
	}
	if !bytes.Equal(tiled.Pix, whole.Pix) {
		t.Error("a canvas drawn in three tiles differs from the canvas drawn whole")
	}
}
