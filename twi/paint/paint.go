package paint

import (
	"fmt"
	"image"
	"iter"
	"math"
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
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/text"
	"github.com/twind-dev/twind/twi/theme"
)

type Look uint8

const (
	Composited Look = iota
	Plain
	Glyphs
)

func (p *Painter) draw(buf *buffer.Buffer, n *scene.Node, look Look, clip layout.Rect) {
	shadows, insets := n.Shadows, n.InsetShadows
	if look != Composited {
		shadows, insets = nil, nil
	}
	body, bg, b := n.Bounds, n.Background, n.Border
	filled := bg.Kind == color.Literal && bg.RGBA.A > 0
	shape := filled && (look != Composited || body.H != 1 || !ground(buf, clip, body, bg))
	var line color.Color
	halo, fill := false, body
	for i := len(shadows) - 1; i >= 0; i-- {
		s, cast := shadows[i], p.cast(body, shadows[i])
		if across := int(math.Round(float64(s.Spread) / stylekonst.NominalCellX)); filled && body.H == 1 && across > 0 && s.X == 0 && s.Y == 0 && s.Blur == 0 && s.Color == bg {
			fill = layout.Rect{X: body.X - across, Y: body.Y, W: body.W + 2*across, H: 1}
			continue
		}
		if p.soft(s) {
			shade(buf, clip, cast, body, s)
			continue
		}
		switch hairline := shadow(buf, clip, cast, body, s, false); {
		case hairline && (!shape || s.Token != theme.Border && s.Token != theme.Input):
			line = s.Color
		case !hairline && s.X == 0 && s.Y == 0 && s.Blur == 0:
			halo = true
		}
	}
	sides := look == Composited && body.H == 1 && b.Style != style.BorderNone && (b.Left || b.Right)
	if sides && !shape {
		line = b.Color
	}
	if look == Composited && filled && b.Style == style.BorderSingle && b.Top && b.Right && b.Bottom && b.Left {
		fill = n.Padding
	}
	if filled {
		f, cell := overlap(overlap(fill, clip), layout.Rect{W: buf.Width(), H: buf.Height()}), buffer.Cell{Grapheme: " ", Bg: bg}
		palette := p.paletted()
		for y := f.Y; y < f.Y+f.H; y++ {
			row := buf.Row(y)
			for x := f.X; x < f.X+f.W; x++ {
				under := row[x].Bg
				if translucent(bg) {
					put(buf, clip, x, y, cell)
				} else {
					buf.Set(x, y, cell)
				}
				if palette {
					row[x].Bg = apart(row[x].Bg, under, p.Profile)
				}
			}
		}
	}
	if look != Plain && n.Gradient.Kind == style.GradientLinear {
		gradient(buf, n, fill, look, clip)
	}
	for _, s := range insets {
		pad := n.Padding
		left, top, right, bottom := reach(s)
		lit := layout.Rect{X: pad.X + right, Y: pad.Y + bottom, W: pad.W - left - right, H: pad.H - top - bottom}
		shadow(buf, clip, pad, lit, s, true)
	}
	if !halo {
		tint(buf, clip, body, line)
	}
	if !sides {
		border(buf, n, look, clip)
	}
	lines(buf, n, clip, p.Widths, p.Profile == color.ANSI16, p.Covers)
}

func split(set string) (glyphs [4]string) {
	i := 0
	for at, r := range set {
		glyphs[i] = set[at : at+utf8.RuneLen(r)]
		i++
	}
	return glyphs
}

func depth(s style.Shadow) [4]float64 {
	blur := float64(s.Blur) * rasterkonst.SigmaPerBlur
	cells := func(offset style.Pixels, cell float64) float64 { return (float64(offset+s.Spread) + blur) / cell }
	return [4]float64{cells(-s.Y, stylekonst.NominalCellY), cells(s.X, stylekonst.NominalCellX), cells(s.Y, stylekonst.NominalCellY), cells(-s.X, stylekonst.NominalCellX)}
}

func whole(v float64) int {
	if v > 0 {
		return max(1, int(math.Round(v)))
	}
	return int(math.Round(v))
}

func reach(s style.Shadow) (left, top, right, bottom int) {
	d := depth(s)
	return whole(d[3]), whole(d[0]), whole(d[1]), whole(d[2])
}

func (p *Painter) soft(s style.Shadow) bool {
	return p.Identity == terminal.IdentityZed && (s.X != 0 || s.Y != 0 || s.Blur != 0)
}

func (p *Painter) cast(body layout.Rect, s style.Shadow) layout.Rect {
	left, top, right, bottom := reach(s)
	if p.soft(s) {
		d := depth(s)
		top, right, bottom, left = shades(d[0]), shades(d[1]), shades(d[2]), shades(d[3])
	}
	return layout.Rect{X: body.X - left, Y: body.Y - top, W: body.W + left + right, H: body.H + top + bottom}
}

