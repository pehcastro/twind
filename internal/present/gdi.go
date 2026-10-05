package present

import (
	"image"
	"slices"

	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	konst "github.com/twind-dev/twind/internal/konst/paint"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/graphics"
	"github.com/twind-dev/twind/twi/terminal"
)

func (s *Screen) gdi() {
	s.sending = s.sending[:0]
	for t, dirty := range s.dirty {
		if s.shifted[t] && !dirty {
			s.hashes[t], s.plain[t] = s.hash(t), s.Profile == color.TrueColor && s.plainTile(s.tileLines(t, s.lineRuns[:0]))
		}
		s.inking = s.glyphs(t, s.inking[:0])
		inked := !slices.Equal(s.inking, s.inks[t])
		if inked {
			s.inks[t] = slices.Clone(s.inking)
		}
		dirty = dirty || s.shifted[t] || inked
		s.shifted[t] = false
		plain := s.plain[t] && len(s.inking) == 0
		switch {
		case plain && (!dirty || s.sent[t] == 0):
		case plain:
			s.painting.Tiles = append(s.painting.Tiles, terminal.Tile{Cells: s.tiles[t]})
			s.sent[t] = 0
		case dirty && (s.hashes[t] != s.sent[t] || inked), s.sent[t] != 0 && s.mask(t) != s.masks[t]:
			s.sending = append(s.sending, t)
		}
	}
	s.gdiPix = slices.Grow(s.gdiPix[:0], len(s.sending)*s.band*konst.TileColumns*s.Cell.X*s.Cell.Y*graphicskonst.GDIBytes)
	for _, t := range s.sending {
		at := len(s.gdiPix)
		s.lineRuns = s.tileLines(t, s.lineRuns[:0])
		s.masks[t], s.sent[t] = s.mask(t), s.hashes[t]
		s.gdiPix = graphics.GDIPixels(s.gdiPix, s.lineRuns, s.Cell, s.masks[t])
		s.strips(t, s.gdiPix[at:])
		s.painting.Tiles = append(s.painting.Tiles, terminal.Tile{Cells: s.tiles[t], Pix: s.gdiPix[at:len(s.gdiPix):len(s.gdiPix)], Glyphs: s.inks[t]})
	}
}

func (s *Screen) lacked(c buffer.Cell) bool {
	return s.Covers != nil && len(c.Grapheme) > 1 && !s.Covers(c.Grapheme)
}

func (s *Screen) drawn(text []buffer.Cell, x int) bool {
	return s.lacked(text[x]) || text[x].Width == buffer.Continuation && x > 0 && s.lacked(text[x-1])
}

func (s *Screen) glyphs(t int, dst []terminal.Glyph) []terminal.Glyph {
	if s.Covers == nil {
		return dst
	}
	cells := s.tiles[t]
	for y := cells.Min.Y; y < cells.Max.Y; y++ {
		text := s.text.Row(y)
		for x := cells.Min.X; x < cells.Max.X; x++ {
			if c := text[x]; s.lacked(c) {
				dst = append(dst, terminal.Glyph{Cell: image.Pt(x, y), Cluster: c.Grapheme, Fg: c.Fg.RGBA, Bold: c.Attr&buffer.Bold != 0, Wide: c.Width == buffer.Wide})
			}
		}
	}
	return dst
}

func (s *Screen) mask(t int) uint64 {
	cells, mask := s.tiles[t], uint64(0)
	for y := cells.Min.Y; y < cells.Max.Y; y++ {
		text := s.text.Row(y)
		for x := cells.Min.X; x < cells.Max.X; x++ {
			if !blank(text[x]) && !s.drawn(text, x) || s.besideSymbol(text, x, y) {
				mask |= 1 << ((y-cells.Min.Y)*cells.Dx() + x - cells.Min.X)
			}
		}
	}
	return mask
}

func (s *Screen) strips(t int, pix []byte) {
	cells, in, width := s.tiles[t], s.inset(), s.Cell.X*graphicskonst.GDIBytes
	for cy := range cells.Dy() {
		middle := s.lineRuns[cy*s.Cell.Y+s.Cell.Y/2]
		for k := range 2 * in {
			py := cy*s.Cell.Y + k
			if k >= in {
				py += s.Cell.Y - 2*in
			}
			line, converted := s.lineRuns[py], false
			for cx := range cells.Dx() {
				if s.masks[t]>>(cy*cells.Dx()+cx)&1 == 0 || same(line, middle, cx*s.Cell.X, (cx+1)*s.Cell.X) {
					continue
				}
				if !converted {
					s.strip, converted = graphics.GDIPixels(s.strip[:0], [][]run{line}, s.Cell, 0), true
				}
				seg := pix[(py*cells.Dx()+cx)*width:][:width]
				copy(seg, s.strip[cx*width:])
				for i := 0; i < width; i += graphicskonst.GDIBytes {
					if seg[i+3] == 0 {
						seg[i], seg[i+1], seg[i+2], seg[i+3] = byte(s.page>>16), byte(s.page>>8), byte(s.page), byte(s.page>>24)
					}
				}
			}
		}
	}
}

func same(a, b []run, from, to int) bool {
	if &a[0] == &b[0] {
		return true
	}
	for i, j, x := find(a, from), find(b, from), from; x < to; {
		if a[i].Pixel != b[j].Pixel {
			return false
		}
		x = int(min(a[i].End, b[j].End))
		if x == int(a[i].End) {
			i++
		}
		if x == int(b[j].End) {
			j++
		}
	}
	return true
}
