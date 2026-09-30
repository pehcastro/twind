package graphics

import "image"

type Placement struct{ Col, Row, Cols, Rows int }

func row(img *image.RGBA, y int) []byte {
	at := img.PixOffset(img.Rect.Min.X, img.Rect.Min.Y+y)
	return img.Pix[at : at+4*img.Rect.Dx()]
}
