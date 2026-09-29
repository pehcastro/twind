package paint

import (
	"fmt"
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/paint"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/text"
)

func Paint(buf *buffer.Buffer, root scene.Node) {
	stack(&root).paint(buf)
}

func draw(buf *buffer.Buffer, n *scene.Node) {
	if bg := n.Background; bg.Kind == color.Literal && bg.RGBA.A > 0 {
		r := n.Bounds
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				put(buf, n.Clip, x, y, buffer.Cell{Grapheme: " ", Bg: bg})
			}
		}
	}
	border(buf, n)
	lines(buf, n)
}

func glyphs(b scene.Border) (edges, corners string) {
	round := konst.SquareCorners
	if b.Radius != style.RadiusNone {
		round = konst.RoundedCorners
	}
	switch b.Style {
	case style.BorderNone:
		return "", ""
	case style.BorderSingle:
		return konst.SingleLines, round
	case style.BorderDashed:
		return konst.DashedLines, round
	case style.BorderDotted:
		return konst.DottedLines, round
	case style.BorderDouble:
		return konst.DoubleLines, konst.DoubleCorners
	}
	panic(fmt.Sprintf("paint: unknown border style %d", b.Style))
}

func border(buf *buffer.Buffer, n *scene.Node) {
	r, b := n.Bounds, n.Border
	edgeSet, cornerSet := glyphs(b)
	if edgeSet == "" || r.W == 0 || r.H == 0 {
		return
	}
	edges, corners := strings.Split(edgeSet, ""), strings.Split(cornerSet, "")
	right, bottom := r.X+r.W-1, r.Y+r.H-1
	glyph := func(x, y int, g string) {
		put(buf, n.Clip, x, y, buffer.Cell{Grapheme: g, Fg: b.Color, Bg: color.Color{Kind: color.Literal}})
	}
	for x := r.X; x <= right; x++ {
		if b.Top {
			glyph(x, r.Y, edges[0])
		}
		if b.Bottom {
			glyph(x, bottom, edges[0])
		}
	}
	for y := r.Y; y <= bottom; y++ {
		if b.Left {
			glyph(r.X, y, edges[1])
		}
		if b.Right {
			glyph(right, y, edges[1])
		}
	}
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
