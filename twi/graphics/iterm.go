package graphics

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
)

type pngBuffer struct{ buffer *png.EncoderBuffer }

func (p *pngBuffer) Get() *png.EncoderBuffer  { return p.buffer }
func (p *pngBuffer) Put(b *png.EncoderBuffer) { p.buffer = b }

type ITerm struct {
	file  bytes.Buffer
	reuse pngBuffer
}

func (i *ITerm) Encode(dst []byte, img *image.RGBA, at Placement) ([]byte, error) {
	if img.Rect.Empty() {
		return dst, nil
	}
	if one := flat(img); one != nil {
		img = one
	}
	i.file.Reset()
	encoder := png.Encoder{CompressionLevel: png.BestSpeed, BufferPool: &i.reuse}
	if err := encoder.Encode(&i.file, img); err != nil {
		return dst, err
	}
	dst = fmt.Appendf(dst, "\x1b[%d;%dH\x1b]1337;File=inline=1;size=%d;width=%d;height=%d;preserveAspectRatio=0;doNotMoveCursor=1:",
		at.Row+1, at.Col+1, i.file.Len(), at.Cols, at.Rows)
	dst = base64.StdEncoding.AppendEncode(dst, i.file.Bytes())
	return append(dst, '\a'), nil
}
