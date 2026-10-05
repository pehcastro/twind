package paint

import (
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"os"
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/buffer"
	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/style"
)

const (
	randomTrees = 160
	randomW     = 40
	randomH     = 14
)

func TestCellsMatchFixture(t *testing.T) {
	var got []string
	record := func(name string, frames ...*buffer.Buffer) {
		var cells []byte
		for _, f := range frames {
			cells = fmt.Appendf(cells, "%dx%d|", f.Width(), f.Height())
			for y := range f.Height() {
				for _, c := range f.Row(y) {
					cells = fmt.Appendf(cells, "%s\x00%d %v %d %v %d %d;", c.Grapheme, c.Fg.Kind, c.Fg.RGBA, c.Bg.Kind, c.Bg.RGBA, c.Attr, c.Width)
				}
			}
		}
		h := fnv.New64a()
		h.Write(cells)
		got = append(got, fmt.Sprintf("%s %016x", name, h.Sum64()))
	}
	retained := func(w, h int, frames ...scene.Node) []*buffer.Buffer {
		var p Painter
		buf := buffer.New(w, h)
		var out []*buffer.Buffer
		for i := range frames {
			p.Paint(buf, &frames[i], Composited)
			snap := buffer.New(w, h)
			for y := range h {
				copy(snap.Row(y), buf.Row(y))
			}
			out = append(out, snap)
		}
		return out
	}
	looks := []Look{Composited, Plain, Glyphs}
	for _, look := range looks {
		record(fmt.Sprintf("bench/look%d", look), painted(appColumns, appRows, app("keys 1"), look), painted(appColumns, appRows, app("keys 2"), look))
		record(fmt.Sprintf("scenery/look%d", look), painted(appColumns, appRows, scenery(), look))
	}
	record("bench/retained", retained(appColumns, appRows, app("keys 1"), app("keys 2"), app("keys 1"), scenery(), app("keys 2"))...)
	for i, f := range fixtureScenes() {
		for _, look := range looks {
			record(fmt.Sprintf("fixture%02d/look%d", i, look), painted(f.w, f.h, f.root, look))
		}
	}
	for seed := range uint64(randomTrees) {
		r := rand.New(rand.NewPCG(seed, seed^0x5eed))
		a := randomPage(r)
		b := jitter(r, clone(a))
		c := randomPage(r)
		frames := []*buffer.Buffer{
			painted(randomW, randomH, a, looks[seed%3]),
			painted(randomW, randomH, c, looks[(seed+1)%3]),
		}
		frames = append(frames, retained(randomW, randomH, a, b, a, c, b)...)
		record(fmt.Sprintf("random%03d", seed), frames...)
	}
	want, err := os.ReadFile("testdata/cells.txt")
	if err != nil {
		t.Fatalf("%v; the fixture would be:\n%s", err, strings.Join(got, "\n"))
	}
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(want), "\r\n", "\n")), "\n")
	if len(lines) != len(got) {
		t.Fatalf("%d cases, fixture has %d", len(got), len(lines))
	}
	wrong := 0
	for i := range got {
		if got[i] != lines[i] {
			wrong++
			t.Errorf("%s, fixture %s", got[i], lines[i])
		}
	}
	if wrong > 0 {
		t.Errorf("%d of %d cases paint different cells from ab09ef1", wrong, len(got))
	}
}

type fixtureScene struct {
	w, h int
	root scene.Node
}