const within = 3

func level(depth float64, cell int) int {
	switch c := depth - float64(cell-1); {
	case c >= konst.ShadeDark:
		return 2
	case c >= konst.ShadeLight, cell > 1 && c+1 >= konst.ShadeDark:
		return 1
	}
	return 0
}

func shades(depth float64) int {
	cells := 0
	for level(depth, cells+1) > 0 {
		cells++
	}
	if cells == 0 {
		return min(whole(depth), 0)
	}
	return cells
}

func shade(buf *buffer.Buffer, clip, cast, body layout.Rect, s style.Shadow) {
	if s.Color.Kind != color.Literal || s.Color.RGBA.A == 0 {
		return
	}
	ink := s.Color
	ink.RGBA.A = uint8(min(float64(ink.RGBA.A)/konst.MediumShadeAlpha, math.MaxUint8))
	d := depth(s)
	axis := func(before, after float64, v, start, size int) int {
		switch {
		case v < start:
			return level(before, start-v)
		case v >= start+size:
			return level(after, v-start-size+1)
		}
		return within
	}
	glyphs := [...]string{1: konst.LightShade, 2: konst.MediumShade}
	area := overlap(cast, clip)
	for y := area.Y; y < area.Y+area.H; y++ {
		for x := area.X; x < area.X+area.W; x++ {
			h, v := axis(d[3], d[1], x, body.X, body.W), axis(d[0], d[2], y, body.Y, body.H)
			step := min(h, v)
			if h != within && v != within {
				step--
			}
			if step <= 0 || step == within || !visible(buf, clip, x, y) {
				continue
			}
			dst := buf.At(x, y)
			if dst.Grapheme != " " && (dst.Grapheme != konst.LightShade || step < 2) {
				continue
			}
			buf.Set(x, y, buffer.Cell{Grapheme: glyphs[step], Fg: over(ink, dst.Bg), Bg: dst.Bg})
		}
	}
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

type ink uint8

const (
	noInk ink = iota
	mergeInk
	thinInk
	halfInk
	fullInk
)

func ground(buf *buffer.Buffer, clip, row layout.Rect, fill color.Color) bool {
	for x := row.X; x < row.X+row.W; x++ {
		if visible(buf, clip, x, row.Y) && over(fill, buf.At(x, row.Y).Bg) != buf.At(x, row.Y).Bg {
			return false
		}
	}
	return true
}

func tint(buf *buffer.Buffer, clip, row layout.Rect, c color.Color) {
	if c.Kind != color.Literal {
		return
	}
	c.RGBA.A = uint8(math.Round(max(float64(c.RGBA.A), konst.OneRowLine*math.MaxUint8)))
	for x := row.X; x < row.X+row.W; x++ {
		if !visible(buf, clip, x, row.Y) {
			continue
		}
		under := buf.At(x, row.Y).Bg
		share := konst.OneRowTint
		if lift := luma(over(c, under).RGBA) - luma(under.RGBA); lift > 0 {
			share = min(max(share, konst.OneRowLiftLuma/float64(lift)), 1)
		}
		k := c
		k.RGBA.A = uint8(math.Round(float64(c.RGBA.A) * share))
		put(buf, clip, x, row.Y, buffer.Cell{Grapheme: " ", Bg: k})
	}
}

func shadow(buf *buffer.Buffer, clip, shaded, lit layout.Rect, s style.Shadow, inset bool) (hairline bool) {
	var weight [4]ink
	drop := s.X != 0 || s.Y != 0 || s.Blur != 0
	if !drop {
		ring := func(cell float64) ink {
			switch {
			case math.Round(float64(s.Spread)*konst.CellEighths/cell) > 1:
				return halfInk
			case lit.H == 1:
				return mergeInk
			}
			return thinInk
		}
		v, h := ring(stylekonst.NominalCellY), ring(stylekonst.NominalCellX)
		weight = [4]ink{v, h, v, h}
	} else {
		for side, v := range depth(s) {
			switch covered := v - float64(whole(v)-1); {
			case covered >= konst.ShadowFullCell:
				weight[side] = fullInk
			case covered >= konst.ShadowHalfCell:
				weight[side] = halfInk
			}
		}
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
			switch weight[side] {
			case noInk:
				continue
			case fullInk:
				put(buf, clip, x, y, buffer.Cell{Grapheme: " ", Bg: s.Color})
				continue
			}
			if !visible(buf, clip, x, y) {
				continue
			}
			if cell, ok := edge(buf.At(x, y), side, weight[side], s.Color, drop && !inset); ok {
				buf.Set(x, y, cell)
			}
		}
	}
	return weight[1] == mergeInk
}

