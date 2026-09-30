package paint

import (
	"hash/maphash"
	"math"
	"math/bits"
	"slices"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/paint"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
)

type Painter struct {
	buf                    *buffer.Buffer
	width, height, columns int
	seed                   maphash.Seed
	nodes                  []*scene.Node
	shapes, drawn          []shape
	index                  map[*scene.Node]int32
	ops                    []op
	groups                 []uint64
	tiles, last            []uint64
	wide, wasWide          []bool
	spans                  []layout.Rect
	layers                 []*buffer.Buffer
}

type shape struct {
	children               int
	position               layout.Position
	z                      int
	hidden, faded, scrolls bool
}

type step uint8

const (
	drawStep step = iota
	openStep
	closeStep
)

type op struct {
	step step
	node int32
	area layout.Rect
}

type face struct {
	bounds, padding, content, clip                   layout.Rect
	background, foreground                           color.Color
	gradient                                         style.Gradient
	border                                           scene.Border
	bold, italic, underline, strikethrough, truncate bool
	align                                            style.TextAlign
	shadows, insets                                  int
}

func Paint(buf *buffer.Buffer, root scene.Node, look Look) {
	var p Painter
	p.Paint(buf, &root, look)
}

func (p *Painter) Paint(buf *buffer.Buffer, root *scene.Node, look Look) {
	p.order(root)
	p.sign(buf, root, look)
	p.damage(buf)
	for _, span := range p.spans {
		p.repaint(buf, root, look, span)
	}
	p.buf, p.width, p.height = buf, buf.Width(), buf.Height()
	p.tiles, p.last = p.last, p.tiles
	p.wide, p.wasWide = p.wasWide, p.wide
}

func (p *Painter) order(root *scene.Node) {
	p.nodes, p.shapes = p.nodes[:0], p.shapes[:0]
	p.preorder(root)
	if slices.Equal(p.shapes, p.drawn) {
		return
	}
	p.shapes, p.drawn = p.drawn, p.shapes
	if p.index == nil {
		p.index = make(map[*scene.Node]int32, len(p.nodes))
	}
	clear(p.index)
	for i, n := range p.nodes {
		p.index[n] = int32(i)
	}
	p.ops = p.ops[:0]
	scene.Walk(root, func(n *scene.Node) { p.ops = append(p.ops, op{step: drawStep, node: p.index[n]}) }, func(n *scene.Node, inside func()) {
		switch {
		case n.Opacity <= 0:
		case n.Opacity >= 1:
			inside()
		default:
			p.ops = append(p.ops, op{step: openStep, node: p.index[n]})
			inside()
			p.ops = append(p.ops, op{step: closeStep, node: p.index[n]})
		}
	})
}

func (p *Painter) preorder(n *scene.Node) {
	p.nodes = append(p.nodes, n)
	p.shapes = append(p.shapes, shape{len(n.Children), n.Position, n.ZIndex, n.Opacity <= 0, n.Opacity < 1, n.Scroll})
	for i := range n.Children {
		p.preorder(&n.Children[i])
	}
}

func (p *Painter) sign(buf *buffer.Buffer, root *scene.Node, look Look) {
	if p.seed == (maphash.Seed{}) {
		p.seed = maphash.MakeSeed()
	}
	width, height := buf.Width(), buf.Height()
	p.columns = (width + konst.DamageColumns - 1) / konst.DamageColumns
	p.tiles = append(p.tiles[:0], make([]uint64, p.columns*height)...)
	p.wide = append(p.wide[:0], make([]bool, height)...)
	canvas := mix(maphash.Comparable(p.seed, root.Background), uint64(look))
	for i := range p.tiles {
		p.tiles[i] = canvas
	}
	screen := layout.Rect{W: width, H: height}
	p.groups = append(p.groups[:0], canvas)
	opened := uint64(0)
	for i := range p.ops {
		o := &p.ops[i]
		n := p.nodes[o.node]
		switch o.step {
		case openStep:
			opened++
			p.groups = append(p.groups, mix(mix(p.groups[len(p.groups)-1], math.Float64bits(n.Opacity)), opened))
			continue
		case closeStep:
			p.groups = p.groups[:len(p.groups)-1]
			continue
		}
		o.area = overlap(overlap(extent(n), n.Clip), screen)
		if o.area.W == 0 || o.area.H == 0 {
			continue
		}
		box := mix(p.box(n), p.groups[len(p.groups)-1])
		p.mark(o.area, box, false)
		ink, lines, widest, wide := box, n.Lines(), 0, false
		lines = lines[:min(len(lines), n.Content.H)]
		for _, line := range lines {
			ink = mix(ink, maphash.String(p.seed, line))
			widest = max(widest, len(line))
			for i := range len(line) {
				wide = wide || line[i] >= utf8.RuneSelf
				if line[i] == '\t' {
					widest = n.Content.W
				}
			}
		}
		if n.TextAlign == style.TextCenter || n.TextAlign == style.TextRight {
			widest = n.Content.W
		}
		c := n.Content
		p.mark(overlap(overlap(layout.Rect{X: c.X, Y: c.Y, W: min(widest, c.W), H: len(lines)}, n.Clip), screen), ink, wide)
	}
}