func fixtureScenes() []fixtureScene {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	green100 := color.RGBA{R: 220, G: 252, B: 231, A: 255}
	pill := func(x int, fill bool, rings ...style.Shadow) scene.Node {
		s := plain()
		if fill {
			s = filled(green100)
		}
		s.Radius, s.Shadows = style.RadiusFull, rings
		return scene.New(place(x, 1, 4, 1, layout.Edges{}), s, scene.Text{})
	}
	bare := scene.New(place(0, 0, 10, 3, layout.Edges{}), plain(), scene.Text{})
	bare.Children = []scene.Node{ringed(2, 1, 4, focusRing(red, 51)...)}
	text := "abcdefghijkl\nmnopqrstuvwx\nABCDEFGHIJKL\nMNOPQRSTUVWX\nyz"
	md := style.Shadow{X: 8, Y: 16, Color: literal(shadowMd)}
	return []fixtureScene{
		{10, 5, page(10, 5, "", ringed(2, 1, 4, ring(zinc800, 1)), ringed(2, 3, 4, ring(zinc800, 1)))},
		{10, 5, page(10, 5, "", ringed(2, 1, 4, ring(red, 1)), ringed(2, 3, 4, ring(blue, 1)))},
		{10, 5, page(10, 5, "", ringed(2, 3, 4, ring(blue, 1)), ringed(2, 1, 4, ring(red, 1)))},
		{10, 5, page(10, 5, "", ringed(2, 1, 4, focusRing(blue, 128)...), ringed(2, 3, 4, ring(zinc800, 1)))},
		{10, 5, page(10, 5, "", ringed(2, 3, 4, ring(zinc800, 1)), ringed(2, 1, 4, focusRing(blue, 128)...))},
		{10, 5, page(10, 5, "", ringed(1, 1, 2, focusRing(blue, 128)...), ringed(4, 1, 2, ring(zinc800, 1)))},
		{10, 5, page(10, 5, "", ringed(4, 1, 2, focusRing(blue, 128)...), ringed(1, 1, 2, ring(zinc800, 1)))},
		{10, 3, page(10, 3, "abcdefghij\n\nklmnopqrst", ringed(2, 1, 4, focusRing(blue, 128)...))},
		{12, 3, page(12, 3, "", pill(4, true, ring(zinc800, 1)))},
		{12, 3, page(12, 3, "", pill(4, false, ring(zinc800, 1)))},
		{10, 3, page(10, 3, "", ringed(2, 1, 4, ring(red, 1)))},
		{10, 3, page(10, 3, "", ringed(2, 1, 4, focusRing(red, 51)...))},
		{10, 3, bare},
		{12, 6, page(12, 6, "", card(place(1, 1, 10, 4, one), style.RadiusNone))},
		{12, 6, page(12, 6, text, card(place(1, 1, 10, 4, one), style.RadiusLg))},
		{6, 3, page(6, 3, "", card(place(1, 0, 4, 2, layout.Edges{Bottom: 1}), style.RadiusLg))},
		{6, 3, page(6, 3, "", card(place(0, 0, 6, 3, one), style.RadiusNone))},
		{6, 3, page(6, 3, "", card(place(0, 0, 6, 3, one), style.RadiusLg))},
		{12, 7, page(12, 7, "", card(place(1, 1, 6, 3, layout.Edges{}), style.RadiusNone, md))},
	}
}

func randomPage(r *rand.Rand) scene.Node {
	screen := layout.Rect{W: randomW, H: randomH}
	root := randomNode(r, screen, screen)
	root.Bounds, root.Padding, root.Content = screen, screen, screen
	root.Position, root.ZIndex, root.TopLayer, root.Opacity = layout.PositionStatic, 0, 0, []float64{1, 1, 1, 0.6}[r.IntN(4)]
	var grow func(n *scene.Node, depth int)
	grow = func(n *scene.Node, depth int) {
		if depth == 4 {
			return
		}
		for range r.IntN(5 - depth) {
			clip := layout.Rect{W: randomW, H: randomH}
			if r.IntN(3) == 0 {
				clip = overlap(n.Padding, n.Clip)
			}
			c := randomNode(r, n.Padding, clip)
			grow(&c, depth+1)
			n.Children = append(n.Children, c)
		}
	}
	grow(&root, 0)
	return root
}

