package scene

import (
	"image"
	"testing"

	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/style"
)

func TestHalfRowEdgesMeetOnOnePixel(t *testing.T) {
	for _, size := range []image.Point{image.Pt(10, 20), image.Pt(8, 17), image.Pt(8, 16)} {
		upper, lower := box(2, 1, 10, 2, paint(200, 0, 0, 255)), box(2, 2, 10, 2, paint(0, 200, 0, 255))
		upper.Halves.Bounds, lower.Halves.Bounds = HalfBottom, HalfTop
		f := new(Frame)
		root := page(upper, lower)
		f.Record(&root, size)
		var fills []raster.Box
		for _, op := range f.Layers[0].Ops {
			if op.Kind == raster.Fill && (op.Color == upper.Background.RGBA || op.Color == lower.Background.RGBA) {
				fills = append(fills, op.Box)
			}
		}
		edge := float64(2*size.Y + size.Y/2)
		if len(fills) != 2 || fills[0].Y != float64(size.Y) || fills[0].Y+fills[0].H != edge || fills[1].Y != edge || fills[1].Y+fills[1].H != float64(4*size.Y) {
			t.Errorf("%v cells: fills %+v, want the upper from %d to %v and the lower from %v to %d", size, fills, size.Y, edge, edge, 4*size.Y)
		}
	}
}

func TestHalfRowClipsCutAtTheHalf(t *testing.T) {
	card := box(2, 1, 20, 4, paint(39, 39, 42, 255))
	card.HidesOverflow = true
	card.Border.Radius = style.RadiusLg
	card.Halves = Halves{Bounds: HalfTop | HalfBottom, Padding: HalfTop | HalfBottom}
	child := box(2, 0, 20, 6, paint(0, 212, 146, 255))
	child.Clip, child.Halves.Clip = card.Padding, HalfTop|HalfBottom
	card.Children = []Node{child}
	var clips []raster.Box
	for _, op := range record(page(card)).Layers[0].Ops {
		if op.Kind == raster.Clip {
			clips = append(clips, op.Box)
		}
	}
	top, bottom := float64(cell.Y+cell.Y/2), float64(5*cell.Y-cell.Y/2)
	if len(clips) == 0 {
		t.Fatal("no clip op for a child taller than its rounded card")
	}
	for _, c := range clips {
		if c.Y != top || c.Y+c.H != bottom {
			t.Errorf("clip %+v, want from %v to %v, the card's half-row edges", c, top, bottom)
		}
	}
}
