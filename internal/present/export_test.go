package present

import (
	"image"
	"testing"
)

type Xterm = xterm

func NewXterm(size, cell image.Point) *Xterm { return sizedXterm(size, cell) }

func (m *xterm) Feed(t *testing.T, p []byte) { m.write(t, p) }

func (m *xterm) Pixels(t *testing.T) []byte {
	img, _ := m.shown(t)
	return img.Pix
}
