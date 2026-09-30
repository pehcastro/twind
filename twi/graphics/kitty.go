package graphics

import (
	"encoding/base64"
	"fmt"
	"image"

	"github.com/twind-dev/twind/internal/konst/graphics"
)

type Kitty struct {
	file []byte
	z    deflater
}

func (k *Kitty) Encode(dst []byte, img *image.RGBA, at Placement, id, placement uint32) []byte {
	if img.Rect.Empty() {
		return dst
	}
	img = k.z.prepare(img)
	var bpp int
	k.file, bpp = k.z.zlib(k.file[:0], img, rawRows)
	dst = fmt.Appendf(dst, "\x1b[%d;%dH\x1b_Ga=T,f=%d,o=z,s=%d,v=%d,i=%d,p=%d,c=%d,r=%d,z=-1,C=1,q=2",
		at.Row+1, at.Col+1, 8*bpp, img.Rect.Dx(), img.Rect.Dy(), id, placement, at.Cols, at.Rows)
	for start := 0; start < len(k.file); start += graphics.KittyRawChunk {
		if start > 0 {
			dst = append(dst, "\x1b_Gq=2"...)
		}
		end := min(start+graphics.KittyRawChunk, len(k.file))
		dst = fmt.Appendf(dst, ",m=%d;", min(len(k.file)-end, 1))
		dst = base64.StdEncoding.AppendEncode(dst, k.file[start:end])
		dst = append(dst, "\x1b\\"...)
	}
	return dst
}

func KittyDelete(dst []byte, id uint32) []byte {
	return fmt.Appendf(dst, "\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", id)
}
