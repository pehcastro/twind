package paint

import (
	"slices"
	"strconv"
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/paint"
	"github.com/pehcastro/twind/twi/buffer"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/layout"
	"github.com/pehcastro/twind/twi/scene"
	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/text"
)

const (
	appColumns = 80
	appRows    = 24
	appNodes   = 1000
)

type spec struct {
	box   *layout.Box
	style style.ComputedStyle
	text  scene.Text
	kids  []*spec
}

func (s *spec) node() scene.Node {
	n := scene.New(s.box, s.style, s.text)
	for _, k := range s.kids {
		n.Children = append(n.Children, k.node())
	}
	return n
}

func (s *spec) add(k *spec) {
	s.kids = append(s.kids, k)
	s.box.Children = append(s.box.Children, k.box)
}

func words(raw string) *spec {
	t := scene.Sanitize(raw)
	measure := func(available int) (int, int) { return t.Size(text.Wrapping{}, available) }
	return &spec{box: &layout.Box{Measure: measure}, style: inked(plain()), text: t}
}

func inked(s style.ComputedStyle) style.ComputedStyle {
	s.Color = literal(zinc100)
	return s
}

func app(keys string) scene.Node {
	left := appNodes - 3
	levels := []layout.Style{
		{Direction: layout.Column, Border: layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}},
		{Direction: layout.Row, ColumnGap: 1},
		{Direction: layout.Column, Grow: 1},
	}
	var grow func(depth int) *spec
	grow = func(depth int) *spec {
		left--
		if depth == len(levels) {
			return words("item " + strconv.Itoa(left))
		}
		s := inked(plain())
		if depth == 0 {
			s.BorderStyle, s.BorderColor = style.BorderSingle, color.Color{Kind: color.Current}
		}
		g := &spec{box: &layout.Box{Style: levels[depth]}, style: s}
		for range 4 {
			if left == 0 {
				break
			}
			g.add(grow(depth + 1))
		}
		return g
	}
	list := &spec{box: &layout.Box{Style: layout.Style{Direction: layout.Column}}, style: inked(filled(zinc950))}
	for left > 0 {
		list.add(grow(0))
	}
	root := &spec{box: &layout.Box{Style: layout.Style{Direction: layout.Column, Width: cells(appColumns), Height: cells(appRows)}}, style: inked(plain())}
	root.add(words(keys))
	root.add(list)
	layout.Layout(root.box, appColumns, cells(appRows))
	return root.node()
}

func at(x, y, w, h, pad int, s style.ComputedStyle, raw string) scene.Node {
	r := layout.Rect{X: x, Y: y, W: w, H: h}
	in := layout.Rect{X: x + pad, Y: y + pad, W: w - 2*pad, H: h - 2*pad}
	return scene.New(&layout.Box{BorderBox: r, PaddingBox: in, ContentBox: in, Clip: layout.Rect{W: appColumns, H: appRows}}, inked(s), scene.Sanitize(raw))
}

func retext(n scene.Node, raw string) scene.Node {
	m := scene.New(&layout.Box{BorderBox: n.Bounds, PaddingBox: n.Padding, ContentBox: n.Content, Clip: n.Clip}, inked(plain()), scene.Sanitize(raw))
	m.Background, m.Children = n.Background, n.Children
	return m
}

func move(n *scene.Node, dx, dy int) {
	for _, r := range []*layout.Rect{&n.Bounds, &n.Padding, &n.Content} {
		r.X, r.Y = r.X+dx, r.Y+dy
	}
}

