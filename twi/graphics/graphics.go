package graphics

import (
	"encoding/base64"
	"encoding/binary"
	"slices"
	"strconv"

	"github.com/twind-dev/twind/internal/konst/graphics"
)

type Placement struct{ Col, Row, Cols, Rows int }

type Run struct {
	End   int32
	Pixel uint32
}

type base64Pairs [graphics.Base64Pairs]uint16

func field(dst []byte, name string, v int) []byte {
	return strconv.AppendInt(append(dst, name...), int64(v), 10)
}

func (t *base64Pairs) appendEncode(dst, src []byte) []byte {
	if t[1] == 0 {
		for i := range t {
			t[i] = uint16(graphics.Base64Alphabet[i>>6]) | uint16(graphics.Base64Alphabet[i&63])<<8
		}
	}
	dst = slices.Grow(dst, base64.StdEncoding.EncodedLen(len(src)))
	for ; len(src) >= 8; src = src[graphics.Base64Group:] {
		v := binary.BigEndian.Uint64(src)
		dst = binary.LittleEndian.AppendUint64(dst, uint64(t[v>>52])|uint64(t[v>>40&0xfff])<<16|uint64(t[v>>28&0xfff])<<32|uint64(t[v>>16&0xfff])<<48)
	}
	return base64.StdEncoding.AppendEncode(dst, src)
}
