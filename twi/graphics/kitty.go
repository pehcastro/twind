package graphics

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"image"
	"slices"

	"github.com/twind-dev/twind/internal/konst/graphics"
)

type Kitty struct {
	row  []byte
	file bytes.Buffer
	z    *zlib.Writer
}

func (k *Kitty) Encode(dst []byte, img *image.RGBA, at Placement, id, placement uint32) []byte {
	if img.Rect.Empty() {
		return dst
	}
	if one := flat(img); one != nil {
		img = one
	}
	w, h := img.Rect.Dx(), img.Rect.Dy()
	opaque := true
	for y := range h {
		r := row(img, y)
		for x := 3; x < len(r); x += 4 {
			opaque = opaque && r[x] == 255
		}
	}
	k.file.Reset()
	if k.z == nil {
		k.z, _ = zlib.NewWriterLevel(&k.file, zlib.BestSpeed)
	}
	k.z.Reset(&k.file)
	bpp := 4
	if opaque {
		bpp = 3
	}
	k.row = slices.Grow(k.row[:0], w*bpp)[:w*bpp]
	for y := range h {
		r := row(img, y)
		for x, o := 0, 0; x < len(r); x, o = x+4, o+bpp {
			p, out := r[x:x+4:x+4], k.row[o:o+bpp:o+bpp]
			if a := uint32(p[3]); a != 0 && a != 255 {
				out[0], out[1], out[2] = byte(uint32(p[0])*0xffff/a>>8), byte(uint32(p[1])*0xffff/a>>8), byte(uint32(p[2])*0xffff/a>>8)
			} else {
				out[0], out[1], out[2] = p[0], p[1], p[2]
			}
			if !opaque {
				out[3] = p[3]
			}
		}
		_, _ = k.z.Write(k.row)
	}
	_ = k.z.Close()
	dst = fmt.Appendf(dst, "\x1b[%d;%dH\x1b_Ga=T,f=%d,o=z,s=%d,v=%d,i=%d,p=%d,c=%d,r=%d,z=-1,C=1,q=2",
		at.Row+1, at.Col+1, 8*bpp, w, h, id, placement, at.Cols, at.Rows)
	file := k.file.Bytes()
	for start := 0; start < len(file); start += graphics.KittyRawChunk {
		if start > 0 {
			dst = append(dst, "\x1b_Gq=2"...)
		}
		end := min(start+graphics.KittyRawChunk, len(file))
		dst = fmt.Appendf(dst, ",m=%d;", min(len(file)-end, 1))
		dst = base64.StdEncoding.AppendEncode(dst, file[start:end])
		dst = append(dst, "\x1b\\"...)
	}
	return dst
}

func KittyDelete(dst []byte, id uint32) []byte {
	return fmt.Appendf(dst, "\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", id)
}
