package graphics

import (
	"bytes"
	"image"
)

type Placement struct{ Col, Row, Cols, Rows int }

func row(img *image.RGBA, y int) []byte {
	at := img.PixOffset(img.Rect.Min.X, img.Rect.Min.Y+y)
	return img.Pix[at : at+4*img.Rect.Dx()]
}

func flat(img *image.RGBA) *image.RGBA {
	first := row(img, 0)[:4]
	for y := range img.Rect.Dy() {
		r := row(img, y)
		if !bytes.Equal(r[:4], first) || !bytes.Equal(r[4:], r[:len(r)-4]) {
			return nil
		}
	}
	return &image.RGBA{Pix: first, Stride: 4, Rect: image.Rect(0, 0, 1, 1)}
}
