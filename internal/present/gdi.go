package present

import (
	"slices"

	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	konst "github.com/twind-dev/twind/internal/konst/paint"
	"github.com/twind-dev/twind/twi/graphics"
	"github.com/twind-dev/twind/twi/terminal"
)

func (s *Screen) gdi() {
	s.sending = s.sending[:0]
	for t, dirty := range s.dirty {
		switch {
		case s.plain[t] && (!dirty || s.sent[t] == 0):
		case s.plain[t]:
			s.painting.Tiles = append(s.painting.Tiles, terminal.Tile{Cells: s.tiles[t]})
			s.sent[t] = 0
		case dirty && s.hashes[t] != s.sent[t], s.sent[t] != 0 && s.mask(t) != s.masks[t]:
			s.sending = append(s.sending, t)
		}
	}
	s.gdiPix = slices.Grow(s.gdiPix[:0], len(s.sending)*s.band*konst.TileColumns*s.Cell.X*s.Cell.Y*graphicskonst.GDIBytes)
	for _, t := range s.sending {
		at := len(s.gdiPix)
		s.lineRuns = s.tileLines(t, s.lineRuns[:0])
		s.masks[t], s.sent[t] = s.mask(t), s.hashes[t]
		s.gdiPix = graphics.GDIPixels(s.gdiPix, s.lineRuns, s.Cell, s.masks[t])
		s.painting.Tiles = append(s.painting.Tiles, terminal.Tile{Cells: s.tiles[t], Pix: s.gdiPix[at:len(s.gdiPix):len(s.gdiPix)]})
	}
}

func (s *Screen) mask(t int) uint64 {
	cells, mask := s.tiles[t], uint64(0)
	for y := cells.Min.Y; y < cells.Max.Y; y++ {
		for x := cells.Min.X; x < cells.Max.X; x++ {
			if !blank(s.text.At(x, y)) {
				mask |= 1 << ((y-cells.Min.Y)*cells.Dx() + x - cells.Min.X)
			}
		}
	}
	return mask
}