func randomNode(r *rand.Rand, in, clip layout.Rect) scene.Node {
	palette := []color.Color{
		{}, literal(zinc950), literal(zinc100), literal(red), literal(blue), literal(white),
		literal(color.RGBA{R: 255, A: 100}), literal(color.RGBA{B: 255, A: 60}), literal(color.RGBA{}), {Kind: color.Current},
	}
	pick := func() color.Color { return palette[r.IntN(len(palette))] }
	texts := []string{"", "", "hello world", "a\tb\tcd", "中文x中", "éclair and more words", "wrap me please over several lines", "👍🏽 ok", "line1\nline2\nline3", "x"}
	s := plain()
	s.Color = palette[1+r.IntN(5)]
	if r.IntN(2) == 0 {
		s.Background = pick()
	}
	s.BorderStyle = style.BorderStyle(r.IntN(5))
	s.BorderColor = pick()
	s.Radius = []style.Radius{style.RadiusNone, style.RadiusLg, style.RadiusFull}[r.IntN(3)]
	s.Bold, s.Italic, s.Underline, s.Strikethrough = r.IntN(4) == 0, r.IntN(5) == 0, r.IntN(5) == 0, r.IntN(6) == 0
	s.TextAlign = style.TextAlign(r.IntN(4))
	for range r.IntN(3) {
		switch r.IntN(3) {
		case 0:
			s.Shadows = append(s.Shadows, style.Shadow{Spread: style.Pixels(1 + r.IntN(3)), Color: pick()})
		case 1:
			s.Shadows = append(s.Shadows, style.Shadow{X: style.Pixels(r.IntN(17) - 8), Y: style.Pixels(r.IntN(17)), Blur: style.Pixels(r.IntN(7)), Spread: style.Pixels(r.IntN(3) - 1), Color: pick()})
		default:
			s.InsetShadows = append(s.InsetShadows, style.Shadow{Y: style.Pixels(r.IntN(9)), Blur: style.Pixels(r.IntN(5)), Color: pick()})
		}
	}
	if r.IntN(10) == 0 {
		s.Gradient = style.Gradient{
			GradientLine: style.GradientLine{Kind: style.GradientLinear, Direction: style.GradientDirection(r.IntN(4))},
			From:         style.GradientStop{Color: literal(blue), Position: 0},
			To:           style.GradientStop{Color: literal(color.RGBA{R: 200, G: 40, B: 90, A: 255}), Position: 1},
		}
	}
	edges := layout.Edges{Top: r.IntN(2), Right: r.IntN(2), Bottom: r.IntN(2), Left: r.IntN(2)}
	bounds := layout.Rect{X: in.X + r.IntN(max(in.W, 1)+4) - 2, Y: in.Y + r.IntN(max(in.H, 1)+2) - 1, W: r.IntN(24), H: r.IntN(7)}
	if r.IntN(4) == 0 {
		bounds.H = 1
	}
	pad := layout.Rect{X: bounds.X + edges.Left, Y: bounds.Y + edges.Top, W: max(bounds.W-edges.Left-edges.Right, 0), H: max(bounds.H-edges.Top-edges.Bottom, 0)}
	content := pad
	if r.IntN(3) == 0 && content.W > 2 && content.H > 2 {
		content = layout.Rect{X: pad.X + 1, Y: pad.Y + 1, W: pad.W - 2, H: pad.H - 2}
	}
	box := &layout.Box{Style: layout.Style{Border: edges}, BorderBox: bounds, PaddingBox: pad, ContentBox: content, Clip: clip}
	n := scene.New(box, s, scene.Sanitize(texts[r.IntN(len(texts))]))
	n.Truncate, n.NoWrap = r.IntN(6) == 0, r.IntN(6) == 0
	switch r.IntN(12) {
	case 0:
		n.Position, n.ZIndex = layout.PositionAbsolute, r.IntN(5)-2
	case 1:
		n.Position, n.ZIndex = layout.PositionRelative, r.IntN(3)-1
	case 2:
		n.Position = layout.PositionFixed
	case 3:
		n.TopLayer = 1 + r.IntN(2)
	case 4:
		n.Opacity = []float64{0, 0.5, 0.8}[r.IntN(3)]
	case 5:
		n.Scroll, n.HidesOverflow = true, true
		n.ScrollContent = layout.Rect{X: pad.X, Y: pad.Y - r.IntN(3), W: pad.W, H: pad.H + 4}
	}
	return n
}

func clone(n scene.Node) scene.Node {
	n.Children = append([]scene.Node(nil), n.Children...)
	n.Shadows = append([]style.Shadow(nil), n.Shadows...)
	for i := range n.Children {
		n.Children[i] = clone(n.Children[i])
	}
	return n
}

func jitter(r *rand.Rand, n scene.Node) scene.Node {
	var visit func(n *scene.Node)
	visit = func(n *scene.Node) {
		switch r.IntN(8) {
		case 0:
			n.Background = literal(color.RGBA{R: uint8(r.IntN(256)), G: 90, B: 30, A: []uint8{255, 120}[r.IntN(2)]})
		case 1:
			move(n, r.IntN(5)-2, r.IntN(3)-1)
		case 2:
			n.Foreground = literal(red)
		case 3:
			if len(n.Shadows) > 0 {
				n.Shadows[0].Spread++
			}
		}
		for i := range n.Children {
			visit(&n.Children[i])
		}
	}
	visit(&n)
	return n
}
