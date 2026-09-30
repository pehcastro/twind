package scene

import (
	"image"
	"slices"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/style"
)

var (
	cell     = image.Pt(10, 20)
	viewport = layout.Rect{W: 40, H: 12}
)

func paint(r, g, b, a uint8) color.Color {
	return color.Color{Kind: color.Literal, RGBA: color.RGBA{R: r, G: g, B: b, A: a}}
}

func box(x, y, w, h int, bg color.Color) Node {
	r := layout.Rect{X: x, Y: y, W: w, H: h}
	return Node{Bounds: r, Padding: r, Content: r, Clip: viewport, Opacity: 1, Background: bg}
}

func page(children ...Node) Node {
	root := box(0, 0, 40, 12, paint(9, 9, 11, 255))
	root.Children = children
	return root
}

func card(bg color.Color) Node {
	c := box(2, 1, 10, 4, bg)
	c.Shadows = []style.Shadow{{X: 8, Y: 16, Blur: 16, Color: paint(0, 0, 0, 64)}}
	c.text = Sanitize("hello")
	return c
}

func popover(x, y int) Node {
	p := box(x, y, 10, 3, paint(255, 255, 255, 255))
	p.Position = layout.PositionFixed
	p.Shadows = []style.Shadow{{Y: 16, Blur: 16, Color: paint(0, 0, 0, 64)}}
	return p
}

func TestShadowPixelsScaleByRem(t *testing.T) {
	c := box(2, 1, 10, 4, paint(255, 255, 255, 255))
	black := paint(0, 0, 0, 26)
	c.Shadows = []style.Shadow{{Y: 4, Blur: 6, Spread: -1, Color: black}, {Y: 2, Blur: 4, Spread: -2, Color: black}}
	var got []raster.BoxShadow
	for _, op := range record(page(c)).Layers[0].Ops {
		if op.Kind == raster.Shadow {
			got = append(got, op.Shadow)
		}
	}
	want := []raster.BoxShadow{{Y: 2.5, Blur: 5, Spread: -2.5}, {Y: 5, Blur: 7.5, Spread: -1.25}}
	if !slices.Equal(got, want) {
		t.Errorf("shadow-md on a 10x20 cell: %+v, want %+v, every px times 20/16 whatever the axis", got, want)
	}
}

func record(root Node) *Frame {
	f := new(Frame)
	f.Record(&root, cell)
	return f
}

func damage(prev, next Node) Damage { return Diff(record(prev), record(next)) }

func TestDamageTextOnly(t *testing.T) {
	typed := card(paint(24, 24, 27, 255))
	typed.text = Sanitize("hello, world")
	typed.Foreground = paint(250, 0, 0, 255)
	typed.Bold = true
	if d := damage(page(card(paint(24, 24, 27, 255))), page(typed)); len(d.Rects) != 0 || len(d.Moves) != 0 {
		t.Errorf("a text-only change gave %+v, want no damage and no move", d)
	}
}

func TestDamageCardBackground(t *testing.T) {
	d := damage(page(card(paint(24, 24, 27, 255))), page(card(paint(39, 39, 42, 255))))
	if want := []image.Rectangle{image.Rect(0, 10, 160, 150)}; !slices.Equal(d.Rects, want) || len(d.Moves) != 0 {
		t.Errorf("a card background change gave %+v, want rects %v: box 20,20-120,100 and its shadow 0,10-160,150", d, want)
	}
}

func TestDamageFixedMove(t *testing.T) {
	d := damage(page(card(paint(24, 24, 27, 255)), popover(5, 5)), page(card(paint(24, 24, 27, 255)), popover(8, 6)))
	if want := []Move{{Layer: 1, From: image.Pt(50, 100), To: image.Pt(80, 120)}}; !slices.Equal(d.Moves, want) || len(d.Rects) != 0 {
		t.Errorf("moving a fixed popover gave %+v, want moves %v and no rects", d, want)
	}
}

func TestDamageOpenPopover(t *testing.T) {
	closed, open := page(card(paint(24, 24, 27, 255))), page(card(paint(24, 24, 27, 255)), popover(5, 5))
	want := []image.Rectangle{image.Rect(20, 90, 180, 210)}
	if d := damage(closed, open); !slices.Equal(d.Rects, want) || len(d.Moves) != 0 {
		t.Errorf("opening a popover gave %+v, want rects %v", d, want)
	}
	if d := damage(open, closed); !slices.Equal(d.Rects, want) || len(d.Moves) != 0 {
		t.Errorf("closing a popover gave %+v, want rects %v", d, want)
	}
}

