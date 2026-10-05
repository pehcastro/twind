package scene

import (
	"image"
	"testing"

	"github.com/pehcastro/twind/twi/raster"
	"github.com/pehcastro/twind/twi/style"
)

func TestTurnCoversTheTurnedCorners(t *testing.T) {
	for _, c := range []struct {
		turn float64
		want image.Rectangle
	}{
		{0, image.Rect(20, 20, 80, 60)},
		{0.125, image.Rect(14, 4, 86, 76)},
		{0.25, image.Rect(30, 10, 70, 70)},
		{0.5, image.Rect(20, 20, 80, 60)},
	} {
		b := box(2, 1, 6, 2, paint(24, 24, 27, 255))
		b.Turn = c.turn
		if got := record(page(b)).Layers[0].Boxes[1].Visual; !c.want.In(got) || !got.In(c.want.Inset(-1)) {
			t.Errorf("a 60x40 px box at turn %v covers %v, want %v give or take a pixel", c.turn, got, c.want)
		}
	}
}

func spinner(turn float64) Node {
	b := box(2, 1, 6, 2, paint(24, 24, 27, 255))
	b.Border = Border{Style: style.BorderSingle, Radius: style.RadiusLg, Top: true, Color: paint(63, 63, 70, 255)}
	b.Shadows = []style.Shadow{{Y: 4, Blur: 6, Color: paint(0, 0, 0, 64)}}
	b.Turn = turn
	return b
}

func TestTurnReachesTheOps(t *testing.T) {
	ops := record(page(spinner(0.125))).Layers[0].Boxes[1].Ops
	if len(ops) != 3 {
		t.Fatalf("a shadowed box with a top border gave %d ops, want shadow, fill and the top line", len(ops))
	}
	for _, op := range ops {
		if op.Turn != 0.125 {
			t.Errorf("op %d turn %v, want 0.125", op.Kind, op.Turn)
		}
	}
	if line := ops[2]; line.Pivot != (raster.Point{Y: 19.5}) {
		t.Errorf("the top line of a 60x40 px box turns about %+v from its own centre, want the box centre 19.5 px below", line.Pivot)
	}
}

func TestTurnDamagesTheSwungCorners(t *testing.T) {
	gradient := func(turn float64) Node {
		b := box(2, 1, 6, 2, paint(0, 0, 0, 0))
		b.Gradient = style.Gradient{GradientLine: style.GradientLine{Kind: style.GradientLinear, Direction: style.ToRight},
			From: style.GradientStop{Color: paint(79, 70, 229, 255)}, To: style.GradientStop{Color: paint(219, 39, 119, 255), Position: 1}}
		b.Turn = turn
		return b
	}
	lined := func(turn float64) Node {
		b := spinner(turn)
		b.Shadows = nil
		return b
	}
	for _, c := range []struct {
		name       string
		prev, next Node
	}{
		{"a lined box from an eighth to three eighths, the same turned bounds", lined(0.125), lined(0.375)},
		{"upright to an eighth", spinner(0), spinner(0.125)},
		{"an eighth to a little more", spinner(0.125), spinner(0.13)},
		{"a gradient half way round", gradient(0), gradient(0.5)},
	} {
		d := damage(page(c.prev), page(c.next))
		var got image.Rectangle
		for _, r := range d.Rects {
			got = got.Union(r)
		}
		for _, n := range []Node{c.prev, c.next} {
			if want := record(page(n)).Layers[0].Boxes[1].Visual; want.Empty() || !want.In(got) {
				t.Errorf("%s: damage %v does not cover the turned box %v", c.name, got, want)
			}
		}
	}
}
