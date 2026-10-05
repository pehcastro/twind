package raster

import (
	"image"
	"testing"

	"github.com/pehcastro/twind/twi/color"
)

var surfacesPage = []struct {
	size image.Point
	ops  []Op
}{
	{image.Pt(1200, 800), []Op{
		{Kind: Fill, Box: Box{Rect: Rect{X: 0, Y: 0, W: 1200, H: 800}}, Color: color.RGBA{R: 0x9, G: 0x9, B: 0xb, A: 0xff}},
	}},
	{image.Pt(1160, 60), []Op{
		{Kind: Fill, Box: Box{Rect: Rect{X: 0, Y: 0, W: 1160, H: 60}, Radii: [4]float64{10, 10, 10, 10}}, Color: color.RGBA{R: 0x18, G: 0x18, B: 0x1b, A: 0xff}},
		{Kind: Border, Box: Box{Rect: Rect{X: 0, Y: 0, W: 1160, H: 60}, Radii: [4]float64{10, 10, 10, 10}}, Color: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x1a}, Width: 1},
	}},
	{image.Pt(1020, 20), []Op{
		{Kind: Clip, Box: Box{Rect: Rect{X: -100, Y: -480, W: 1200, H: 680}}},
		{Kind: Fill, Box: Box{Rect: Rect{X: 0, Y: 0, W: 1020, H: 20}, Radii: [4]float64{1048576, 1048576, 1048576, 1048576}}, Color: color.RGBA{R: 0x27, G: 0x27, B: 0x2a, A: 0xff}},
		{Kind: Pop},
	}},
	{image.Pt(680, 20), []Op{
		{Kind: Clip, Box: Box{Rect: Rect{X: -100, Y: -480, W: 1200, H: 680}}},
		{Kind: Fill, Box: Box{Rect: Rect{X: 0, Y: 0, W: 680, H: 20}, Radii: [4]float64{1048576, 1048576, 1048576, 1048576}}, Stops: []Stop{{Color: color.RGBA{R: 0x0, G: 0xd4, B: 0x92, A: 0xff}, At: 0}, {Color: color.RGBA{R: 0x0, G: 0xa6, B: 0xf4, A: 0xff}, At: 1}}, Angle: 90},
		{Kind: Pop},
	}},
	{image.Pt(350, 149), []Op{
		{Kind: Clip, Box: Box{Rect: Rect{X: -785, Y: -348, W: 1200, H: 680}}},
		{Kind: Shadow, Box: Box{Rect: Rect{X: 25, Y: 12, W: 300, H: 100}, Radii: [4]float64{7.5, 7.5, 7.5, 7.5}}, Color: color.RGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x1a}, Shadow: BoxShadow{X: 0, Y: 5, Blur: 7.5, Spread: -5}},
		{Kind: Shadow, Box: Box{Rect: Rect{X: 25, Y: 12, W: 300, H: 100}, Radii: [4]float64{7.5, 7.5, 7.5, 7.5}}, Color: color.RGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x1a}, Shadow: BoxShadow{X: 0, Y: 12.5, Blur: 18.75, Spread: -3.75}},
		{Kind: Fill, Box: Box{Rect: Rect{X: 25, Y: 12, W: 300, H: 100}, Radii: [4]float64{7.5, 7.5, 7.5, 7.5}}, Color: color.RGBA{R: 0x18, G: 0x18, B: 0x1b, A: 0xff}},
		{Kind: Border, Box: Box{Rect: Rect{X: 25, Y: 12, W: 300, H: 100}, Radii: [4]float64{7.5, 7.5, 7.5, 7.5}}, Color: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x1a}, Width: 1},
		{Kind: Pop},
	}},
	{image.Pt(110, 20), []Op{
		{Kind: Clip, Box: Box{Rect: Rect{X: -30, Y: -440, W: 1200, H: 680}}},
		{Kind: Fill, Box: Box{Rect: Rect{X: 0, Y: 0, W: 110, H: 20}, Radii: [4]float64{1048576, 1048576, 1048576, 1048576}}, Color: color.RGBA{R: 0x27, G: 0x27, B: 0x2a, A: 0xff}},
		{Kind: Pop},
	}},
	{image.Pt(390, 120), []Op{
		{Kind: Clip, Box: Box{Rect: Rect{X: -20, Y: -295, W: 1200, H: 680}}},
		{Kind: Shadow, Box: Box{Rect: Rect{X: 10, Y: 5, W: 370, H: 100}, Radii: [4]float64{10, 10, 10, 10}}, Color: color.RGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x1a}, Shadow: BoxShadow{X: 0, Y: 2.5, Blur: 5, Spread: -2.5}},
		{Kind: Shadow, Box: Box{Rect: Rect{X: 10, Y: 5, W: 370, H: 100}, Radii: [4]float64{10, 10, 10, 10}}, Color: color.RGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x1a}, Shadow: BoxShadow{X: 0, Y: 5, Blur: 7.5, Spread: -1.25}},
		{Kind: Fill, Box: Box{Rect: Rect{X: 10, Y: 5, W: 370, H: 100}, Radii: [4]float64{10, 10, 10, 10}}, Color: color.RGBA{R: 0x18, G: 0x18, B: 0x1b, A: 0xff}},
		{Kind: Border, Box: Box{Rect: Rect{X: 10, Y: 5, W: 370, H: 100}, Radii: [4]float64{10, 10, 10, 10}}, Color: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x1a}, Width: 1},
		{Kind: Pop},
	}},
	{image.Pt(80, 20), []Op{
		{Kind: Clip, Box: Box{Rect: Rect{X: -160, Y: -440, W: 1200, H: 680}}},
		{Kind: Fill, Box: Box{Rect: Rect{X: 0, Y: 0, W: 80, H: 20}, Radii: [4]float64{1048576, 1048576, 1048576, 1048576}}, Color: color.RGBA{R: 0x27, G: 0x27, B: 0x2a, A: 0xff}},
		{Kind: Pop},
	}},
	{image.Pt(130, 20), []Op{
		{Kind: Clip, Box: Box{Rect: Rect{X: -260, Y: -440, W: 1200, H: 680}}},
		{Kind: Fill, Box: Box{Rect: Rect{X: 0, Y: 0, W: 130, H: 20}, Radii: [4]float64{1048576, 1048576, 1048576, 1048576}}, Color: color.RGBA{R: 0xff, G: 0x64, B: 0x67, A: 0xff}},
		{Kind: Pop},
	}},
	{image.Pt(90, 20), []Op{
		{Kind: Clip, Box: Box{Rect: Rect{X: -410, Y: -440, W: 1200, H: 680}}},
		{Kind: Fill, Box: Box{Rect: Rect{X: 0, Y: 0, W: 90, H: 20}, Radii: [4]float64{1048576, 1048576, 1048576, 1048576}}, Color: color.RGBA{R: 0x0, G: 0xd4, B: 0x92, A: 0xff}},
		{Kind: Pop},
	}},
	{image.Pt(390, 120), []Op{
		{Kind: Clip, Box: Box{Rect: Rect{X: -410, Y: -295, W: 1200, H: 680}}},
		{Kind: Shadow, Box: Box{Rect: Rect{X: 10, Y: 5, W: 370, H: 100}, Radii: [4]float64{10, 10, 10, 10}}, Color: color.RGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x1a}, Shadow: BoxShadow{X: 0, Y: 2.5, Blur: 5, Spread: -2.5}},
		{Kind: Shadow, Box: Box{Rect: Rect{X: 10, Y: 5, W: 370, H: 100}, Radii: [4]float64{10, 10, 10, 10}}, Color: color.RGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x1a}, Shadow: BoxShadow{X: 0, Y: 5, Blur: 7.5, Spread: -1.25}},
		{Kind: Fill, Box: Box{Rect: Rect{X: 10, Y: 5, W: 370, H: 100}, Radii: [4]float64{10, 10, 10, 10}}, Color: color.RGBA{R: 0x18, G: 0x18, B: 0x1b, A: 0xff}},
		{Kind: Border, Box: Box{Rect: Rect{X: 10, Y: 5, W: 370, H: 100}, Radii: [4]float64{10, 10, 10, 10}}, Color: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x1a}, Width: 1},
		{Kind: Pop},
	}},
	{image.Pt(380, 120), []Op{
		{Kind: Clip, Box: Box{Rect: Rect{X: -800, Y: -295, W: 1200, H: 680}}},
		{Kind: Shadow, Box: Box{Rect: Rect{X: 10, Y: 5, W: 360, H: 100}, Radii: [4]float64{10, 10, 10, 10}}, Color: color.RGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x1a}, Shadow: BoxShadow{X: 0, Y: 2.5, Blur: 5, Spread: -2.5}},
		{Kind: Shadow, Box: Box{Rect: Rect{X: 10, Y: 5, W: 360, H: 100}, Radii: [4]float64{10, 10, 10, 10}}, Color: color.RGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x1a}, Shadow: BoxShadow{X: 0, Y: 5, Blur: 7.5, Spread: -1.25}},
		{Kind: Fill, Box: Box{Rect: Rect{X: 10, Y: 5, W: 360, H: 100}, Radii: [4]float64{10, 10, 10, 10}}, Color: color.RGBA{R: 0x18, G: 0x18, B: 0x1b, A: 0xff}},
		{Kind: Border, Box: Box{Rect: Rect{X: 10, Y: 5, W: 360, H: 100}, Radii: [4]float64{10, 10, 10, 10}}, Color: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x1a}, Width: 1},
		{Kind: Pop},
	}},
	{image.Pt(150, 20), []Op{
		{Kind: Clip, Box: Box{Rect: Rect{X: -990, Y: -280, W: 1200, H: 680}}},
		{Kind: Fill, Box: Box{Rect: Rect{X: 0, Y: 0, W: 150, H: 20}, Radii: [4]float64{1048576, 1048576, 1048576, 1048576}}, Color: color.RGBA{R: 0xe4, G: 0xe4, B: 0xe7, A: 0xff}},
		{Kind: Pop},
	}},
	{image.Pt(70, 20), []Op{
		{Kind: Fill, Box: Box{Rect: Rect{X: 0, Y: 0, W: 70, H: 20}, Radii: [4]float64{1048576, 1048576, 1048576, 1048576}}, Color: color.RGBA{R: 0x0, G: 0xd4, B: 0x92, A: 0xff}},
	}},
	{image.Pt(100, 20), []Op{
		{Kind: Fill, Box: Box{Rect: Rect{X: 0, Y: 0, W: 100, H: 20}, Radii: [4]float64{1048576, 1048576, 1048576, 1048576}}, Color: color.RGBA{R: 0xe4, G: 0xe4, B: 0xe7, A: 0xff}},
	}},
}

func BenchmarkSurfacesPage(b *testing.B) {
	images := make([]*image.RGBA, len(surfacesPage))
	for i, box := range surfacesPage {
		images[i] = image.NewRGBA(image.Rectangle{Max: box.size})
	}
	var r Raster
	page := func() {
		for i, box := range surfacesPage {
			r.Draw(images[i], box.ops, images[i].Rect)
		}
	}
	page()
	b.ReportAllocs()
	for b.Loop() {
		page()
	}
}
