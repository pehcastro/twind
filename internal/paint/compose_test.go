package paint

import (
	"math"
	"testing"

	"github.com/pehcastro/twind/internal/buffer"
	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/twi/color"
)

func halfOver(src, dst color.RGBA) color.RGBA {
	mix := func(s, d uint8) uint8 { return uint8(math.Round((float64(s) + float64(d)) / 2)) }
	return color.RGBA{R: mix(src.R, dst.R), G: mix(src.G, dst.G), B: mix(src.B, dst.B), A: 255}
}

func near(got color.Color, want color.RGBA) bool {
	d := func(a, b uint8) bool { return math.Abs(float64(a)-float64(b)) <= 1 }
	return got.Kind == color.Literal && d(got.RGBA.R, want.R) && d(got.RGBA.G, want.G) && d(got.RGBA.B, want.B)
}

func TestBlendKeepsGlyphBeneath(t *testing.T) {
	box := sized(3, 1, layout.Edges{})
	s := filled(zinc950)
	s.Color = literal(white)
	page := scene.New(box, s, scene.Sanitize("x"))
	page.Children = []scene.Node{scene.New(box, filled(color.RGBA{A: 128}), scene.Text{})}
	buf := buffer.New(3, 1)
	Paint(buf, page, Composited)
	black := color.RGBA{A: 255}
	c := buf.At(0, 0)
	if c.Grapheme != "x" {
		t.Errorf("glyph %q under bg-black/50, want x kept", c.Grapheme)
	}
	if want := halfOver(black, white); !near(c.Fg, want) {
		t.Errorf("fg %+v, want %+v", c.Fg, want)
	}
	if want := halfOver(black, zinc950); !near(c.Bg, want) || !near(buf.At(2, 0).Bg, want) {
		t.Errorf("bg %+v and %+v, want %+v", c.Bg, buf.At(2, 0).Bg, want)
	}
}

func TestBlendOpaqueReplaces(t *testing.T) {
	box := sized(2, 1, layout.Edges{})
	s := filled(zinc950)
	s.Color = literal(white)
	page := scene.New(box, s, scene.Sanitize("xy"))
	page.Children = []scene.Node{scene.New(box, filled(zinc800), scene.Text{})}
	buf := buffer.New(2, 1)
	Paint(buf, page, Composited)
	if c := buf.At(0, 0); c != (buffer.Cell{Grapheme: " ", Bg: literal(zinc800)}) {
		t.Errorf("opaque layer left %+v", c)
	}
}

func TestBlendOverTerminalDefault(t *testing.T) {
	box := sized(2, 1, layout.Edges{})
	page := scene.New(box, plain(), scene.Sanitize("x"))
	page.Children = []scene.Node{scene.New(box, filled(color.RGBA{A: 128}), scene.Text{})}
	buf := buffer.New(2, 1)
	Paint(buf, page, Composited)
	c := buf.At(0, 0)
	if c.Grapheme != "x" || c.Fg.Kind != color.Unset {
		t.Errorf("default-coloured x became %+v, want the glyph kept with the default fg", c)
	}
	if c.Bg.Kind != color.Literal || c.Bg.RGBA != (color.RGBA{A: 128}) {
		t.Errorf("bg over the terminal default %+v, want the layer colour", c.Bg)
	}
}

func TestBlendWideGlyph(t *testing.T) {
	box := sized(4, 1, layout.Edges{})
	s := filled(zinc950)
	s.Color = literal(white)
	page := scene.New(box, s, scene.Sanitize("中x"))
	page.Children = []scene.Node{scene.New(sized(1, 1, layout.Edges{}), filled(color.RGBA{A: 128}), scene.Text{})}
	buf := buffer.New(4, 1)
	Paint(buf, page, Composited)
	expect(t, buf, "中x ")
	if buf.At(0, 0).Width != buffer.Wide || buf.At(1, 0).Width != buffer.Continuation {
		t.Errorf("a translucent cell over half a wide glyph split it: %+v %+v", buf.At(0, 0), buf.At(1, 0))
	}
}

func TestOpacityGroup(t *testing.T) {
	box := sized(4, 1, layout.Edges{})
	card := filled(zinc100)
	card.Color = literal(zinc950)
	faded := filled(zinc800)
	faded.Opacity = 0.5

	group := scene.New(box, faded, scene.Text{})
	group.Children = []scene.Node{scene.New(box, card, scene.Sanitize("hi"))}
	page := scene.New(box, filled(zinc950), scene.Text{})
	page.Children = []scene.Node{group}
	got := buffer.New(4, 1)
	Paint(got, page, Composited)

	half := card
	half.Background.RGBA.A, half.Color.RGBA.A = 128, 128
	page = scene.New(box, filled(zinc950), scene.Text{})
	page.Children = []scene.Node{scene.New(box, half, scene.Sanitize("hi"))}
	want := buffer.New(4, 1)
	Paint(want, page, Composited)

	for x := range 4 {
		if got.At(x, 0) != want.At(x, 0) {
			t.Errorf("cell %d: opacity-50 group %+v, want the card at 0.5 %+v", x, got.At(x, 0), want.At(x, 0))
		}
	}
	if !near(got.At(2, 0).Bg, halfOver(zinc100, zinc950)) {
		t.Errorf("bg %+v, want zinc-100 at 0.5 over zinc-950", got.At(2, 0).Bg)
	}
}