func scenery() scene.Node {
	root := app("keys 1")
	pill := filled(blue)
	pill.Radius = style.RadiusFull
	group := at(2, 20, 30, 2, 0, filled(red), "")
	group.Opacity = 0.5
	group.Children = []scene.Node{at(3, 20, 20, 1, 0, filled(color.RGBA{R: 255, A: 100}), "faded text")}
	card := filled(zinc800)
	card.BorderStyle, card.BorderColor, card.Radius = style.BorderSingle, literal(white), style.RadiusLg
	card.Shadows = []style.Shadow{{Y: 4, Blur: 6, Spread: -1, Color: literal(color.RGBA{A: 64})}}
	popover := at(10, 5, 20, 6, 1, card, "popover")
	popover.Position, popover.ZIndex = layout.PositionAbsolute, 10
	root.Children = append(root.Children, at(7, 3, 12, 1, 0, plain(), "中文中文x"), at(40, 1, 6, 1, 0, pill, "pill"), group, popover, at(9, 3, 4, 1, 0, filled(color.RGBA{B: 255, A: 100}), ""))
	return root
}

func column(root *scene.Node) *scene.Node {
	return &root.Children[1].Children[0].Children[0].Children[0]
}

func TestDamagedPaintMatchesFull(t *testing.T) {
	cases := []struct {
		name   string
		look   Look
		change func(*scene.Node)
	}{
		{"text", Composited, func(r *scene.Node) { r.Children[0] = retext(r.Children[0], "keys 22") }},
		{"background", Composited, func(r *scene.Node) { column(r).Background = literal(red) }},
		{"moved popover", Composited, func(r *scene.Node) { move(&r.Children[5], 17, 9) }},
		{"removed node", Composited, func(r *scene.Node) {
			c := column(r)
			c.Children = append(c.Children[:1:1], c.Children[2:]...)
		}},
		{"removed popover", Composited, func(r *scene.Node) { r.Children = append(r.Children[:5], r.Children[6]) }},
		{"tint beside a wide glyph", Composited, func(r *scene.Node) { r.Children[6].Background = literal(color.RGBA{G: 255, A: 100}) }},
		{"wide text", Composited, func(r *scene.Node) { r.Children[2] = retext(r.Children[2], "ab") }},
		{"wide text moves", Composited, func(r *scene.Node) { move(&r.Children[2], 1, 0) }},
		{"group opacity", Composited, func(r *scene.Node) { r.Children[4].Opacity = 0.8 }},
		{"text in a group", Composited, func(r *scene.Node) { r.Children[4].Children[0] = retext(r.Children[4].Children[0], "changed") }},
		{"pill colour", Composited, func(r *scene.Node) { r.Children[3].Background = literal(red) }},
		{"popover below", Composited, func(r *scene.Node) { r.Children[5].ZIndex = -1 }},
		{"shadow grows", Composited, func(r *scene.Node) { r.Children[5].Shadows[0].Blur = 30 }},
		{"look", Glyphs, func(*scene.Node) {}},
		{"root background", Composited, func(r *scene.Node) { r.Background = literal(zinc800) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before, after := scenery(), scenery()
			c.change(&after)
			var p Painter
			buf := buffer.New(appColumns, appRows)
			for i, frame := range []struct {
				root scene.Node
				look Look
			}{{before, Composited}, {after, c.look}, {before, Composited}, {after, c.look}} {
				p.Paint(buf, &frame.root, frame.look)
				want := buffer.New(appColumns, appRows)
				reference(want, frame.root, frame.look)
				same(t, i, buf, want)
			}
		})
	}
}

func TestOneTextRepaintsItsTile(t *testing.T) {
	before, after := scenery(), scenery()
	after.Children[0] = retext(after.Children[0], "keys 2")
	var p Painter
	buf := buffer.New(appColumns, appRows)
	p.Paint(buf, &before, Composited)
	p.Paint(buf, &after, Composited)
	if want := []layout.Rect{{W: konst.DamageColumns, H: 1}}; !slices.Equal(p.spans, want) {
		t.Errorf("repainted %v, want only the text's tile %v", p.spans, want)
	}
	p.Paint(buf, &after, Composited)
	if len(p.spans) != 0 {
		t.Errorf("an unchanged frame repainted %v", p.spans)
	}
}

func TestDamagedPaintAfterResize(t *testing.T) {
	root := scenery()
	var p Painter
	buf := buffer.New(appColumns, appRows)
	p.Paint(buf, &root, Composited)
	for i, size := range [][2]int{{appColumns - 17, appRows - 5}, {appColumns, appRows}} {
		buf.Resize(size[0], size[1])
		p.Paint(buf, &root, Composited)
		want := buffer.New(size[0], size[1])
		reference(want, root, Composited)
		same(t, i, buf, want)
	}
}

