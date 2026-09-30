package paint

import (
	"fmt"
	"image"
	"math"
	"strings"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/paint"
	rasterkonst "github.com/twind-dev/twind/internal/konst/raster"
	stylekonst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/text"
)

type Look uint8

const (
	Composited Look = iota
	Plain
	Glyphs
)

func draw(buf *buffer.Buffer, n *scene.Node, look Look, clip layout.Rect, widths text.Widths) {
	shadows, insets := n.Shadows, n.InsetShadows
	if look != Composited {
		shadows, insets = nil, nil
	}
	body := outline(n)
	for i := len(shadows) - 1; i >= 0; i-- {
		left, top, right, bottom := reach(shadows[i])
		cast := layout.Rect{X: body.X - left, Y: body.Y - top, W: body.W + left + right, H: body.H + top + bottom}
		shadow(buf, clip, cast, body, shadows[i], false)
	}
	fill := n.Bounds
	if look == Composited && n.Border.Style == style.BorderSingle {
		fill = n.Padding
	}
	bg := n.Background
	filled := bg.Kind == color.Literal && bg.RGBA.A > 0
	if filled {
		f := overlap(fill, clip)
		for y := f.Y; y < f.Y+f.H; y++ {
			for x := f.X; x < f.X+f.W; x++ {
				put(buf, clip, x, y, buffer.Cell{Grapheme: " ", Bg: bg})
			}
		}
	}
	if look != Plain && n.Gradient.Kind == style.GradientLinear {
		gradient(buf, n, fill, look, clip)
	}
	if look == Composited && body != n.Bounds {
		caps := split(konst.PillCaps)
		put(buf, clip, body.X, body.Y, buffer.Cell{Grapheme: caps[0], Fg: bg, Bg: color.Color{Kind: color.Literal}})
		put(buf, clip, body.X+body.W-1, body.Y, buffer.Cell{Grapheme: caps[1], Fg: bg, Bg: color.Color{Kind: color.Literal}})
	}
	for _, s := range insets {
		p := n.Padding
		left, top, right, bottom := reach(s)
		lit := layout.Rect{X: p.X + right, Y: p.Y + bottom, W: p.W - left - right, H: p.H - top - bottom}
		shadow(buf, clip, p, lit, s, true)
	}
	border(buf, n, look, clip)
	lines(buf, n, clip, widths)
}

func outline(n *scene.Node) layout.Rect {
	r, bg := n.Bounds, n.Background
	if n.Border.Radius == style.RadiusFull && r.H == 1 && bg.Kind == color.Literal && bg.RGBA.A > 0 {
		r.X, r.W = r.X-1, r.W+2
	}
	return r
}

func split(set string) (glyphs [4]string) {
	i := 0
	for at, r := range set {
		glyphs[i] = set[at : at+utf8.RuneLen(r)]
		i++
	}
	return glyphs
}

func reach(s style.Shadow) (left, top, right, bottom int) {
	blur := float64(s.Blur) * rasterkonst.SigmaPerBlur
	cells := func(offset style.Pixels, cell float64) int {
		v := (float64(offset+s.Spread) + blur) / cell
		if v > 0 {
			return max(1, int(math.Round(v)))
		}
		return int(math.Round(v))
	}
	return cells(-s.X, stylekonst.NominalCellX), cells(-s.Y, stylekonst.NominalCellY), cells(s.X, stylekonst.NominalCellX), cells(s.Y, stylekonst.NominalCellY)
}

func gradient(buf *buffer.Buffer, n *scene.Node, fill layout.Rect, look Look, clip layout.Rect) {
	w, h := float64(fill.W), float64(2*fill.H)
	op := scene.GradientFill(n.Gradient, raster.Box{Rect: raster.Rect{W: w, H: h}})
	sin, cos := math.Sincos(op.Angle * math.Pi / 180)
	if line := math.Abs(w*sin) + math.Abs(h*cos); line > 1 {
		for i := range op.Stops {
			op.Stops[i].At = (0.5 + op.Stops[i].At*(line-1)) / line
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, fill.W, 2*fill.H))
	new(raster.Raster).Draw(img, []raster.Op{op}, img.Rect)
	for y := range fill.H {
		for x := range fill.W {
			top := color.Color{Kind: color.Literal, RGBA: raster.Mean(img, image.Rect(x, 2*y, x+1, 2*y+1))}
			bottom := color.Color{Kind: color.Literal, RGBA: raster.Mean(img, image.Rect(x, 2*y+1, x+1, 2*y+2))}
			cell := buffer.Cell{Grapheme: konst.UpperHalf, Fg: top, Bg: bottom}
			if top == bottom || look == Glyphs {
				cell = buffer.Cell{Grapheme: " ", Bg: bottom}
			}
			put(buf, clip, fill.X+x, fill.Y+y, cell)
		}
	}
}