func TestDamageOpacityAndOrder(t *testing.T) {
	faded := popover(5, 5)
	faded.Opacity = 0.5
	if d := damage(page(popover(5, 5)), page(faded)); len(d.Rects) == 0 {
		t.Errorf("fading a layer gave %+v, want its visual rect damaged", d)
	}
	a, b := box(2, 2, 6, 2, paint(255, 0, 0, 255)), box(4, 3, 6, 2, paint(0, 0, 255, 255))
	if d := damage(page(a, b), page(b, a)); len(d.Rects) == 0 {
		t.Errorf("swapping the paint order of two overlapping boxes gave %+v, want damage", d)
	}
}

func TestLayersBackdropDialog(t *testing.T) {
	backdrop := box(0, 0, 40, 12, paint(0, 0, 0, 128))
	backdrop.Position = layout.PositionFixed
	dialog := popover(10, 3)
	layers := record(page(card(paint(24, 24, 27, 255)), backdrop, dialog)).Layers
	parents := make([]int, len(layers))
	for i, l := range layers {
		parents[i] = l.Parent
	}
	if want := []int{-1, 0, 0}; !slices.Equal(parents, want) {
		t.Fatalf("backdrop and dialog gave layer parents %v, want %v: root, backdrop, dialog", parents, want)
	}
	if l := layers[2]; l.Origin != image.Pt(100, 60) || l.Ops[len(l.Ops)-1].Box.Rect != (raster.Rect{W: 100, H: 60}) {
		t.Errorf("dialog layer at %v with last op %+v, want origin 100,60 and a local 100x60 fill", l.Origin, l.Ops[len(l.Ops)-1])
	}
	if n := len(layers[0].Ops); n != 3 {
		t.Errorf("root layer has %d ops, want 3: page fill, card shadow, card fill", n)
	}
}

func TestLayersOpacityGroup(t *testing.T) {
	group := box(2, 2, 20, 5, paint(39, 39, 42, 255))
	group.Opacity = 0.5
	group.Children = []Node{box(3, 3, 5, 1, paint(250, 250, 250, 255))}
	layers := record(page(group)).Layers
	if len(layers) != 2 || layers[1].Opacity != 0.5 || layers[1].Parent != 0 || len(layers[1].Ops) != 2 {
		t.Fatalf("an opacity-50 group gave %+v, want a second layer at opacity 0.5 holding both fills", layers)
	}
	if got := layers[1].Ops[1].Box.Rect; got != (raster.Rect{X: 10, Y: 20, W: 50, H: 20}) {
		t.Errorf("child fill at %+v in group space, want 10,20 50x20", got)
	}
}

func BenchmarkLayers1000(b *testing.B) {
	trees := benchTrees()
	var frames [2]Frame
	frames[0].Record(&trees[0], cell)
	now := clock(b)
	var took []time.Duration
	for i := 1; b.Loop(); i++ {
		start := now()
		next := &frames[i%2]
		next.Record(&trees[i%2], cell)
		d := Diff(&frames[1-i%2], next)
		took = append(took, now()-start)
		if len(d.Rects) != 25 || len(d.Moves) != 1 {
			b.Fatalf("got %d rects and %d moves, want 25 changed cards and 1 move", len(d.Rects), len(d.Moves))
		}
	}
	slices.Sort(took)
	b.ReportMetric(float64(took[len(took)/2].Nanoseconds()), "p50-ns/frame")
	b.ReportMetric(float64(took[len(took)*95/100].Nanoseconds()), "p95-ns/frame")
}

func TestDamageGroupOpacityReachesChildLayers(t *testing.T) {
	group := func(opacity float64) Node {
		g := box(2, 2, 5, 2, paint(39, 39, 42, 255))
		g.Opacity = opacity
		g.Children = []Node{popover(20, 5)}
		return page(g)
	}
	d := damage(group(0.5), group(0.6))
	popped := image.Rect(200, 100, 300, 160)
	for _, r := range d.Rects {
		if popped.In(r) {
			return
		}
	}
	t.Errorf("fading a group gave %+v, want the fixed child's layer %v damaged too", d, popped)
}

func TestRootBackgroundPaintsTheCanvas(t *testing.T) {
	short := page()
	short.Bounds.H, short.Padding.H, short.Content.H = 5, 5, 5
	l := record(short).Layers[0]
	if got := l.Ops[0].Box.Rect; got != (raster.Rect{W: 400, H: 240}) || l.Visual != image.Rect(0, 0, 400, 240) {
		t.Errorf("a 5-row page in a 12-row viewport fills %+v with visual %v, want the whole 400x240 canvas", got, l.Visual)
	}
}

func TestLayersNegativeZ(t *testing.T) {
	under := box(1, 1, 6, 2, paint(255, 0, 0, 255))
	under.Position, under.ZIndex = layout.PositionFixed, -1
	layers := record(page(under, card(paint(24, 24, 27, 255)))).Layers
	if len(layers) != 3 || layers[2].Parent != 0 || len(layers[0].Ops) != 1 || len(layers[2].Ops) != 2 {
		t.Errorf("a z-1 fixed child gave %d layers %+v, want page fill, the child, then the card above it", len(layers), layers)
	}
}
