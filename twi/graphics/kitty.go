package graphics

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"

	"github.com/twind-dev/twind/internal/konst/graphics"
)

type Kitty struct {
	file []byte
	z    deflater
}

func (k *Kitty) Encode(dst []byte, rows [][]byte, at Placement, id, placement uint32) []byte {
	if len(rows) == 0 || len(rows[0]) == 0 {
		return dst
	}
	rows, flat := k.z.prepare(rows)
	bpp, compression := 4, ""
	if flat {
		p := straight(pixel(rows[0], 0))
		if p>>24 == 0xff {
			bpp = 3
		}
		k.file = binary.LittleEndian.AppendUint32(k.file[:0], p)[:bpp]
	} else {
		k.file, bpp = k.z.zlib(k.file[:0], rows, rawRows)
		compression = "o=z,"
	}
	dst = fmt.Appendf(dst, "\x1b[%d;%dH\x1b_Ga=T,f=%d,%ss=%d,v=%d,i=%d,p=%d,c=%d,r=%d,z=-1,C=1,q=2",
		at.Row+1, at.Col+1, 8*bpp, compression, len(rows[0])/4, len(rows), id, placement, at.Cols, at.Rows)
	for start := 0; start < len(k.file); start += graphics.KittyRawChunk {
		if start > 0 {
			dst = append(dst, "\x1b_Gq=2"...)
		}
		end := min(start+graphics.KittyRawChunk, len(k.file))
		if start > 0 || end < len(k.file) {
			dst = fmt.Appendf(dst, ",m=%d", min(len(k.file)-end, 1))
		}
		dst = append(dst, ';')
		dst = base64.StdEncoding.AppendEncode(dst, k.file[start:end])
		dst = append(dst, "\x1b\\"...)
	}
	return dst
}

func KittyPlace(dst []byte, at Placement, id, placement uint32) []byte {
	return fmt.Appendf(dst, "\x1b[%d;%dH\x1b_Ga=p,i=%d,p=%d,c=%d,r=%d,z=-1,C=1,q=2\x1b\\", at.Row+1, at.Col+1, id, placement, at.Cols, at.Rows)
}

func KittyUnplace(dst []byte, id, placement uint32) []byte {
	return fmt.Appendf(dst, "\x1b_Ga=d,d=i,i=%d,p=%d,q=2\x1b\\", id, placement)
}

func KittyDelete(dst []byte, id uint32) []byte {
	return fmt.Appendf(dst, "\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", id)
}
