package paint

import (
	"hash/maphash"
	"math"
	"math/bits"
	"unicode/utf8"

	konst "github.com/pehcastro/twind/internal/konst/paint"
	"github.com/pehcastro/twind/twi/buffer"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/layout"
	"github.com/pehcastro/twind/twi/scene"
	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/terminal"
	"github.com/pehcastro/twind/twi/text"
)

type Painter struct {
	Widths                 text.Widths
	Profile                color.Profile
	Covers                 func(cluster string) bool
	Identity               terminal.Identity
	buf                    *buffer.Buffer
	width, height, columns int
	seed                   maphash.Seed
	nodes                  []*scene.Node
	areas                  []layout.Rect
	shapes                 []shape
	index                  map[*scene.Node]int32
	reshaped               bool
	ops                    []op
	groups                 []uint64
	tiles, last            []uint64
	wide, wasWide          []bool
	spans                  []layout.Rect
	layers                 []*buffer.Buffer
	walker                 scene.Walker
	scrolls                bool
}

type shape struct {
	z, top                 int
	children               int32
	position               layout.Position
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
}

type face struct {
	bounds, padding, content, clip                       layout.Rect
	angle, fromAt, viaAt, toAt                           uint64
	radius, shadows, insets                              int
	background, foreground, from, via, to, border, flags uint64
}

func Paint(buf *buffer.Buffer, root scene.Node, look Look) {
	var p Painter
	p.Paint(buf, &root, look)
}

func (p *Painter) Paint(buf *buffer.Buffer, root *scene.Node, look Look) {
	p.order(root, layout.Rect{W: buf.Width(), H: buf.Height()})
	p.sign(buf, root, look)
	p.damage(buf)
	for _, span := range p.spans {
		p.repaint(buf, root, look, span)
	}
	p.buf, p.width, p.height = buf, buf.Width(), buf.Height()
	p.tiles, p.last = p.last, p.tiles
	p.wide, p.wasWide = p.wasWide, p.wide
}

func (p *Painter) Repainted() []layout.Rect { return p.spans }

func (p *Painter) Scrolls() bool { return p.scrolls }

func (p *Painter) order(root *scene.Node, screen layout.Rect) {
	if p.nodes == nil {
		total := size(root)
		p.nodes, p.areas, p.shapes, p.ops = make([]*scene.Node, 0, total), make([]layout.Rect, 0, total), make([]shape, 0, total), make([]op, 0, total)
	}
	p.nodes, p.areas, p.reshaped, p.scrolls = p.nodes[:0], p.areas[:0], false, false
	p.preorder(root, screen)
	if !p.reshaped && len(p.nodes) == len(p.shapes) {
		return
	}
	p.shapes, p.ops = p.shapes[:len(p.nodes)], p.ops[:0]
	next, indexed := int32(0), false
	at := func(n *scene.Node) int32 {
		if int(next) < len(p.nodes) && p.nodes[next] == n {
			return next
		}
		if !indexed {
			if p.index == nil {
				p.index = make(map[*scene.Node]int32, len(p.nodes))
			}
			clear(p.index)
			for i, n := range p.nodes {
				p.index[n] = int32(i)
			}
			indexed = true
		}
		next = p.index[n]
		return next
	}
	p.walker.Walk(root, func(n *scene.Node) {
		p.ops = append(p.ops, op{step: drawStep, node: at(n)})
		next++
	}, func(n *scene.Node, inside func()) {
		switch {
		case n.Opacity <= 0:
		case n.Opacity >= 1:
			inside()
		default:
			i := at(n)
			p.ops = append(p.ops, op{step: openStep, node: i})
			inside()
			p.ops = append(p.ops, op{step: closeStep, node: i})
		}
	})
}

func size(n *scene.Node) int {
	total := 1
	for i := range n.Children {
		total += size(&n.Children[i])
	}
	return total
}