func shadow(buf *buffer.Buffer, clip, shaded, lit layout.Rect, s style.Shadow, inset bool) {
	weight := [4]int{1, 1, 1, 1}
	if s.X == 0 && s.Y == 0 && s.Blur == 0 {
		eighths := func(cell float64) int {
			return min(max(int(math.Round(float64(s.Spread)*konst.CellEighths/cell)), 1), konst.RingEighths)
		}
		v, h := eighths(stylekonst.NominalCellY), eighths(stylekonst.NominalCellX)
		weight = [4]int{v, h, v, h}
	}
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
	area := overlap(shaded, clip)
	for y := area.Y; y < area.Y+area.H; y++ {
		for x := area.X; x < area.X+area.W; x++ {
			outX, right, fractionX := band(x, shaded.X, shaded.W, lit.X, lit.W)
			outY, below, fractionY := band(y, shaded.Y, shaded.H, lit.Y, lit.H)
			if outX == outY {
				continue
			}
			if outX && !fractionX || outY && !fractionY {
				put(buf, clip, x, y, buffer.Cell{Grapheme: " ", Bg: s.Color})
				continue
			}
			side := 3
			switch {
			case outY && below:
				side = 2
			case outY:
				side = 0
			case right:
				side = 1
			}
			if !visible(buf, clip, x, y) {
				continue
			}
			if cell, ok := edge(buf.At(x, y), side, weight[side], s.Color); ok {
				buf.Set(x, y, cell)
			}
		}
	}
}

func edge(dst buffer.Cell, side, weight int, c color.Color) (buffer.Cell, bool) {
	was, had := edgeOf(dst)
	ink := over(c, dst.Bg)
	switch {
	case dst.Grapheme == " ":
	case had == 0 || was%2 != side%2:
		return dst, false
	case side == was:
		return block(side, max(weight, had), over(c, dst.Fg), dst.Bg), true
	case weight < had || weight == had && ink == dst.Fg:
		return dst, false
	}
	return block(side, weight, ink, dst.Bg), true
}

func edgeOf(c buffer.Cell) (side, weight int) {
	if len(c.Grapheme) != konst.BlockBytes || c.Attr&^buffer.Inverse != 0 {
		return 0, 0
	}
	inverse := c.Attr == buffer.Inverse
	if at := strings.Index(konst.Hairlines, c.Grapheme); at >= 2*konst.BlockBytes && !inverse {
		return at / konst.BlockBytes, 1
	}
	for side, set := range [2]string{konst.LowerBlocks, konst.LeftBlocks} {
		at := strings.Index(set, c.Grapheme)
		eighths := at/konst.BlockBytes + 1
		switch {
		case at < 0:
		case inverse && eighths >= konst.CellEighths-konst.RingEighths:
			return side + 2, konst.CellEighths - eighths
		case !inverse && eighths <= konst.RingEighths:
			return side, eighths
		}
	}
	return 0, 0
}

func block(side, weight int, fg, bg color.Color) buffer.Cell {
	set, eighths, attr := konst.LowerBlocks, weight, buffer.Attr(0)
	if side%2 == 1 {
		set = konst.LeftBlocks
	}
	switch {
	case side < 2:
	case weight == 1 || bg.Kind != color.Literal:
		return buffer.Cell{Grapheme: konst.Hairlines[side*konst.BlockBytes : (side+1)*konst.BlockBytes], Fg: fg, Bg: bg}
	default:
		eighths, attr = konst.CellEighths-weight, buffer.Inverse
	}
	return buffer.Cell{Grapheme: set[(eighths-1)*konst.BlockBytes : eighths*konst.BlockBytes], Fg: fg, Bg: bg, Attr: attr}
}

func glyphs(b scene.Border, look Look) (edges, corners string) {
	if look == Glyphs {
		return "", ""
	}
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

func border(buf *buffer.Buffer, n *scene.Node, look Look, clip layout.Rect) {
	r, b := n.Bounds, n.Border
	edgeSet, cornerSet := glyphs(b, look)
	if edgeSet == "" || r.W == 0 || r.H == 0 {
		return
	}
	edges := split(edgeSet)
	right, bottom := r.X+r.W-1, r.Y+r.H-1
	glyph := func(x, y int, g string) {
		put(buf, clip, x, y, buffer.Cell{Grapheme: g, Fg: b.Color, Bg: color.Color{Kind: color.Literal}})
	}
	for x := max(r.X, clip.X); x <= min(right, clip.X+clip.W-1); x++ {
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
	for y := max(r.Y, clip.Y); y <= min(bottom, clip.Y+clip.H-1); y++ {
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
	corners := split(cornerSet)
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

func lines(buf *buffer.Buffer, n *scene.Node, clip layout.Rect, widths text.Widths) {
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
	lines := n.Lines(widths)
	for i, line := range lines[:min(len(lines), r.H)] {
		y, x := r.Y+i, r.X
		if y < clip.Y || y >= clip.Y+clip.H {
			continue
		}
		switch n.TextAlign {
		case style.TextLeft, style.TextJustify:
		case style.TextCenter:
			x += max(r.W-widths.Width(line), 0) / 2
		case style.TextRight:
			x += max(r.W-widths.Width(line), 0)
		default:
			panic(fmt.Sprintf("paint: unknown text align %d", n.TextAlign))
		}
		for cluster := range text.Graphemes(line) {
			if cluster == "\t" {
				for stop := min(r.X+((x-r.X)/konst.TabStop+1)*konst.TabStop, end); x < stop; x++ {
					put(buf, clip, x, y, ink)
				}
				continue
			}
			w := widths.Width(cluster)
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
			put(buf, clip, x, y, cell)
			x += w
		}
	}
}
