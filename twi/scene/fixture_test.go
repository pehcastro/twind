package scene

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/style"
)

func benchTrees() [2]Node {
	var trees [2]Node
	for t := range trees {
		root := page()
		for i := range 50 {
			c := card(paint(24, 24, uint8(27+t*(i%2)), 255))
			c.Bounds.Y = i * 5
			c.Border = Border{Style: style.BorderSingle, Radius: style.RadiusLg, Top: true, Right: true, Bottom: true, Left: true, Color: paint(63, 63, 70, 255)}
			for j := range 19 {
				row := box(3, i*5+j%3, 8, 1, paint(39, 39, 42, uint8(255*(j%2))))
				row.text = Sanitize("row")
				c.Children = append(c.Children, row)
			}
			root.Children = append(root.Children, c)
		}
		root.Children = append(root.Children, popover(5+t, 5))
		trees[t] = root
	}
	return trees
}

func fixturePairs() [][2]Node {
	trees := benchTrees()
	backdrop := box(0, 0, 40, 12, paint(0, 0, 0, 128))
	backdrop.Position = layout.PositionFixed
	under := box(1, 1, 6, 2, paint(255, 0, 0, 255))
	under.Position, under.ZIndex = layout.PositionFixed, -1
	faded := popover(5, 5)
	faded.Opacity = 0.5
	nested := track(style.RadiusFull, box(2, 1, 12, 1, paint(0, 212, 146, 255)))
	nested.Children[0].HidesOverflow, nested.Children[0].Border.Radius = true, style.RadiusMd
	nested.Children[0].Children = []Node{box(2, 1, 3, 1, paint(1, 2, 3, 255))}
	nested.Children[0].Children[0].Clip = nested.Children[0].Padding
	leaf := track(style.RadiusFull, box(2, 1, 12, 1, paint(0, 212, 146, 255)))
	absolute := box(4, 2, 5, 3, paint(9, 90, 9, 255))
	absolute.Position = layout.PositionAbsolute
	above := popover(6, 6)
	above.Position, above.ZIndex = layout.PositionRelative, 2
	dashed := card(paint(24, 24, 27, 255))
	dashed.Border = Border{Style: style.BorderDashed, Color: paint(255, 255, 255, 255), Bottom: true, Left: true}
	dashed.InsetShadows = []style.Shadow{{Y: 2, Blur: 4, Color: paint(0, 0, 0, 40)}}
	return [][2]Node{
		trees,
		{scroller(3), scroller(4)},
		{page(card(paint(24, 24, 27, 255))), page(card(paint(24, 24, 27, 255)), backdrop, popover(10, 3))},
		{page(under, card(paint(24, 24, 27, 255))), page(card(paint(24, 24, 27, 255)), under)},
		{page(popover(5, 5)), page(faded)},
		{page(nested, absolute), page(absolute, nested, above)},
		{page(leaf), page(leaf, nested)},
		{page(dashed), page(card(paint(24, 24, 27, 255)))},
	}
}

func dump(out *strings.Builder, f *Frame) {
	ops := func(list []raster.Op, indent string) string {
		var s strings.Builder
		for _, op := range list {
			fmt.Fprintf(&s, "%s%d %v %v %v %v %v %v %v %v %v\n", indent, op.Kind, op.Box.Rect, op.Box.Radii, op.Color, op.Stops, op.Angle, op.Width, op.Dash, op.Shadow, op.Opacity)
		}
		return s.String()
	}
	for i, l := range f.Layers {
		fmt.Fprintf(out, "layer %d parent %d origin %v opacity %v visual %v clip %v ops %d boxes %d\n", i, l.Parent, l.Origin, l.Opacity, l.Visual, l.Clip, len(l.Ops), len(l.Boxes))
		var boxed, joined strings.Builder
		for _, b := range l.Boxes {
			fmt.Fprintf(&boxed, "  box %v look %x ops %d\n%s", b.Visual, b.Look, len(b.Ops), ops(b.Ops, "    "))
			joined.WriteString(ops(b.Ops, "    "))
		}
		if all := ops(l.Ops, "    "); joined.String() != all {
			out.WriteString(all)
		}
		out.WriteString(boxed.String())
	}
}

func displayList(frames *[2]Frame, round int) string {
	var out strings.Builder
	for i, pair := range fixturePairs() {
		prev, next := &frames[round], &frames[1-round]
		prev.Record(&pair[0], cell)
		next.Record(&pair[1], cell)
		fmt.Fprintf(&out, "pair %d\n", i)
		dump(&out, prev)
		dump(&out, next)
		fmt.Fprintf(&out, "damage %+v\nback %+v\n", Diff(prev, next), Diff(next, prev))
	}
	return out.String()
}

func TestDisplayListAndDamageMatchFixture(t *testing.T) {
	want, err := os.ReadFile("testdata/display.txt")
	if err != nil {
		t.Fatal(err)
	}
	var frames [2]Frame
	if displayList(&frames, 0) != displayList(&frames, 1) {
		t.Fatal("recording into reused frames changed the display list")
	}
	if got := displayList(&frames, 0); got != string(want) {
		lines, wants := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := range min(len(lines), len(wants)) {
			if lines[i] != wants[i] {
				t.Fatalf("line %d differs from the display list recorded at 2345815:\n got %s\nwant %s", i+1, lines[i], wants[i])
			}
		}
		t.Fatalf("%d lines, want %d as recorded at 2345815", len(lines), len(wants))
	}
}
