package graphics

import (
	"image"
	"math"
	"slices"

	"github.com/pehcastro/twind/internal/konst/graphics"
)

func GDIPixels(dst []byte, lines [][]Run, cell image.Point, text uint64) []byte {
	width := int(lines[0][len(lines[0])-1].End)
	stride, cols, at := width*graphics.GDIBytes, width/cell.X, len(dst)
	dst = slices.Grow(dst, stride*len(lines))[:at+stride*len(lines)]
	for y, line := range lines {
		row := dst[at+y*stride : at+(y+1)*stride]
		if y%cell.Y != 0 && &line[0] == &lines[y-1][0] {
			copy(row, dst[at+(y-1)*stride:])
			continue
		}
		x := 0
		for _, r := range line {
			var px [graphics.GDIBytes]byte
			if a := r.Pixel >> 24; a >= graphics.GDIOpaque {
				px = [graphics.GDIBytes]byte{opaque(r.Pixel>>16, a), opaque(r.Pixel>>8, a), opaque(r.Pixel, a), math.MaxUint8}
			}
			seg := row[x*graphics.GDIBytes : int(r.End)*graphics.GDIBytes]
			copy(seg, px[:])
			for n := graphics.GDIBytes; n < len(seg); n *= 2 {
				copy(seg[n:], seg[:n])
			}
			x = int(r.End)
		}
		for c, first := 0, y/cell.Y*cols; c < cols; c++ {
			if text>>(first+c)&1 != 0 {
				clear(row[c*cell.X*graphics.GDIBytes : (c+1)*cell.X*graphics.GDIBytes])
			}
		}
	}
	return dst
}

func opaque(channel, alpha uint32) byte {
	return byte(((channel&math.MaxUint8)*math.MaxUint8 + alpha/2) / alpha)
}