func edge(dst buffer.Cell, side int, weight ink, c color.Color, covers bool) (buffer.Cell, bool) {
	thin, half := split(konst.SingleLines), split(konst.HalfEdges)
	glyph, fg := thin[side], over(c, dst.Fg)
	if weight == halfInk || dst.Grapheme == half[side] {
		glyph = half[side]
	}
	switch {
	case dst.Grapheme == half[side]:
	case weight == mergeInk:
		return dst, false
	case dst.Grapheme == thin[side]:
	case covers && drawn(dst.Grapheme), dst.Grapheme == " " || weight == halfInk && dst.Grapheme == half[(side+2)%4]:
		fg = over(c, dst.Bg)
	default:
		return dst, false
	}
	return buffer.Cell{Grapheme: glyph, Fg: fg, Bg: dst.Bg}, true
}

func drawn(cluster string) bool {
	r, size := utf8.DecodeRuneInString(cluster)
	return size == len(cluster) && r >= konst.FirstLineGlyph && r <= konst.LastBlockGlyph
}

func glyphs(b scene.Border, look Look) (edges string, corners [4]string) {
	if look == Glyphs {
		return "", corners
	}
	switch b.Style {
	case style.BorderNone:
		return "", corners
	case style.BorderSingle:
		return konst.SingleLines, rounded(b.Radius)
	case style.BorderDashed:
		return konst.DashedLines, rounded(b.Radius)
	case style.BorderDotted:
		return konst.DottedLines, rounded(b.Radius)
	case style.BorderDouble:
		return konst.DoubleLines, split(konst.DoubleCorners)
	}
	panic(fmt.Sprintf("paint: unknown border style %d", b.Style))
}

func rounded(r style.Radius) (corners [4]string) {
	square, round := split(konst.SquareCorners), split(konst.RoundedCorners)
	for i, c := range [4]style.Corner{style.CornerTopLeft, style.CornerTopRight, style.CornerBottomLeft, style.CornerBottomRight} {
		corners[i] = square[i]
		if r.At(c) != style.RadiusNone {
			corners[i] = round[i]
		}
	}
	return corners
}

func border(buf *buffer.Buffer, n *scene.Node, look Look, clip layout.Rect) {
	r, b := n.Bounds, n.Border
	edgeSet, corners := glyphs(b, look)
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

func lines(buf *buffer.Buffer, n *scene.Node, clip layout.Rect, widths text.Widths, console bool, covers func(string) bool) {
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
	for g := range Placed(n, widths, clip) {
		if g.Cluster == "\t" {
			for x := g.X; x < g.X+g.Width; x++ {
				put(buf, clip, x, g.Y, ink)
			}
			continue
		}
		cell := ink
		cell.Grapheme = g.Cluster
		switch {
		case console && len(g.Cluster) > 1:
			cell.Grapheme = standIn(g.Cluster)
		case covers != nil && len(g.Cluster) > 1 && !covers(g.Cluster):
			for x := g.X + 1; x < g.X+g.Width; x++ {
				put(buf, clip, x, g.Y, ink)
			}
			cell.Grapheme, g.Width = standIn(g.Cluster), 1
			if cell.Grapheme == g.Cluster || !covers(cell.Grapheme) {
				cell.Grapheme = " "
			}
		}
		if g.Width > 1 {
			cell.Width = buffer.Wide
		}
		put(buf, clip, g.X, g.Y, cell)
	}
}

func standIn(cluster string) string {
	want, size := utf8.DecodeRuneInString(cluster)
	if size != len(cluster) {
		return cluster
	}
	standIns := konst.ConsoleStandIns
	for _, r := range konst.ConsoleMissing {
		_, next := utf8.DecodeRuneInString(standIns)
		if r == want {
			return standIns[:next]
		}
		standIns = standIns[next:]
	}
	return cluster
}

func printable(line string) bool {
	for i := range len(line) {
		if line[i] < ' ' || line[i] > '~' {
			return false
		}
	}
	return true
}

type Glyph struct {
	X, Y, Width int
	Cluster     string
}

func Placed(n *scene.Node, widths text.Widths, rows layout.Rect) iter.Seq[Glyph] {
	return func(yield func(Glyph) bool) {
		r := n.Content
		end := r.X + r.W
		lines := n.Lines(widths)
		for i, line := range lines[:min(len(lines), r.H)] {
			y, x := r.Y+i, r.X
			if y < rows.Y || y >= rows.Y+rows.H {
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
			if printable(line) {
				for i := range min(len(line), end-x) {
					if !yield(Glyph{x + i, y, 1, line[i : i+1]}) {
						return
					}
				}
				continue
			}
			for cluster := range text.Graphemes(line) {
				w := widths.Width(cluster)
				if cluster == "\t" {
					w = min(r.X+((x-r.X)/konst.TabStop+1)*konst.TabStop, end) - x
				}
				if w <= 0 {
					continue
				}
				if x+w > end {
					break
				}
				if !yield(Glyph{x, y, w, cluster}) {
					return
				}
				x += w
			}
		}
	}
}