func (p *Painter) preorder(n *scene.Node, screen layout.Rect) {
	s := shape{n.ZIndex, n.TopLayer, int32(len(n.Children)), n.Position, n.Opacity <= 0, n.Opacity < 1, n.Scroll}
	switch i := len(p.nodes); {
	case i == len(p.shapes):
		p.shapes, p.reshaped = append(p.shapes, s), true
	case p.shapes[i] != s:
		p.shapes[i], p.reshaped = s, true
	}
	p.nodes, p.areas, p.scrolls = append(p.nodes, n), append(p.areas, overlap(overlap(p.extent(n), n.Clip), screen)), p.scrolls || n.Scroll
	for i := range n.Children {
		p.preorder(&n.Children[i], screen)
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
	canvas := mix(mix(mix(maphash.Comparable(p.seed, root.Background), uint64(look)), uint64(p.Profile)), maphash.Comparable(p.seed, p.Widths))
	for i := range p.tiles {
		p.tiles[i] = canvas
	}
	screen := layout.Rect{W: width, H: height}
	p.groups = append(p.groups[:0], canvas)
	opened := uint64(0)
	for _, o := range p.ops {
		switch o.step {
		case openStep:
			opened++
			p.groups = append(p.groups, mix(mix(p.groups[len(p.groups)-1], math.Float64bits(p.nodes[o.node].Opacity)), opened))
			continue
		case closeStep:
			p.groups = p.groups[:len(p.groups)-1]
			continue
		}
		area := p.areas[o.node]
		if area.W == 0 || area.H == 0 {
			continue
		}
		n := p.nodes[o.node]
		box := mix(p.box(n), p.groups[len(p.groups)-1])
		p.mark(area, box, false)
		ink, lines, widest, wide := box, n.Lines(p.Widths), 0, false
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
	g, b := &n.Gradient, &n.Border
	h := maphash.Comparable(p.seed, face{
		n.Bounds, n.Padding, n.Content, n.Clip,
		math.Float64bits(g.Angle), math.Float64bits(g.From.Position), math.Float64bits(g.Via.Position), math.Float64bits(g.To.Position),
		int(b.Radius), len(n.Shadows), len(n.InsetShadows),
		word(n.Background), word(n.Foreground), word(g.From.Color), word(g.Via.Color), word(g.To.Color), word(b.Color),
		uint64(g.Kind) | uint64(g.Direction)<<8 | uint64(g.Space)<<16 | uint64(b.Style)<<24 | uint64(n.TextAlign)<<32 |
			bit(g.HasVia, 40) | bit(b.Top, 41) | bit(b.Right, 42) | bit(b.Bottom, 43) | bit(b.Left, 44) |
			bit(n.Bold, 45) | bit(n.Italic, 46) | bit(n.Underline, 47) | bit(n.Strikethrough, 48) | bit(n.Truncate, 49),
	})
	for _, s := range n.Shadows {
		h = mix(h, maphash.Comparable(p.seed, s))
	}
	for _, s := range n.InsetShadows {
		h = mix(h, maphash.Comparable(p.seed, s))
	}
	return h
}

func word(c color.Color) uint64 {
	return uint64(c.Kind)<<32 | uint64(c.RGBA.R)<<24 | uint64(c.RGBA.G)<<16 | uint64(c.RGBA.B)<<8 | uint64(c.RGBA.A)
}

func bit(b bool, at uint) uint64 {
	if b {
		return 1 << at
	}
	return 0
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

func (p *Painter) extent(n *scene.Node) layout.Rect {
	r := n.Bounds
	x0, y0, x1, y1 := r.X, r.Y, r.X+r.W, r.Y+r.H
	for _, s := range n.Shadows {
		c := p.cast(r, s)
		x0, y0, x1, y1 = min(x0, c.X), min(y0, c.Y), max(x1, c.X+c.W), max(y1, c.Y+c.H)
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
			if touched := overlap(p.areas[o.node], span); touched.W > 0 && touched.H > 0 {
				p.draw(target(depth), n, look, overlap(n.Clip, span))
			}
		}
	}
	if !p.paletted() {
		return
	}
	for y := span.Y; y < span.Y+span.H; y++ {
		row := buf.Row(y)
		for x := span.X; x < span.X+span.W; x++ {
			if c := &row[x]; c.Grapheme != " " && c.Width != buffer.Continuation {
				c.Fg = apart(c.Fg, c.Bg, p.Profile)
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
