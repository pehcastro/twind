package graphics

import (
	"encoding/binary"
	"hash/crc32"

	"github.com/twind-dev/twind/internal/konst/graphics"
)

type ITerm struct {
	file  []byte
	z     deflater
	pairs base64Pairs
}

func pngChunk(file []byte, start int) []byte {
	binary.BigEndian.PutUint32(file[start:], uint32(len(file)-start-8))
	return binary.BigEndian.AppendUint32(file, crc32.ChecksumIEEE(file[start+4:]))
}

func (i *ITerm) Encode(dst []byte, rows [][]byte, at Placement) []byte {
	if len(rows) == 0 || len(rows[0]) == 0 {
		return dst
	}
	rows, _ = i.z.prepare(rows)
	i.file = append(i.file[:0], graphics.PNGSignature+graphics.PNGHeader...)
	i.file = binary.BigEndian.AppendUint32(i.file, uint32(len(rows[0])/4))
	i.file = binary.BigEndian.AppendUint32(i.file, uint32(len(rows)))
	i.file = append(append(i.file, graphics.PNGDepth, graphics.PNGAlpha, 0, 0, 0), "\x00\x00\x00\x00\x00\x00\x00\x00IDAT"...)
	idat := len(i.file) - 8
	file, bpp := i.z.zlib(i.file, rows, pngUpRows)
	if bpp == 3 {
		file[idat-8] = graphics.PNGOpaque
	}
	binary.BigEndian.PutUint32(file[idat-4:], crc32.ChecksumIEEE(file[len(graphics.PNGSignature)+4:idat-4]))
	file = pngChunk(file, idat)
	i.file = pngChunk(append(file, "\x00\x00\x00\x00IEND"...), len(file))
	dst = field(field(dst, "\x1b[", at.Row+1), ";", at.Col+1)
	dst = field(field(field(dst, "H\x1b]1337;File=inline=1;size=", len(i.file)), ";width=", at.Cols), ";height=", at.Rows)
	dst = append(dst, ";preserveAspectRatio=0;doNotMoveCursor=1:"...)
	dst = i.pairs.appendEncode(dst, i.file)
	return append(dst, '\a')
}
