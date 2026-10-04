package scene

import (
	"image"
	"testing"

	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/style"
)

func TestShrinkScalesThePixelsAboutTheCentre(t *testing.T) {
	b := box(2, 1, 6, 2, paint(24, 24, 27, 255))
	b.Border = Border{Style: style.BorderSingle, Radius: style.RadiusLg, Top: true, Right: true, Bottom: true, Left: true, Color: paint(63, 63, 70, 255)}
	full := record(page(b)).Layers[0].Boxes[1].Ops
	b.Shrink = 0.05
	shrunk := record(page(b)).Layers[0].Boxes[1].Ops
	if len(full) != len(shrunk) {
		t.Fatalf("%d ops shrunk, %d at full size", len(shrunk), len(full))
	}
	for i, op := range shrunk {
		if want := (raster.Rect{X: 21.5, Y: 21, W: 57, H: 38}); op.Box.Rect != want {
			t.Errorf("op %d of a 60x40 px box at 20,20 shrunk by 5%% covers %+v, want %+v", op.Kind, op.Box.Rect, want)
		}
		if op.Box.Radii[0] != full[i].Box.Radii[0]*0.95 {
			t.Errorf("op %d radius %v, want 95%% of %v", op.Kind, op.Box.Radii[0], full[i].Box.Radii[0])
		}
	}
}

func TestShrinkDamagesTheBox(t *testing.T) {
	still, shrunk := box(2, 1, 6, 2, paint(24, 24, 27, 255)), box(2, 1, 6, 2, paint(24, 24, 27, 255))
	shrunk.Shrink = 0.05
	var got image.Rectangle
	for _, r := range damage(page(still), page(shrunk)).Rects {
		got = got.Union(r)
	}
	if want := image.Rect(20, 20, 80, 60); !want.In(got) {
		t.Errorf("shrinking a box damages %v, want its %v", got, want)
	}
}
