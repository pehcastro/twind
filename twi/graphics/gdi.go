package graphics

import (
	"image"
	"math"
	"slices"

	"github.com/twind-dev/twind/internal/konst/graphics"
)

func GDIPixels(dst []byte, lines [][]Run, cell image.Point, text uint64) []byte {
	width := int(lines[0][len(lines[0])-1].End)
	dst = slices.Grow(dst, width*len(lines)*graphics.GDIBytes)
	for y, line := range lines {
		first, x := y/cell.Y*(width/cell.X), 0
		for _, r := range line {
			var px [graphics.GDIBytes]byte
			if a := r.Pixel >> 24; a >= graphics.GDIOpaque {
				px = [graphics.GDIBytes]byte{opaque(r.Pixel>>16, a), opaque(r.Pixel>>8, a), opaque(r.Pixel, a), math.MaxUint8}
			}
			for ; x < int(r.End); x++ {
				if text>>(first+x/cell.X)&1 != 0 {
					dst = append(dst, 0, 0, 0, 0)
					continue
				}
				dst = append(dst, px[:]...)
			}
		}
	}
	return dst
}

func opaque(channel, alpha uint32) byte {
	return byte(((channel&math.MaxUint8)*math.MaxUint8 + alpha/2) / alpha)
}
