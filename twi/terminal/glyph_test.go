package terminal

import (
	"image"
	"testing"

	"github.com/pehcastro/twind/twi/color"
)

func TestGlyphsBlendIntoTheirCell(t *testing.T) {
	f := &fakeTTY{lacking: "ש", lacksFace: "Courier New", coverage: func(_, _ string, size image.Point, _ bool) []uint8 {
		mask := make([]uint8, size.X*size.Y)
		mask[2*size.X+1] = 255
		mask[3*size.X+2] = 128
		return mask
	}}
	b := &Backend{tty: f}
	cell, tile := image.Pt(4, 5), image.Rect(3, 1, 5, 2)
	pix := make([]byte, tile.Dx()*cell.X*cell.Y*4)
	for i := 3; i < len(pix); i += 4 {
		pix[i-3], pix[i-2], pix[i-1], pix[i] = 10, 20, 30, 255
	}
	b.ink(Pixels{Cell: cell, Tiles: []Tile{{Cells: tile, Pix: pix, Glyphs: []Glyph{{Cell: image.Pt(4, 1), Cluster: "ש", Fg: color.RGBA{R: 200, G: 100, B: 50, A: 255}}}}}})
	at := func(x, y int) [4]byte {
		i := (y*tile.Dx()*cell.X + x) * 4
		return [4]byte(pix[i : i+4])
	}
	if got, want := at(cell.X+1, 2), [4]byte{50, 100, 200, 255}; got != want {
		t.Errorf("full coverage pixel %v, want the glyph colour %v in BGRA", got, want)
	}
	if got, want := at(cell.X+2, 3), [4]byte{30, 60, 115, 255}; got != want {
		t.Errorf("half coverage pixel %v, want %v", got, want)
	}
	if got := at(1, 2); got != [4]byte{10, 20, 30, 255} {
		t.Errorf("the cell left of the glyph changed to %v", got)
	}
	if len(f.drawn) != 1 || f.drawn[0] != "Segoe UI ש" {
		t.Errorf("drawn with %q, want the first fallback face that has the letter, Segoe UI", f.drawn)
	}
}