func (p *Painter) box(n *scene.Node) uint64 {
	h := maphash.Comparable(p.seed, face{
		n.Bounds, n.Padding, n.Content, n.Clip, n.Background, n.Foreground, n.Gradient, n.Border,
		n.Bold, n.Italic, n.Underline, n.Strikethrough, n.Truncate, n.TextAlign, len(n.Shadows), len(n.InsetShadows),
	})
	for _, s := range n.Shadows {
		h = mix(h, maphash.Comparable(p.seed, s))
	}
	for _, s := range n.InsetShadows {
		h = mix(h, maphash.Comparable(p.seed, s))
	}
	return h
}

func (p *Painter) mark(area layout.Rect, h uint64, wide bool) {
	if area.W == 0 {
		return
	}
	for y := area.Y; y < area.Y+area.H; y++ {
		p.wide[y] = p.wide[y] || wide
		row := p.tiles[y*p.columns : (y+1)*p.columns]
		for t := area.X / konst.DamageColumns; t < (area.X+area.W+konst.DamageColumns-1)/konst.DamageColumns; t++ {
			row[t] = mix(row[t], h)
		}
	}
}

func mix(h, v uint64) uint64 { return bits.RotateLeft64((h^v)*konst.HashPrime, konst.HashRotate) }

func extent(n *scene.Node) layout.Rect {
	r := outline(n)
	x0, y0, x1, y1 := r.X, r.Y, r.X+r.W, r.Y+r.H
	for _, s := range n.Shadows {
		left, top, right, bottom := reach(s)
		x0, y0, x1, y1 = min(x0, r.X-left), min(y0, r.Y-top), max(x1, r.X+r.W+right), max(y1, r.Y+r.H+bottom)
	}
	return layout.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

func overlap(a, b layout.Rect) layout.Rect {
	x0, y0 := max(a.X, b.X), max(a.Y, b.Y)
	return layout.Rect{X: x0, Y: y0, W: max(min(a.X+a.W, b.X+b.W)-x0, 0), H: max(min(a.Y+a.H, b.Y+b.H)-y0, 0)}
}

func (p *Painter) damage(buf *buffer.Buffer) {
	width := buf.Width()
	full := buf != p.buf || width != p.width || buf.Height() != p.height
	p.spans = p.spans[:0]
	for y := range buf.Height() {
		dirty := func(t int) bool { return full || p.tiles[y*p.columns+t] != p.last[y*p.columns+t] }
		for t := 0; t < p.columns; t++ {
			if !dirty(t) {
				continue
			}
			first := t
			for t < p.columns && dirty(t) {
				t++
			}
			run := layout.Rect{X: first * konst.DamageColumns, Y: y, W: min(t*konst.DamageColumns, width) - first*konst.DamageColumns, H: 1}
			if full || p.wide[y] || p.wasWide[y] {
				run.X, run.W, t = 0, width, p.columns
			}
			p.extend(run)
		}
	}
}

func (p *Painter) extend(run layout.Rect) {
	for i := range p.spans {
		if s := &p.spans[i]; s.X == run.X && s.W == run.W && s.Y+s.H == run.Y {
			s.H++
			return
		}
	}
	p.spans = append(p.spans, run)
}

func (p *Painter) repaint(buf *buffer.Buffer, root *scene.Node, look Look, span layout.Rect) {
	canvas := buffer.Cell{Grapheme: " "}
	if bg := root.Background; bg.Kind == color.Literal && bg.RGBA.A > 0 {
		canvas.Bg = bg
	}
	wipe(buf, span, canvas)
	depth := 0
	target := func(d int) *buffer.Buffer {
		if d == 0 {
			return buf
		}
		return p.layers[d-1]
	}
	for _, o := range p.ops {
		n := p.nodes[o.node]
		switch o.step {
		case openStep:
			if depth == len(p.layers) {
				p.layers = append(p.layers, new(buffer.Buffer))
			}
			layer := p.layers[depth]
			if layer.Width() != buf.Width() || layer.Height() != buf.Height() {
				layer.Resize(buf.Width(), buf.Height())
			}
			wipe(layer, span, buffer.Cell{Grapheme: " ", Bg: color.Color{Kind: color.Literal}})
			depth++
		case closeStep:
			depth--
			fade(target(depth), p.layers[depth], n.Opacity, span)
		case drawStep:
			if touched := overlap(o.area, span); touched.W > 0 && touched.H > 0 {
				draw(target(depth), n, look, overlap(n.Clip, span))
			}
		}
	}
}

func wipe(buf *buffer.Buffer, span layout.Rect, c buffer.Cell) {
	for y := span.Y; y < span.Y+span.H; y++ {
		row := buf.Row(y)[span.X : span.X+span.W]
		for x := range row {
			row[x] = c
		}
	}
}
