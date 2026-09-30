package graphics

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"

	"github.com/twind-dev/twind/internal/konst/graphics"
)

type ITerm struct {
	file []byte
	idat []byte
	z    deflater
}

func pngChunk(file []byte, start int) []byte {
	binary.BigEndian.PutUint32(file[start:], uint32(len(file)-start-8))
	return binary.BigEndian.AppendUint32(file, crc32.ChecksumIEEE(file[start+4:]))
}

func (i *ITerm) Encode(dst []byte, img *image.RGBA, at Placement) ([]byte, error) {
	if img.Rect.Empty() {
		return dst, nil
	}
	img = i.z.prepare(img)
	var bpp int
	i.idat, bpp = i.z.zlib(i.idat[:0], img, pngUpRows)
	colour := byte(graphics.PNGAlpha)
	if bpp == 3 {
		colour = graphics.PNGOpaque
	}
	i.file = append(i.file[:0], graphics.PNGSignature+"\x00\x00\x00\x00IHDR"...)
	i.file = binary.BigEndian.AppendUint32(i.file, uint32(img.Rect.Dx()))
	i.file = binary.BigEndian.AppendUint32(i.file, uint32(img.Rect.Dy()))
	i.file = pngChunk(append(i.file, 8, colour, 0, 0, 0), len(graphics.PNGSignature))
	start := len(i.file)
	i.file = pngChunk(append(append(i.file, "\x00\x00\x00\x00IDAT"...), i.idat...), start)
	start = len(i.file)
	i.file = pngChunk(append(i.file, "\x00\x00\x00\x00IEND"...), start)
	dst = fmt.Appendf(dst, "\x1b[%d;%dH\x1b]1337;File=inline=1;size=%d;width=%d;height=%d;preserveAspectRatio=0;doNotMoveCursor=1:",
		at.Row+1, at.Col+1, len(i.file), at.Cols, at.Rows)
	dst = base64.StdEncoding.AppendEncode(dst, i.file)
	return append(dst, '\a'), nil
}
