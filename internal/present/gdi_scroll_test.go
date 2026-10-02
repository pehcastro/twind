package present

import (
	"bytes"
	"image"
	"slices"
	"strconv"
	"testing"

	"github.com/twind-dev/twind/twi/terminal"
)

func applied(calls []terminal.Pixels) []byte {
	var pix []byte
	for _, c := range calls {
		size := image.Pt(c.Grid.X*c.Cell.X, c.Grid.Y*c.Cell.Y)
		stride := size.X * 4
		if c.Clear || pix == nil {
			pix = make([]byte, stride*size.Y)
		}
		for _, tile := range c.Tiles {
			r := image.Rect(tile.Cells.Min.X*c.Cell.X, tile.Cells.Min.Y*c.Cell.Y, tile.Cells.Max.X*c.Cell.X, tile.Cells.Max.Y*c.Cell.Y)
			for y := r.Min.Y; y < r.Max.Y; y++ {
				row := pix[y*stride+r.Min.X*4 : y*stride+r.Max.X*4]
				if tile.Pix == nil {
					clear(row)
					continue
				}
				copy(row, tile.Pix[(y-r.Min.Y)*r.Dx()*4:])
			}
		}
	}
	return pix
}

func TestGDIScrollMatchesAFreshFrame(t *testing.T) {
	offsets := []int{0, 1, 2, 3, 2, 1, 0, 10, 9, 30, 29, 180, 179, 178, 0}
	for _, id := range []terminal.Identity{terminal.IdentityConhost, terminal.IdentityZed} {
		for _, beside := range []int{-1, 10, 60} {
			name := "identity " + strconv.Itoa(int(id)) + ", box at " + strconv.Itoa(beside)
			s, _, p := gdiScreen()
			s.Identity = id
			for i, root := range scrolled(t, offsets...) {
				if beside >= 0 {
					root.Children = append(root.Children, box(beside, 6+3*i%12, 3, 4, ink(200, 60, 40, 255)))
				}
				frame(t, s, root)
				fresh, _, fp := gdiScreen()
				fresh.Identity = id
				frame(t, fresh, root)
				at := name + ", offset " + strconv.Itoa(offsets[i])
				if !bytes.Equal(applied(p.calls), applied(fp.calls)) {
					t.Errorf("%s: the painted surface differs from a fresh frame", at)
				}
				for y := range rows {
					if !slices.Equal(onScreen(s.shown.Row(y)), onScreen(fresh.shown.Row(y))) {
						t.Errorf("%s: row %d on screen is %q, a fresh frame shows %q", at, y, cells(s.shown.Row(y)), cells(fresh.shown.Row(y)))
					}
				}
				for tile, sent := range s.sent {
					if sent != 0 && sent != s.hash(tile) {
						t.Errorf("%s: tile %v is believed painted as other pixels than its lines", at, s.tiles[tile])
					}
				}
				if i == 0 || beside >= 0 {
					continue
				}
				v := view(root)
				exposed, one := map[int]int{1: v.Y + v.H - 1, -1: v.Y}[offsets[i]-offsets[i-1]]
				if !one {
					continue
				}
				row, bar := image.Rect(v.X, exposed, v.X+v.W, exposed+1), image.Rect(v.X+v.W-1, v.Y, v.X+v.W, v.Y+v.H)
				for _, tile := range s.drawing {
					if !s.tiles[tile].Overlaps(row) && !s.tiles[tile].Overlaps(bar) {
						t.Errorf("%s after a one-row step: tile %v rasterised, outside the exposed row %v and the scrollbar %v", at, s.tiles[tile], row, bar)
					}
				}
			}
		}
	}
}