func reference(buf *buffer.Buffer, root scene.Node, look Look) {
	whole := layout.Rect{W: buf.Width(), H: buf.Height()}
	if bg := root.Background; bg.Kind == color.Literal && bg.RGBA.A > 0 {
		buf.Fill(buffer.Rect(whole), buffer.Cell{Grapheme: " ", Bg: bg})
	}
	target := buf
	var p Painter
	new(scene.Walker).Walk(&root, func(n *scene.Node) { p.draw(target, n, look, n.Clip) }, func(n *scene.Node, inside func()) {
		switch {
		case n.Opacity <= 0:
		case n.Opacity >= 1:
			inside()
		default:
			under := target
			target = buffer.New(buf.Width(), buf.Height())
			target.Fill(buffer.Rect(whole), buffer.Cell{Grapheme: " ", Bg: color.Color{Kind: color.Literal}})
			inside()
			fade(under, target, n.Opacity, whole)
			target = under
		}
	})
}

func same(t *testing.T, frame int, got, want *buffer.Buffer) {
	t.Helper()
	wrong := 0
	for y := range want.Height() {
		for x, c := range want.Row(y) {
			if g := got.At(x, y); g != c {
				if wrong == 0 {
					t.Errorf("frame %d, cell %d,%d: %+v, want %+v", frame, x, y, g, c)
				}
				wrong++
			}
		}
	}
	if wrong > 0 {
		t.Errorf("frame %d: %d cells differ from a full paint", frame, wrong)
	}
}

func TestPainterSeesANestedScroller(t *testing.T) {
	plainTree, scrolling := app("keys 1"), app("keys 1")
	scrolling.Children[1].Children[0].Scroll = true
	buf := buffer.New(appColumns, appRows)
	var p Painter
	for i, frame := range []struct {
		root *scene.Node
		want bool
	}{{&plainTree, false}, {&scrolling, true}, {&plainTree, false}} {
		p.Paint(buf, frame.root, Composited)
		if got := p.Scrolls(); got != frame.want {
			t.Errorf("frame %d: Scrolls() = %v, want %v", i, got, frame.want)
		}
	}
}

