package paint

import (
	"math"
	"strings"
	"unicode/utf8"

	konst "github.com/pehcastro/twind/internal/konst/paint"

	"github.com/pehcastro/twind/internal/buffer"
	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/twi/color"
)

func translucent(c color.Color) bool { return c.Kind == color.Literal && c.RGBA.A < math.MaxUint8 }

func over(src, dst color.Color) color.Color {
	if src.Kind != color.Literal || src.RGBA.A == math.MaxUint8 {
		return src
	}
	if src.RGBA.A == 0 {
		return dst
	}
	const full = math.MaxUint8
	s, d := src.RGBA, dst.RGBA
	if dst.Kind != color.Literal {
		d = color.RGBA{}
	}
	srcWeight := uint32(s.A) * full
	dstWeight := uint32(d.A) * (full - uint32(s.A))
	total := srcWeight + dstWeight
	mix := func(s, d uint8) uint8 { return uint8((uint32(s)*srcWeight + uint32(d)*dstWeight + total/2) / total) }
	return color.Color{Kind: color.Literal, RGBA: color.RGBA{
		R: mix(s.R, d.R), G: mix(s.G, d.G), B: mix(s.B, d.B), A: uint8((total + full/2) / full),
	}}
}

func (p *Painter) paletted() bool { return p.Profile == color.ANSI16 || p.Profile == color.ANSI256 }

func luma(k color.RGBA) int { return 299*int(k.R) + 587*int(k.G) + 114*int(k.B) }

func apart(c, from color.Color, p color.Profile) color.Color {
	index := color.RGBA.ANSI256
	if p == color.ANSI16 {
		index = color.RGBA.ANSI16
	}
	opaque := func(k color.Color) bool { return k.Kind == color.Literal && k.RGBA.A == math.MaxUint8 }
	if !opaque(c) || !opaque(from) || c.RGBA == from.RGBA || index(c.RGBA) != index(from.RGBA) {
		return c
	}
	white, toward := color.RGBA{R: math.MaxUint8, G: math.MaxUint8, B: math.MaxUint8, A: math.MaxUint8}, color.Color{Kind: color.Literal}
	if lc, lf := luma(c.RGBA), luma(from.RGBA); lc > lf || lc == lf && 2*lf < luma(white) {
		toward.RGBA = white
	}
	for step := 1; step <= konst.ContrastSteps; step++ {
		toward.RGBA.A = uint8(step * math.MaxUint8 / konst.ContrastSteps)
		if next := over(toward, c); index(next.RGBA) != index(from.RGBA) {
			return next
		}
	}
	return c
}

func visible(buf *buffer.Buffer, clip layout.Rect, x, y int) bool {
	return x >= max(clip.X, 0) && y >= max(clip.Y, 0) && x < min(clip.X+clip.W, buf.Width()) && y < min(clip.Y+clip.H, buf.Height())
}

func put(buf *buffer.Buffer, clip layout.Rect, x, y int, src buffer.Cell) {
	if !visible(buf, clip, x, y) {
		return
	}
	if !translucent(src.Bg) && !translucent(src.Fg) {
		buf.Set(x, y, src)
		return
	}
	dst := &buf.Row(y)[x]
	if src.Grapheme == " " && src.Attr == 0 && translucent(src.Bg) {
		if float64(src.Bg.RGBA.A) >= konst.CoverAlpha*math.MaxUint8 {
			buf.Set(x, y, buffer.Cell{Grapheme: " ", Bg: over(src.Bg, dst.Bg)})
			return
		}
		dst.Bg = over(src.Bg, dst.Bg)
		if dst.Fg.Kind == color.Literal {
			dst.Fg = over(src.Bg, dst.Fg)
		}
		return
	}
	under := dst.Bg
	if dst.Grapheme == konst.UpperHalf {
		upper := dst.Fg
		upper.RGBA.A /= 2
		under = over(upper, dst.Bg)
	}
	stroke := over(src.Bg, dst.Fg)
	src.Bg = over(src.Bg, under)
	if src.Grapheme != dst.Grapheme || src.Attr != dst.Attr || src.Grapheme == " " || dst.Fg.Kind != color.Literal {
		stroke = src.Bg
	}
	src.Fg = over(src.Fg, stroke)
	buf.Set(x, y, src)
}

func fade(dst, layer *buffer.Buffer, opacity float64, span layout.Rect) {
	scale := func(c *color.Color) {
		if c.Kind == color.Literal {
			c.RGBA.A = uint8(math.Round(float64(c.RGBA.A) * opacity))
		}
	}
	for y := span.Y; y < span.Y+span.H; y++ {
		for x := span.X; x < span.X+span.W; x++ {
			c := layer.At(x, y)
			if c.Width == buffer.Continuation {
				continue
			}
			under := dst.At(x, y)
			if r, _ := utf8.DecodeRuneInString(c.Grapheme); c.Bg.RGBA.A == 0 && strings.ContainsRune(konst.HalfEdges, r) && under.Grapheme != " " && !drawn(under.Grapheme) {
				continue
			}
			solid := c.Bg.Kind == color.Literal && c.Bg.RGBA.A == math.MaxUint8 && c.Grapheme == " " && c.Attr == 0
			scale(&c.Fg)
			scale(&c.Bg)
			if solid && visible(dst, span, x, y) {
				dst.Set(x, y, buffer.Cell{Grapheme: " ", Bg: over(c.Bg, under.Bg)})
				continue
			}
			put(dst, span, x, y, c)
		}
	}
}
