package paint

import (
	"fmt"
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/paint"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/text"
)

type Look uint8

const (
	Composited Look = iota
	Plain
)

func Paint(buf *buffer.Buffer, root scene.Node, look Look) {
	stack(&root).paint(buf, look)
}

func draw(buf *buffer.Buffer, n *scene.Node, look Look) {
	shadows, insets := n.Shadows, n.InsetShadows
	if look == Plain {
		shadows, insets = nil, nil
	}
	for _, s := range shadows {
		r := n.Bounds
		cast := layout.Rect{X: r.X + s.X - s.Spread, Y: r.Y + s.Y - s.Spread, W: r.W + 2*s.Spread, H: r.H + 2*s.Spread}
		shadow(buf, n.Clip, cast, r, s.Color, false)
	}
	fill := n.Bounds
	if look == Composited && n.Border.Style == style.BorderSingle {
		fill = n.Padding
	}
	if bg := n.Background; bg.Kind == color.Literal && bg.RGBA.A > 0 {
		for y := fill.Y; y < fill.Y+fill.H; y++ {
			for x := fill.X; x < fill.X+fill.W; x++ {
				put(buf, n.Clip, x, y, buffer.Cell{Grapheme: " ", Bg: bg})
			}
		}
	}
	for _, s := range insets {
		p := n.Padding
		lit := layout.Rect{X: p.X + s.X + s.Spread, Y: p.Y + s.Y + s.Spread, W: p.W - 2*s.Spread, H: p.H - 2*s.Spread}
		shadow(buf, n.Clip, p, lit, s.Color, true)
	}
	border(buf, n, look)
	lines(buf, n)
}

func shadow(buf *buffer.Buffer, clip, shaded, lit layout.Rect, c color.Color, inset bool) {
	hairlines := strings.Split(konst.Hairlines, "")
	band := func(v, lo, size, litLo, litSize int) (out, after, fraction bool) {
		after = v >= litLo+litSize
		out = after || v < litLo
		far, next := lo, litLo-1
		if after {
			far, next = lo+size-1, litLo+litSize
		}
		if inset {
			return out, !after, v == next
		}
		return out, after, v == far
	}
	for y := shaded.Y; y < shaded.Y+shaded.H; y++ {
		for x := shaded.X; x < shaded.X+shaded.W; x++ {
			outX, right, fractionX := band(x, shaded.X, shaded.W, lit.X, lit.W)
			outY, below, fractionY := band(y, shaded.Y, shaded.H, lit.Y, lit.H)
			if outX == outY {
				continue
			}
			if outX && !fractionX || outY && !fractionY {
				put(buf, clip, x, y, buffer.Cell{Grapheme: " ", Bg: c})
				continue
			}
			glyph := hairlines[3]
			switch {
			case outY && below:
				glyph = hairlines[2]
			case outY:
				glyph = hairlines[0]
			case right:
				glyph = hairlines[1]
			}
			if !visible(buf, clip, x, y) {
				continue
			}
			dst := buf.At(x, y)
			base := dst.Bg
			switch dst.Grapheme {
			case " ":
			case glyph:
				base = dst.Fg
			default:
				continue
			}
			put(buf, clip, x, y, buffer.Cell{Grapheme: glyph, Fg: over(c, base), Bg: color.Color{Kind: color.Literal}})
		}
	}
}

func glyphs(b scene.Border, look Look) (edges, corners string) {
	round := konst.SquareCorners
	if b.Radius != style.RadiusNone {
		round = konst.RoundedCorners
	}
	switch b.Style {
	case style.BorderNone:
		return "", ""
	case style.BorderSingle:
		if look == Plain {
			return konst.SingleLines, round
		}
		return konst.Hairlines, ""
	case style.BorderDashed:
		return konst.DashedLines, round
	case style.BorderDotted:
		return konst.DottedLines, round
	case style.BorderDouble:
		return konst.DoubleLines, konst.DoubleCorners
	}
	panic(fmt.Sprintf("paint: unknown border style %d", b.Style))
}

func border(buf *buffer.Buffer, n *scene.Node, look Look) {
	r, b := n.Bounds, n.Border
	edgeSet, cornerSet := glyphs(b, look)
	if edgeSet == "" || r.W == 0 || r.H == 0 {
		return
	}
	edges := strings.Split(edgeSet, "")
	right, bottom := r.X+r.W-1, r.Y+r.H-1
	glyph := func(x, y int, g string) {
		put(buf, n.Clip, x, y, buffer.Cell{Grapheme: g, Fg: b.Color, Bg: color.Color{Kind: color.Literal}})
	}
	for x := r.X; x <= right; x++ {
		if x == r.X && b.Left || x == right && b.Right {
			continue
		}
		if b.Top {
			glyph(x, r.Y, edges[0])
		}
		if b.Bottom {
			glyph(x, bottom, edges[2])
		}
	}
	for y := r.Y; y <= bottom; y++ {
		if y == r.Y && b.Top || y == bottom && b.Bottom {
			continue
		}
		if b.Left {
			glyph(r.X, y, edges[3])
		}
		if b.Right {
			glyph(right, y, edges[1])
		}
	}
	if cornerSet == "" {
		return
	}
	corners := strings.Split(cornerSet, "")
	if b.Top && b.Left {
		glyph(r.X, r.Y, corners[0])
	}
	if b.Top && b.Right {
		glyph(right, r.Y, corners[1])
	}
	if b.Bottom && b.Left {
		glyph(r.X, bottom, corners[2])
	}
	if b.Bottom && b.Right {
		glyph(right, bottom, corners[3])
	}
}

func lines(buf *buffer.Buffer, n *scene.Node) {
	var attr buffer.Attr
	if n.Bold {
		attr |= buffer.Bold
	}
	if n.Italic {
		attr |= buffer.Italic
	}
	if n.Underline {
		attr |= buffer.Underline
	}
	if n.Strikethrough {
		attr |= buffer.Strikethrough
	}
	ink := buffer.Cell{Grapheme: " ", Fg: n.Foreground, Bg: color.Color{Kind: color.Literal}, Attr: attr}
	r := n.Content
	end := r.X + r.W
	for i, line := range n.Lines()[:min(len(n.Lines()), r.H)] {
		y, x := r.Y+i, r.X
		for cluster := range text.Graphemes(line) {
			if cluster == "\t" {
				for stop := min(r.X+((x-r.X)/konst.TabStop+1)*konst.TabStop, end); x < stop; x++ {
					put(buf, n.Clip, x, y, ink)
				}
				continue
			}
			w := text.Width(cluster)
			if w == 0 {
				continue
			}
			if x+w > end {
				break
			}
			cell := ink
			cell.Grapheme = cluster
			if w > 1 {
				cell.Width = buffer.Wide
			}
			put(buf, n.Clip, x, y, cell)
			x += w
		}
	}
}