func TestBoxSignatureSeesEveryPaintedField(t *testing.T) {
	ink := func(r uint8) color.Color { return literal(color.RGBA{R: r, A: 255}) }
	base := scene.Node{Bounds: layout.Rect{X: 1, Y: 2, W: 3, H: 4}, Padding: layout.Rect{X: 1, Y: 2, W: 3, H: 4}, Content: layout.Rect{X: 1, Y: 2, W: 3, H: 4}, Clip: layout.Rect{W: 9, H: 9}, Background: ink(1), Foreground: ink(2)}
	base.Gradient = style.Gradient{GradientLine: style.GradientLine{Kind: style.GradientLinear, Angle: 0.5}, From: style.GradientStop{Color: ink(3)}, Via: style.GradientStop{Color: ink(4), Position: 0.5}, To: style.GradientStop{Color: ink(5), Position: 1}}
	base.Border = scene.Border{Style: style.BorderSingle, Radius: 1, Top: true, Color: ink(6)}
	changes := map[string]func(n *scene.Node){
		"bounds x":        func(n *scene.Node) { n.Bounds.X++ },
		"bounds h":        func(n *scene.Node) { n.Bounds.H++ },
		"padding w":       func(n *scene.Node) { n.Padding.W++ },
		"content y":       func(n *scene.Node) { n.Content.Y++ },
		"clip w":          func(n *scene.Node) { n.Clip.W++ },
		"background":      func(n *scene.Node) { n.Background.RGBA.G++ },
		"background kind": func(n *scene.Node) { n.Background.Kind = color.Current },
		"foreground":      func(n *scene.Node) { n.Foreground.RGBA.B++ },
		"gradient kind":   func(n *scene.Node) { n.Gradient.Kind = 0 },
		"gradient angle":  func(n *scene.Node) { n.Gradient.Angle = 0.25 },
		"gradient dir":    func(n *scene.Node) { n.Gradient.Direction++ },
		"gradient space":  func(n *scene.Node) { n.Gradient.Space++ },
		"from colour":     func(n *scene.Node) { n.Gradient.From.Color.RGBA.A-- },
		"from at":         func(n *scene.Node) { n.Gradient.From.Position = 0.1 },
		"via colour":      func(n *scene.Node) { n.Gradient.Via.Color.RGBA.R++ },
		"via at":          func(n *scene.Node) { n.Gradient.Via.Position = 0.6 },
		"to colour":       func(n *scene.Node) { n.Gradient.To.Color.RGBA.G++ },
		"to at":           func(n *scene.Node) { n.Gradient.To.Position = 0.9 },
		"has via":         func(n *scene.Node) { n.Gradient.HasVia = true },
		"border style":    func(n *scene.Node) { n.Border.Style = style.BorderDouble },
		"border radius":   func(n *scene.Node) { n.Border.Radius++ },
		"border top":      func(n *scene.Node) { n.Border.Top = false },
		"border right":    func(n *scene.Node) { n.Border.Right = true },
		"border bottom":   func(n *scene.Node) { n.Border.Bottom = true },
		"border left":     func(n *scene.Node) { n.Border.Left = true },
		"border colour":   func(n *scene.Node) { n.Border.Color.RGBA.R++ },
		"bold":            func(n *scene.Node) { n.Bold = true },
		"italic":          func(n *scene.Node) { n.Italic = true },
		"underline":       func(n *scene.Node) { n.Underline = true },
		"strikethrough":   func(n *scene.Node) { n.Strikethrough = true },
		"truncate":        func(n *scene.Node) { n.Truncate = true },
		"align":           func(n *scene.Node) { n.TextAlign = style.TextRight },
		"shadow":          func(n *scene.Node) { n.Shadows = []style.Shadow{{Blur: 1}} },
		"inset shadow":    func(n *scene.Node) { n.InsetShadows = []style.Shadow{{Blur: 1}} },
	}
	var p Painter
	p.Paint(buffer.New(4, 4), &base, Composited)
	want := p.box(&base)
	for name, change := range changes {
		n := base
		change(&n)
		if p.box(&n) == want {
			t.Errorf("%s: the box signature did not change, so the box would not be repainted", name)
		}
	}
	if again := base; p.box(&again) != want {
		t.Error("the same box gave two signatures")
	}
}

func count(n scene.Node) int {
	total := 1
	for _, c := range n.Children {
		total += count(c)
	}
	return total
}

func BenchmarkPaint1000(b *testing.B) {
	frames := [2]scene.Node{app("keys 1"), app("keys 2")}
	if got := count(frames[0]); got != appNodes {
		b.Fatalf("tree has %d nodes, want %d", got, appNodes)
	}
	blank := buffer.Cell{Grapheme: " "}
	whole := buffer.Rect{W: appColumns, H: appRows}
	b.Run("full", func(b *testing.B) {
		buf := buffer.New(appColumns, appRows)
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			buf.Fill(whole, blank)
			Paint(buf, frames[i%2], Composited)
		}
	})
	b.Run("theme", func(b *testing.B) {
		themes := [2]scene.Node{app("keys 1"), app("keys 1")}
		themes[1].Background = literal(zinc800)
		buf := buffer.New(appColumns, appRows)
		var p Painter
		p.Paint(buf, &themes[0], Composited)
		b.ReportAllocs()
		for i := 1; b.Loop(); i++ {
			p.Paint(buf, &themes[i%2], Composited)
		}
	})
	b.Run("text", func(b *testing.B) {
		buf := buffer.New(appColumns, appRows)
		var p Painter
		p.Paint(buf, &frames[0], Composited)
		b.ReportAllocs()
		for i := 1; b.Loop(); i++ {
			p.Paint(buf, &frames[i%2], Composited)
		}
	})
}
