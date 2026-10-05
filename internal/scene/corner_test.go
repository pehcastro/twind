package scene

import (
	"image"
	"testing"

	"github.com/pehcastro/twind/internal/raster"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/style"
)

func rounded(size style.Radius, at ...style.Corner) style.Radius {
	r := style.RadiusNone
	for _, c := range at {
		r = r.With(c, size)
	}
	return r
}

func drawn(root Node) (*Frame, *image.RGBA) {
	f := record(root)
	img := image.NewRGBA(image.Rect(0, 0, 400, 240))
	new(raster.Raster).Draw(img, f.Layers[0].Ops, img.Rect)
	return f, img
}

func TestCornerButtonGroup(t *testing.T) {
	edge, fill := paint(63, 63, 70, 255), paint(24, 24, 27, 255)
	md := 0.375 * float64(cell.Y)
	radii := [3]style.Radius{
		rounded(style.RadiusMd, style.CornerTopLeft, style.CornerBottomLeft),
		style.RadiusNone,
		rounded(style.RadiusMd, style.CornerTopRight, style.CornerBottomRight),
	}
	var buttons []Node
	for i, r := range radii {
		b := box(2+6*i, 1, 6, 2, fill)
		b.Border = Border{Style: style.BorderSingle, Radius: r, Top: true, Right: true, Bottom: true, Left: true, Color: edge}
		buttons = append(buttons, b)
	}
	f, img := drawn(page(buttons...))
	want := [3][4]float64{{md, 0, 0, md}, {}, {0, md, md, 0}}
	seen := 0
	for _, op := range f.Layers[0].Ops {
		if op.Kind != raster.Fill && op.Kind != raster.Border || op.Color != edge.RGBA && op.Color != fill.RGBA {
			continue
		}
		i := (int(op.Box.X)/cell.X - 2) / 6
		if op.Box.Radii != want[i] {
			t.Errorf("button %d op %d: radii %v, want %v (top left, top right, bottom right, bottom left)", i, op.Kind, op.Box.Radii, want[i])
		}
		seen++
	}
	if seen != 6 {
		t.Errorf("%d fill and border ops, want a fill and a border per button", seen)
	}
	bg := paint(9, 9, 11, 255).RGBA
	for _, p := range []struct {
		x, y int
		want color.RGBA
	}{
		{20, 20, bg}, {20, 59, bg}, {199, 20, bg}, {199, 59, bg},
		{79, 20, edge.RGBA}, {80, 20, edge.RGBA}, {79, 59, edge.RGBA}, {140, 59, edge.RGBA},
	} {
		if got := img.RGBAAt(p.x, p.y); color.RGBA(got) != p.want {
			t.Errorf("pixel (%d,%d) is %v, want %v: outer corners round, inner corners square", p.x, p.y, got, p.want)
		}
	}
}

func TestCornerDrawerClip(t *testing.T) {
	green := paint(0, 212, 146, 255)
	lg := 0.5 * float64(cell.Y)
	drawer := box(2, 1, 20, 4, paint(39, 39, 42, 255))
	drawer.HidesOverflow = true
	drawer.Border.Radius = rounded(style.RadiusLg, style.CornerTopLeft, style.CornerTopRight)
	child := box(2, 1, 20, 4, green)
	child.Clip = drawer.Padding
	drawer.Children = []Node{child}
	f, img := drawn(page(drawer))
	var mask []raster.Box
	for _, op := range f.Layers[0].Ops {
		if op.Kind == raster.Clip && op.Box.Radii != [4]float64{} {
			mask = append(mask, op.Box)
		}
	}
	if len(mask) != 1 || mask[0].Radii != [4]float64{lg, lg, 0, 0} {
		t.Fatalf("masks %+v, want one with radii [%v %v 0 0]", mask, lg, lg)
	}
	for _, p := range []struct {
		x, y int
		want color.RGBA
	}{{20, 20, paint(9, 9, 11, 255).RGBA}, {219, 20, paint(9, 9, 11, 255).RGBA}, {20, 99, green.RGBA}, {219, 99, green.RGBA}, {120, 20, green.RGBA}} {
		if got := img.RGBAAt(p.x, p.y); color.RGBA(got) != p.want {
			t.Errorf("pixel (%d,%d) is %v, want %v: the child masked at the top corners only", p.x, p.y, got, p.want)
		}
	}
}

func TestCornerRingInsetShrinksEachCorner(t *testing.T) {
	b := box(2, 1, 10, 4, paint(24, 24, 27, 255))
	b.Border = Border{Style: style.BorderSingle, Radius: rounded(style.RadiusLg, style.CornerTopLeft), Top: true, Right: true, Bottom: true, Left: true, Color: paint(63, 63, 70, 255)}
	b.InsetShadows = []style.Shadow{{Y: 2, Blur: 4, Color: paint(0, 0, 0, 40)}}
	for _, op := range record(page(b)).Layers[0].Ops {
		if op.Kind == raster.Shadow && op.Box.Radii != [4]float64{0.5*float64(cell.Y) - 1, 0, 0, 0} {
			t.Errorf("inset shadow radii %v, want the top left radius less the 1 px border, the rest 0", op.Box.Radii)
		}
	}
}
