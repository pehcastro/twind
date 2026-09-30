package present

import (
	"fmt"
	"image"

	konst "github.com/twind-dev/twind/internal/konst/scene"
	termkonst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/terminal"
)

func (s *Screen) scroll(f *scene.Frame, sc scene.Scroll) {
	clip, by := f.Layers[sc.Layer].Clip, sc.By
	area := image.Rect(clip.Min.X/s.Cell.X, clip.Min.Y/s.Cell.Y, clip.Max.X/s.Cell.X, clip.Max.Y/s.Cell.Y)
	if !s.Margins {
		area.Min.X, area.Max.X = 0, s.cols
	}
	lines := by.Y / s.Cell.Y
	if s.Graphics != terminal.GraphicsSixel || by.X != 0 || max(lines, -lines) >= area.Dy() {
		s.damage(clip)
		return
	}
	s.out.WriteString(termkonst.Reset)
	if s.Margins {
		s.out.WriteString(termkonst.CSI + konst.MarginsOn)
	}
	fmt.Fprintf(&s.out, "%s%d;%d%s", termkonst.CSI, area.Min.Y+1, area.Max.Y, konst.Rows)
	if s.Margins {
		fmt.Fprintf(&s.out, "%s%d;%d%s", termkonst.CSI, area.Min.X+1, area.Max.X, konst.Columns)
	}
	way, count := konst.ScrollDown, lines
	if lines < 0 {
		way, count = konst.ScrollUp, -lines
	}
	fmt.Fprintf(&s.out, "%s%d%s", termkonst.CSI, count, way)
	if s.Margins {
		s.out.WriteString(termkonst.CSI + konst.MarginsOff)
	}
	s.out.WriteString(termkonst.CSI + konst.Rows)
	s.writer = terminal.Writer{Out: &s.out, Profile: s.Profile}
	cells := func(y int) (int, int) { return y*s.cols + area.Min.X, y*s.cols + area.Max.X }
	slide(area.Min.Y, area.Max.Y, lines, func(dst, src int) {
		copy(s.shown.Row(dst)[area.Min.X:area.Max.X], s.shown.Row(src)[area.Min.X:area.Max.X])
		from, to := cells(src)
		at, _ := cells(dst)
		copy(s.samples[at:], s.samples[from:to])
		copy(s.sampled[at:], s.sampled[from:to])
	}, func(dst int) {
		s.shown.Fill(buffer.Rect{X: area.Min.X, Y: dst, W: area.Dx(), H: 1}, buffer.Cell{Grapheme: "\x00"})
		from, to := cells(dst)
		clear(s.sampled[from:to])
		s.damage(s.pixels(image.Rect(area.Min.X, dst, area.Max.X, dst+1)))
	})
	px := s.pixels(area)
	row := func(y int) []uint8 {
		return s.surface.Pix[s.surface.PixOffset(px.Min.X, y):s.surface.PixOffset(px.Max.X, y)]
	}
	slide(px.Min.Y, px.Max.Y, by.Y, func(dst, src int) { copy(row(dst), row(src)) }, func(dst int) { clear(row(dst)) })
	for t, tile := range s.tiles {
		s.moved[t] = s.moved[t] || tile.Overlaps(area)
	}
	moving := make([]bool, len(f.Layers))
	for i := range f.Layers {
		l := &f.Layers[i]
		if moving[i] = i == sc.Layer || l.Parent >= 0 && moving[l.Parent]; moving[i] {
			continue
		}
		for j := range l.Boxes {
			s.unshifted(l, &l.Boxes[j], by.Y, px)
		}
	}
}

func (s *Screen) unshifted(l *scene.Layer, b *scene.Box, dy int, area image.Rectangle) {
	v := b.Visual.Add(l.Origin).Intersect(l.Clip)
	band := s.boxRaster(b).uniform
	top, bottom := max(v.Min.Y, b.Visual.Min.Y+l.Origin.Y+band[0]), min(v.Max.Y, b.Visual.Min.Y+l.Origin.Y+band[1])
	outer := image.Rect(v.Min.X, min(v.Min.Y, v.Min.Y+dy), v.Max.X, max(v.Max.Y, v.Max.Y+dy))
	inner := image.Rect(v.Min.X, max(top, top+dy), v.Max.X, min(bottom, bottom+dy))
	if inner.Empty() {
		s.damage(outer.Intersect(area))
		return
	}
	s.damage(image.Rect(outer.Min.X, outer.Min.Y, outer.Max.X, inner.Min.Y).Intersect(area))
	s.damage(image.Rect(outer.Min.X, inner.Max.Y, outer.Max.X, outer.Max.Y).Intersect(area))
}

func slide(lo, hi, by int, move func(dst, src int), blank func(dst int)) {
	for i := range hi - lo {
		dst := lo + i
		if by > 0 {
			dst = hi - 1 - i
		}
		if src := dst - by; src >= lo && src < hi {
			move(dst, src)
		} else {
			blank(dst)
		}
	}
}

func (s *Screen) scrollbars(n *scene.Node) {
	for i := range n.Children {
		s.scrollbars(&n.Children[i])
	}
	from, to, ok := n.Thumb(konst.ThumbEighths)
	if !ok {
		return
	}
	x, clip := n.Padding.X+n.Padding.W-1, n.Clip
	for y := n.Padding.Y + from/konst.ThumbEighths; (y-n.Padding.Y)*konst.ThumbEighths < to; y++ {
		if x < clip.X || x >= clip.X+clip.W || y < clip.Y || y >= clip.Y+clip.H {
			continue
		}
		top := max(from-(y-n.Padding.Y)*konst.ThumbEighths, 0)
		bottom := min(to-(y-n.Padding.Y)*konst.ThumbEighths, konst.ThumbEighths)
		under := s.want.At(x, y).Bg
		c := buffer.Cell{Grapheme: block(konst.ThumbEighths - top), Fg: thumb(n.Foreground, under), Bg: under}
		if bottom < konst.ThumbEighths {
			c.Grapheme, c.Attr = block(konst.ThumbEighths-bottom), buffer.Inverse
		}
		s.want.Set(x, y, c)
	}
}

func block(eighths int) string { return string(konst.LowerEighth + rune(eighths-1)) }

func thumb(fg, bg color.Color) color.Color {
	if fg.Kind != color.Literal || bg.Kind != color.Literal {
		return fg
	}
	mix := func(f, b uint8) uint8 { return uint8(float64(b) + (float64(f)-float64(b))*konst.ThumbAlpha) }
	return color.Color{Kind: color.Literal, RGBA: color.RGBA{R: mix(fg.RGBA.R, bg.RGBA.R), G: mix(fg.RGBA.G, bg.RGBA.G), B: mix(fg.RGBA.B, bg.RGBA.B), A: bg.RGBA.A}}
}
