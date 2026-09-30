package present

import (
	"bytes"
	"image"
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/terminal"
)

func TestPlannedRowsMatchAFullRaster(t *testing.T) {
	grey, ink := color.RGBA{R: 90, G: 90, B: 90, A: 200}, color.RGBA{R: 20, G: 120, B: 200, A: 255}
	box := func(x, y, w, h float64, radii ...float64) raster.Box {
		b := raster.Box{Rect: raster.Rect{X: x, Y: y, W: w, H: h}}
		copy(b.Radii[:], radii)
		return b
	}
	cases := map[string][]raster.Op{
		"rounded fill":      {{Kind: raster.Fill, Box: box(2, 3, 40, 40, 9, 9, 4, 4), Color: grey}},
		"fractional edges":  {{Kind: raster.Fill, Box: box(1.5, 2.25, 30, 30.5), Color: grey}},
		"rounded border":    {{Kind: raster.Fill, Box: box(0, 0, 50, 48, 6, 6, 6, 6), Color: ink}, {Kind: raster.Border, Box: box(0, 0, 50, 48, 6, 6, 6, 6), Color: grey, Width: 2.5}},
		"stacked fills":     {{Kind: raster.Fill, Box: box(0, 0, 50, 50), Color: ink}, {Kind: raster.Fill, Box: box(4, 10.5, 30, 5.25, 2, 2, 2, 2), Color: grey}},
		"wide radius":       {{Kind: raster.Fill, Box: box(0, 0, 50, 12, 30, 30, 30, 30), Color: grey}},
		"shadow":            {{Kind: raster.Shadow, Box: box(8, 8, 30, 30, 5, 5, 5, 5), Color: grey, Shadow: raster.BoxShadow{Y: 3, Blur: 6}}, {Kind: raster.Fill, Box: box(8, 8, 30, 30, 5, 5, 5, 5), Color: ink}},
		"vertical gradient": {{Kind: raster.Fill, Box: box(0, 0, 50, 50), Color: ink, Stops: []raster.Stop{{Color: ink}, {Color: grey, At: 1}}, Angle: 180}},
		"dashed border":     {{Kind: raster.Border, Box: box(1, 1, 48, 48), Color: grey, Width: 1, Dash: raster.Dashed}},
		"offset shadow":     {{Kind: raster.Shadow, Box: box(8, 4, 30, 30, 5, 5, 5, 5), Color: grey, Shadow: raster.BoxShadow{Y: 9, Blur: 10, Spread: 3}}, {Kind: raster.Fill, Box: box(8, 4, 30, 30, 5, 5, 5, 5), Color: ink}},
		"shrunk shadow":     {{Kind: raster.Shadow, Box: box(8, 8, 30, 30, 5, 5, 5, 5), Color: grey, Shadow: raster.BoxShadow{Y: 5, Blur: 7.5, Spread: -4}}},
		"inset shadow":      {{Kind: raster.Fill, Box: box(4, 4, 40, 40, 6, 6, 6, 6), Color: ink}, {Kind: raster.Shadow, Box: box(4, 4, 40, 40, 6, 6, 6, 6), Color: grey, Shadow: raster.BoxShadow{Y: 2, Blur: 6, Inset: true}}},
		"clip across": {
			{Kind: raster.Clip, Box: box(0, 17, 50, 21)},
			{Kind: raster.Fill, Box: box(3, 0, 40, 50), Color: ink},
			{Kind: raster.Pop},
			{Kind: raster.Fill, Box: box(2, 41, 40, 3), Color: grey},
		},
		"rounded clip": {
			{Kind: raster.Clip, Box: box(6, 12, 36, 28, 9, 9, 9, 9)},
			{Kind: raster.Fill, Box: box(0, 0, 50, 50), Color: ink},
			{Kind: raster.Pop},
		},
	}
	for name, ops := range cases {
		want := image.NewRGBA(image.Rect(0, 0, 50, 50))
		var r raster.Raster
		r.Draw(want, ops, want.Rect)
		shifted := append([]raster.Op(nil), ops...)
		for i := range shifted {
			shifted[i].Box.X, shifted[i].Box.Y = shifted[i].Box.X+10, shifted[i].Box.Y+20
		}
		s, _ := screen(terminal.GraphicsSixel)
		s.cache = map[uint64]*cached{}
		c := s.look(&scene.Box{Visual: image.Rect(10, 20, 60, 70), Ops: shifted})
		s.rasterise()
		got := image.NewRGBA(want.Rect)
		over(got, got.Rect, c, image.Point{})
		for y := range 50 {
			if !bytes.Equal(got.Pix[y*got.Stride:][:got.Stride], want.Pix[y*want.Stride:][:want.Stride]) {
				t.Errorf("%s: row %d differs from a full raster", name, y)
				break
			}
		}
	}
}