func TestOpacityNestedAndZero(t *testing.T) {
	box := sized(2, 1, layout.Edges{})
	outer, inner, gone := filled(white), filled(white), filled(zinc100)
	outer.Opacity, inner.Opacity, gone.Opacity = 0.5, 0.5, 0
	o := scene.New(box, outer, scene.Text{})
	o.Children = []scene.Node{scene.New(box, inner, scene.Text{})}
	page := scene.New(box, filled(zinc950), scene.Text{})
	page.Children = []scene.Node{o, scene.New(box, gone, scene.Sanitize("zz"))}
	buf := buffer.New(2, 1)
	Paint(buf, page, Composited)
	if want := halfOver(white, zinc950); !near(buf.At(0, 0).Bg, want) {
		t.Errorf("white group over white at 0.5 gave %+v, want %+v", buf.At(0, 0).Bg, want)
	}
	if buf.At(0, 0).Grapheme != " " {
		t.Errorf("opacity-0 painted %q", buf.At(0, 0).Grapheme)
	}
}

func absolute(x, y, w, h, z int) *layout.Box {
	return &layout.Box{Style: layout.Style{
		Position: layout.PositionAbsolute, ZIndex: z, Width: cells(w), Height: cells(h),
		Inset: layout.Insets{Left: cells(x), Top: cells(y)},
	}}
}

func TestStackZIndex(t *testing.T) {
	top, under := absolute(0, 0, 4, 2, 20), absolute(2, 1, 4, 2, 10)
	root := &layout.Box{Style: layout.Style{Width: cells(8), Height: cells(3)}, Children: []*layout.Box{top, under}}
	layout.Layout(root, 8, cells(3))
	page := scene.New(root, filled(zinc950), scene.Text{})
	page.Children = []scene.Node{scene.New(top, filled(red), scene.Text{}), scene.New(under, filled(blue), scene.Text{})}
	buf := buffer.New(8, 3)
	Paint(buf, page, Composited)
	if c := buf.At(3, 1).Bg.RGBA; c != red {
		t.Errorf("overlap %+v, want the z-20 box (red) above the later z-10 box", c)
	}
	if c := buf.At(5, 2).Bg.RGBA; c != blue {
		t.Errorf("z-10 box outside the overlap %+v, want blue", c)
	}
}

func TestStackHoistsOutOfPositionedWrapper(t *testing.T) {
	popover := absolute(0, 0, 3, 1, 50)
	wrapper := &layout.Box{Style: layout.Style{Position: layout.PositionRelative, Width: cells(3), Height: cells(1)}, Children: []*layout.Box{popover}}
	later := absolute(0, 0, 6, 1, 0)
	root := &layout.Box{Style: layout.Style{Width: cells(6), Height: cells(1)}, Children: []*layout.Box{wrapper, later}}
	layout.Layout(root, 6, cells(1))
	w := scene.New(wrapper, plain(), scene.Text{})
	w.Children = []scene.Node{scene.New(popover, filled(red), scene.Text{})}
	page := scene.New(root, plain(), scene.Text{})
	page.Children = []scene.Node{w, scene.New(later, filled(blue), scene.Text{})}
	buf := buffer.New(6, 1)
	Paint(buf, page, Composited)
	if c := buf.At(0, 0).Bg.RGBA; c != red {
		t.Errorf("z-50 popover inside a z-auto relative wrapper: %+v, want red above the later positioned box", c)
	}
}

func TestStackNegativeZ(t *testing.T) {
	behind := absolute(0, 0, 4, 1, -1)
	flow := &layout.Box{Style: layout.Style{Width: cells(2), Height: cells(1)}}
	root := &layout.Box{Style: layout.Style{Width: cells(4), Height: cells(1)}, Children: []*layout.Box{flow, behind}}
	layout.Layout(root, 4, cells(1))
	page := scene.New(root, filled(zinc950), scene.Text{})
	page.Children = []scene.Node{scene.New(flow, filled(red), scene.Text{}), scene.New(behind, filled(blue), scene.Text{})}
	buf := buffer.New(4, 1)
	Paint(buf, page, Composited)
	if buf.At(0, 0).Bg.RGBA != red || buf.At(3, 0).Bg.RGBA != blue {
		t.Errorf("-z-1 box: %+v %+v, want above the root background and below the flow child", buf.At(0, 0).Bg.RGBA, buf.At(3, 0).Bg.RGBA)
	}
}

func TestStackOverflowHiddenClips(t *testing.T) {
	child := &layout.Box{Style: layout.Style{Width: cells(8), Height: cells(2)}}
	fixed := &layout.Box{Style: layout.Style{Position: layout.PositionFixed, Width: cells(8), Height: cells(1), Inset: layout.Insets{Top: cells(2)}}}
	clipper := &layout.Box{Style: layout.Style{Width: cells(4), Height: cells(2), Overflow: layout.OverflowHidden}, Children: []*layout.Box{child, fixed}}
	root := &layout.Box{Style: layout.Style{Width: cells(10), Height: cells(3)}, Children: []*layout.Box{clipper}}
	layout.Layout(root, 10, cells(3))
	c := scene.New(clipper, plain(), scene.Text{})
	c.Children = []scene.Node{scene.New(child, filled(blue), scene.Sanitize("abcdefgh")), scene.New(fixed, filled(red), scene.Sanitize("fixedbox"))}
	page := scene.New(root, plain(), scene.Text{})
	page.Children = []scene.Node{c}
	buf := buffer.New(10, 3)
	Paint(buf, page, Composited)
	expect(t, buf, "abcd      ", "          ", "fixedbox  ")
	if buf.At(4, 0).Bg.Kind != color.Unset || buf.At(3, 1).Bg.RGBA != blue {
		t.Errorf("clip edge: %+v inside, %+v outside", buf.At(3, 1).Bg, buf.At(4, 0).Bg)
	}
}
