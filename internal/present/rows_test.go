package present

import (
	"bytes"
	"image"
	"slices"
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/terminal"
)

func TestLooksShareOnlyTheSamePixels(t *testing.T) {
	ink := color.RGBA{R: 20, G: 120, B: 200, A: 255}
	fill := raster.Op{Kind: raster.Fill, Box: raster.Box{Rect: raster.Rect{X: 13, Y: 24, W: 40, H: 30}, Radii: [4]float64{4, 4, 4, 4}}, Color: ink}
	clip := func(x, y, w, h float64) raster.Op {
		return raster.Op{Kind: raster.Clip, Box: raster.Box{Rect: raster.Rect{X: x, Y: y, W: w, H: h}}}
	}
	pop := raster.Op{Kind: raster.Pop}
	s, _ := screen(terminal.GraphicsSixel)
	s.cache, s.shapes = map[uint64]*cached{}, map[string]*cached{}
	look := func(key uint64, x int, ops ...raster.Op) *cached {
		shifted := append([]raster.Op(nil), ops...)
		for i := range shifted {
			shifted[i].Box.X += float64(x)
		}
		return s.look(&scene.Box{Visual: image.Rect(10+x, 20, 60+x, 60), Ops: shifted, Look: key})
	}
	plain := look(1, 0, fill)
	for _, c := range []struct {
		name  string
		got   *cached
		share bool
	}{
		{"moved", look(2, 300, fill), true},
		{"inside a clip around it", look(3, 0, clip(0, 0, 900, 900), fill, pop), true},
		{"inside a clip that cuts it", look(4, 0, clip(0, 0, 30, 900), fill, pop), false},
		{"inside a rounded clip around it", look(5, 0, raster.Op{Kind: raster.Clip, Box: raster.Box{Rect: raster.Rect{W: 900, H: 900}, Radii: [4]float64{2, 2, 2, 2}}}, fill, pop), false},
		{"another colour", look(6, 0, raster.Op{Kind: fill.Kind, Box: fill.Box, Color: color.RGBA{A: 255}}), false},
	} {
		if (c.got == plain) != c.share {
			t.Errorf("a fill %s shares the plain fill's pixels: %v, want %v", c.name, c.got == plain, c.share)
		}
	}
}

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
		"clip across columns": {
			{Kind: raster.Clip, Box: box(17, 0, 21, 50)},
			{Kind: raster.Fill, Box: box(0, 3, 50, 40), Color: ink},
			{Kind: raster.Pop},
			{Kind: raster.Fill, Box: box(41, 2, 3, 40), Color: grey},
		},
		"uneven radii":        {{Kind: raster.Fill, Box: box(2, 3, 44, 40, 2, 12, 1, 3), Color: grey}, {Kind: raster.Border, Box: box(2, 3, 44, 40, 2, 12, 1, 3), Color: ink, Width: 1.5}},
		"sideways shadow":     {{Kind: raster.Shadow, Box: box(2, 8, 16, 30, 3, 3, 3, 3), Color: grey, Shadow: raster.BoxShadow{X: 20, Blur: 4, Spread: 1}}, {Kind: raster.Fill, Box: box(2, 8, 16, 30, 3, 3, 3, 3), Color: ink}},
		"horizontal gradient": {{Kind: raster.Fill, Box: box(0, 0, 50, 50, 4, 4, 4, 4), Color: ink, Stops: []raster.Stop{{Color: ink}, {Color: grey, At: 1}}, Angle: 90}},
		"translucent fill":    {{Kind: raster.Fill, Box: box(0, 0, 50, 50), Color: ink}, {Kind: raster.Fill, Box: box(5.5, 5, 39, 30, 6, 6, 6, 6), Color: grey}},
	}
	s, _ := screen(terminal.GraphicsSixel)
	for name, ops := range cases {
		want := image.NewRGBA(image.Rect(0, 0, 50, 50))
		var r raster.Raster
		r.Draw(want, ops, want.Rect)
		shifted := append([]raster.Op(nil), ops...)
		for i := range shifted {
			shifted[i].Box.X, shifted[i].Box.Y = shifted[i].Box.X+10, shifted[i].Box.Y+20
		}
		s.cache, s.shapes = map[uint64]*cached{}, map[string]*cached{}
		c := s.look(&scene.Box{Visual: image.Rect(10, 20, 60, 70), Ops: shifted})
		s.rasterise(0, nil)
		got := image.NewRGBA(want.Rect)
		paintLook(got, got.Rect, c, image.Point{})
		for y := range 50 {
			if !bytes.Equal(got.Pix[y*got.Stride:][:got.Stride], want.Pix[y*want.Stride:][:want.Stride]) {
				t.Errorf("%s: row %d differs from a full raster", name, y)
				break
			}
		}
		for y := 1; y < 50; y++ {
			for lo := range 50 {
				for _, w := range []int{1, 7} {
					hi := min(lo+w, 50)
					equal := bytes.Equal(got.Pix[got.PixOffset(lo, y):got.PixOffset(hi, y)], got.Pix[got.PixOffset(lo, y-1):got.PixOffset(hi, y-1)])
					if repeats([]part{{step: drawBox, c: c, r: image.Rect(lo, 0, hi, 50)}}, y) && !equal {
						t.Errorf("%s: row %d repeats row %d between x %d and %d, but its pixels differ", name, y, y-1, lo, hi)
					}
				}
			}
		}
		background := make([]uint8, 4*50)
		for i := range background {
			background[i] = uint8(i * 37 % 200)
			if i%4 == 3 {
				background[i] = 255
			}
		}
		for y := range 50 {
			line, under := c.lines[c.row[y]], encode(nil, 0, background, 0)
			whole, windows := compose(nil, under, 0, 50, line, 0, floodBlend, 0), under
			for x := 0; x < 50; x += 7 {
				windows = compose(nil, windows, int32(x), int32(min(x+7, 50)), line, 0, floodBlend, 0)
			}
			if !slices.Equal(whole, windows) {
				t.Errorf("%s: row %d composited in 7 pixel windows differs from one pass over a background", name, y)
				break
			}
		}
	}
}
