package paint

import (
	"math"

	konst "github.com/twind-dev/twind/internal/konst/paint"

	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
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
	src.Bg = over(src.Bg, under)
	src.Fg = over(src.Fg, src.Bg)
	buf.Set(x, y, src)
}

func fade(dst, layer *buffer.Buffer, opacity float64) {
	whole := layout.Rect{W: dst.Width(), H: dst.Height()}
	scale := func(c *color.Color) {
		if c.Kind == color.Literal {
			c.RGBA.A = uint8(math.Round(float64(c.RGBA.A) * opacity))
		}
	}
	for y := range layer.Height() {
		for x, c := range layer.Row(y) {
			if c.Width == buffer.Continuation {
				continue
			}
			scale(&c.Fg)
			scale(&c.Bg)
			put(dst, whole, x, y, c)
		}
	}
}
