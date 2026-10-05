package scene

import (
	"image"
	stdcolor "image/color"
	"testing"

	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/layout"
	"github.com/pehcastro/twind/twi/raster"
	"github.com/pehcastro/twind/twi/style"
)

func track(radius style.Radius, children ...Node) Node {
	t := box(2, 1, 20, 1, paint(39, 39, 42, 255))
	t.HidesOverflow, t.Border.Radius = true, radius
	for _, c := range children {
		c.Clip = t.Padding
		t.Children = append(t.Children, c)
	}
	return t
}

func masks(f *Frame) (n int) {
	for _, op := range f.Layers[0].Ops {
		if op.Kind == raster.Clip && op.Box.Radii != [4]float64{} {
			n++
		}
	}
	return n
}

func TestRoundClipMasksChildren(t *testing.T) {
	green := paint(0, 212, 146, 255)
	f := record(page(track(style.RadiusFull, box(2, 1, 12, 1, green))))
	img := image.NewRGBA(image.Rect(0, 0, 400, 240))
	new(raster.Raster).Draw(img, f.Layers[0].Ops, img.Rect)
	if got := img.RGBAAt(20, 20); got != (stdcolor.RGBA{R: 9, G: 9, B: 11, A: 255}) {
		t.Errorf("the track's corner pixel (20,20) is %v, want the page, the unrounded indicator masked out", got)
	}
	if got := img.RGBAAt(60, 30); got.G != 212 {
		t.Errorf("the indicator's middle (60,30) is %v, want green", got)
	}
	ops := f.Layers[0].Ops
	first := -1
	for i, op := range ops {
		if op.Kind == raster.Clip && op.Box.Radii != [4]float64{} && first < 0 {
			first = i
		}
	}
	if first < 2 || ops[1].Kind != raster.Fill || ops[1].Box.Radii == [4]float64{} {
		t.Errorf("want the track's own rounded fill unmasked, then the mask: %+v", ops)
	}

	if n := masks(record(page(track(style.RadiusFull, box(8, 1, 4, 1, green))))); n != 0 {
		t.Errorf("a child clear of the corners got %d masks, want none", n)
	}
	escaped := box(2, 1, 4, 1, green)
	escaped.Position = layout.PositionAbsolute
	parent := track(style.RadiusFull)
	parent.Children = []Node{escaped}
	if n := masks(record(page(parent))); n != 0 {
		t.Errorf("an absolute child clipped by the page, not the track, got %d masks", n)
	}
	bordered := track(style.RadiusLg, box(2, 1, 12, 1, green))
	bordered.Bounds, bordered.Border.Top, bordered.Border.Color = layout.Rect{X: 1, Y: 0, W: 22, H: 3}, true, color.Color{}
	if n := masks(record(page(bordered))); n != 0 {
		t.Errorf("a padding box inset past the 10 px radius got %d masks, want none", n)
	}
}

func TestDashedBorderReachesRaster(t *testing.T) {
	c := box(2, 1, 10, 4, paint(24, 24, 27, 255))
	c.Border = Border{Style: style.BorderDashed, Color: paint(255, 255, 255, 255), Top: true, Right: true, Bottom: true, Left: true}
	dashes := func() (got []raster.Dash) {
		for _, op := range record(page(c)).Layers[0].Ops {
			if op.Color.R == 255 {
				got = append(got, op.Dash)
			}
		}
		return got
	}
	if got := dashes(); len(got) != 1 || got[0] != raster.Dashed {
		t.Errorf("border-dashed ring: %v, want one dashed border op", got)
	}
	c.Border.Style, c.Border.Top, c.Border.Right, c.Border.Left = style.BorderDotted, false, false, false
	if got := dashes(); len(got) != 1 || got[0] != raster.Dotted {
		t.Errorf("border-b border-dotted: %v, want one dotted side", got)
	}
}
