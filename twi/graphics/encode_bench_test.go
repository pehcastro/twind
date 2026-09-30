package graphics

import (
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

func benchSixel(b *testing.B, img *image.RGBA) {
	var s Sixel
	var buf []byte
	b.ReportAllocs()
	for b.Loop() {
		buf = s.Encode(buf[:0], img, Placement{Cols: 44, Rows: 8})
	}
	b.ReportMetric(float64(len(buf)), "bytes")
}

func benchKitty(b *testing.B, img *image.RGBA) {
	var k Kitty
	var buf []byte
	b.ReportAllocs()
	for b.Loop() {
		buf = k.Encode(buf[:0], img, Placement{Cols: 44, Rows: 8}, 1, 1)
	}
	b.ReportMetric(float64(len(buf)), "bytes")
}

func benchITerm(b *testing.B, img *image.RGBA) {
	var enc ITerm
	var buf []byte
	b.ReportAllocs()
	for b.Loop() {
		var err error
		if buf, err = enc.Encode(buf[:0], img, Placement{Cols: 44, Rows: 8}); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(len(buf)), "bytes")
}

func BenchmarkEncodeSixelCard(b *testing.B) { benchSixel(b, loadCard(b)) }
func BenchmarkEncodeSixelFlat(b *testing.B) { benchSixel(b, flatCard()) }
func BenchmarkEncodeKittyCard(b *testing.B) { benchKitty(b, loadCard(b)) }
func BenchmarkEncodeKittyFlat(b *testing.B) { benchKitty(b, flatCard()) }
func BenchmarkEncodeITermCard(b *testing.B) { benchITerm(b, loadCard(b)) }
func BenchmarkEncodeITermFlat(b *testing.B) { benchITerm(b, flatCard()) }
