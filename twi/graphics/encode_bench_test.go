package graphics

import (
	"bytes"
	"encoding/binary"
	"image"
	"testing"
)

func flatCard() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 440, 168))
	for i := 0; i < len(img.Pix); i += 4 {
		copy(img.Pix[i:], []byte{244, 244, 245, 255})
	}
	return img
}

func lines(img *image.RGBA) [][]byte {
	var rows [][]byte
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		rows = append(rows, img.Pix[img.PixOffset(img.Rect.Min.X, y):img.PixOffset(img.Rect.Max.X, y)])
	}
	return rows
}

func runs(img *image.RGBA) [][]Run {
	var rows [][]Run
	pixels := lines(img)
	for y, row := range pixels {
		if y > 0 && bytes.Equal(row, pixels[y-1]) {
			rows = append(rows, rows[y-1])
			continue
		}
		var line []Run
		for x := range len(row) / 4 {
			if px, n := binary.LittleEndian.Uint32(row[4*x:]), len(line); n > 0 && line[n-1].Pixel == px {
				line[n-1].End++
			} else {
				line = append(line, Run{End: int32(x + 1), Pixel: px})
			}
		}
		rows = append(rows, line)
	}
	return rows
}

func benchSixel(b *testing.B, img *image.RGBA) {
	var s Sixel
	var buf []byte
	rows := runs(img)
	b.ReportAllocs()
	for b.Loop() {
		buf = s.Encode(buf[:0], rows, Placement{Cols: 44, Rows: 8})
	}
	b.ReportMetric(float64(len(buf)), "bytes")
}

func benchKitty(b *testing.B, img *image.RGBA) {
	var k Kitty
	var buf []byte
	rows := lines(img)
	b.ReportAllocs()
	for b.Loop() {
		buf = k.Encode(buf[:0], rows, Placement{Cols: 44, Rows: 8}, 1, 1)
	}
	b.ReportMetric(float64(len(buf)), "bytes")
}

func benchITerm(b *testing.B, img *image.RGBA) {
	var enc ITerm
	var buf []byte
	rows := lines(img)
	b.ReportAllocs()
	for b.Loop() {
		buf = enc.Encode(buf[:0], rows, Placement{Cols: 44, Rows: 8})
	}
	b.ReportMetric(float64(len(buf)), "bytes")
}

func BenchmarkEncodeSixelCard(b *testing.B) { benchSixel(b, loadCard(b)) }
func BenchmarkEncodeSixelFlat(b *testing.B) { benchSixel(b, flatCard()) }
func BenchmarkEncodeKittyCard(b *testing.B) { benchKitty(b, loadCard(b)) }
func BenchmarkEncodeKittyFlat(b *testing.B) { benchKitty(b, flatCard()) }
func BenchmarkEncodeITermCard(b *testing.B) { benchITerm(b, loadCard(b)) }
func BenchmarkEncodeITermFlat(b *testing.B) { benchITerm(b, flatCard()) }
