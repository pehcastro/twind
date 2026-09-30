package graphics

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"io"
	"slices"
	"strings"
	"testing"
)

func straightPixels(img *image.RGBA, bpp int) []byte {
	var want []byte
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			want = append(want, []byte{c.R, c.G, c.B, c.A}[:bpp]...)
		}
	}
	return want
}

func roundTripKitty(t *testing.T, img *image.RGBA) {
	t.Helper()
	var k Kitty
	for range 2 {
		_, chunks, raw := kittyChunks(t, k.Encode(nil, lines(img), Placement{Cols: 4, Rows: 2}, 1, 1))
		bpp := 3
		if chunks[0].keys["f"] == "32" {
			bpp = 4
		}
		w, h := chunks[0].keys["s"], chunks[0].keys["v"]
		want := straightPixels(img, bpp)
		if w == "1" && h == "1" {
			want = want[:bpp]
		}
		if !bytes.Equal(raw, want) {
			t.Fatalf("kitty %v: %d decoded bytes differ from the %d straight-alpha bytes", img.Rect, len(raw), len(want))
		}
	}
}

func roundTripITerm(t *testing.T, img *image.RGBA) {
	t.Helper()
	var enc ITerm
	for range 2 {
		_, data, _ := strings.Cut(string(enc.Encode(nil, lines(img), Placement{Cols: 4, Rows: 2})), ":")
		file, err := base64.StdEncoding.DecodeString(strings.TrimSuffix(data, "\x07"))
		if err != nil {
			t.Fatal(err)
		}
		got, err := png.Decode(bytes.NewReader(file))
		if err != nil {
			t.Fatalf("iterm %v: %v", img.Rect, err)
		}
		var idat []byte
		for rest := file[8:]; len(rest) >= 12; {
			n := binary.BigEndian.Uint32(rest)
			if string(rest[4:8]) == "IDAT" {
				idat = append(idat, rest[8:8+n]...)
			}
			rest = rest[12+n:]
		}
		z, err := zlib.NewReader(bytes.NewReader(idat))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.ReadAll(z); err != nil {
			t.Fatalf("iterm %v: IDAT stream: %v", img.Rect, err)
		}
		b := got.Bounds()
		flat := b.Dx() == 1 && b.Dy() == 1
		if !flat && (b.Dx() != img.Rect.Dx() || b.Dy() != img.Rect.Dy()) {
			t.Fatalf("iterm %v: png is %v", img.Rect, b)
		}
		for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
			for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
				px, py := x-img.Rect.Min.X, y-img.Rect.Min.Y
				if flat {
					px, py = 0, 0
				}
				want := color.NRGBAModel.Convert(img.At(x, y))
				if have := color.NRGBAModel.Convert(got.At(b.Min.X+px, b.Min.Y+py)); have != want {
					t.Fatalf("iterm %v pixel %d,%d: want %v got %v", img.Rect, x, y, want, have)
				}
			}
		}
	}
}

func stripes(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		img.Set(x, 0, color.NRGBA{uint8(x * 7), uint8(x / 3), 90, 255})
	}
	for y := 1; y < h; y++ {
		copy(img.Pix[y*img.Stride:], img.Pix[:w*4])
	}
	return img
}

func roundTripImages(t *testing.T) map[string]*image.RGBA {
	card := loadCard(t)
	shadowed := image.NewRGBA(card.Rect)
	for i := 0; i < len(card.Pix); i += 4 {
		a := uint32(card.Pix[i]) * 3 / 4
		for c := range 3 {
			shadowed.Pix[i+c] = uint8(uint32(card.Pix[i+c]) * a / 255)
		}
		shadowed.Pix[i+3] = uint8(a)
	}
	return map[string]*image.RGBA{
		"card":          card,
		"card tile":     card.SubImage(image.Rect(3, 5, 431, 160)).(*image.RGBA),
		"card alpha":    shadowed,
		"flat":          flatCard(),
		"gradient":      gradient(100, 100, true),
		"gradient tile": gradient(120, 110, false).SubImage(image.Rect(10, 5, 110, 105)).(*image.RGBA),
		"noise":         noise(37, 23),
		"one pixel":     noise(1, 1),
	}
}

func TestSixelSharedRowsEncodeAsCopies(t *testing.T) {
	images := roundTripImages(t)
	images["stripes"] = stripes(97, 17)
	for name, img := range images {
		shared := runs(img)
		copied := make([][]Run, len(shared))
		for y, line := range shared {
			copied[y] = slices.Clone(line)
		}
		var s Sixel
		if got, want := string(s.Encode(nil, shared, Placement{Col: 2, Row: 3})), string(s.Encode(nil, copied, Placement{Col: 2, Row: 3})); got != want {
			t.Errorf("%s: rows sharing one slice encode to %d bytes, the same rows copied to %d, want the same bytes", name, len(got), len(want))
		}
	}
}

func TestRoundTripPixels(t *testing.T) {
	for name, img := range roundTripImages(t) {
		t.Run(name, func(t *testing.T) {
			roundTripKitty(t, img)
			roundTripITerm(t, img)
		})
	}
}

func TestRoundTripRunLengths(t *testing.T) {
	for w := 1; w <= 200; w++ {
		img := stripes(w, 3+w%5)
		roundTripKitty(t, img)
		roundTripITerm(t, img)
	}
}
