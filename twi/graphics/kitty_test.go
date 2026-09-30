package graphics

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"image"
	"image/color"
	"io"
	"strings"
	"testing"
)

type kittyChunk struct {
	keys    map[string]string
	payload string
}

func kittyChunks(t *testing.T, out []byte) (string, []kittyChunk, []byte) {
	t.Helper()
	s := string(out)
	i := strings.Index(s, "\x1b_G")
	if i < 0 {
		t.Fatalf("no APC in %q", s)
	}
	var chunks []kittyChunk
	var payload strings.Builder
	for _, part := range strings.Split(s[i:], "\x1b\\") {
		if part == "" {
			continue
		}
		body, ok := strings.CutPrefix(part, "\x1b_G")
		if !ok {
			t.Fatalf("junk between chunks: %q", part)
		}
		control, data, _ := strings.Cut(body, ";")
		keys := map[string]string{}
		for kv := range strings.SplitSeq(control, ",") {
			k, v, _ := strings.Cut(kv, "=")
			keys[k] = v
		}
		chunks = append(chunks, kittyChunk{keys, data})
		payload.WriteString(data)
	}
	compressed, err := base64.StdEncoding.DecodeString(payload.String())
	if err != nil {
		t.Fatal(err)
	}
	if chunks[0].keys["o"] != "z" {
		return s[:i], chunks, compressed
	}
	z, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(z)
	if err != nil {
		t.Fatal(err)
	}
	return s[:i], chunks, raw
}

func gradient(w, h int, alpha bool) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			a := uint8(255)
			if alpha {
				a = uint8(x * 255 / w)
			}
			img.Set(x, y, color.NRGBA{uint8(x * 2), uint8(y * 2), 200, a})
		}
	}
	return img
}

func noise(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	seed := uint32(1)
	for y := range h {
		for x := range w {
			seed = seed*1664525 + 1013904223
			img.Set(x, y, color.NRGBA{uint8(seed >> 24), uint8(seed >> 16), uint8(seed >> 8), uint8(seed)})
		}
	}
	return img
}

func checkKitty(t *testing.T, img *image.RGBA, out []byte, format string, bpp int) []kittyChunk {
	t.Helper()
	prefix, chunks, raw := kittyChunks(t, out)
	if prefix != "\x1b[3;2H" {
		t.Fatalf("cursor prefix %q", prefix)
	}
	first := chunks[0].keys
	for k, v := range map[string]string{"a": "T", "f": format, "o": "z", "s": "100", "v": "100", "i": "7", "p": "3", "c": "10", "r": "5", "z": "-1", "C": "1", "q": "2"} {
		if first[k] != v {
			t.Fatalf("first chunk %s=%q, want %q: %v", k, first[k], v, first)
		}
	}
	for n, c := range chunks {
		want := "1"
		switch {
		case len(chunks) == 1:
			want = ""
		case n == len(chunks)-1:
			want = "0"
		}
		if c.keys["m"] != want {
			t.Fatalf("chunk %d of %d: m=%q", n, len(chunks), c.keys["m"])
		}
		if n > 0 && len(c.keys) != 2 {
			t.Fatalf("continuation chunk %d carries %v", n, c.keys)
		}
		if len(c.payload) == 0 || len(c.payload) > 4096 || len(c.payload)%4 != 0 {
			t.Fatalf("chunk %d payload length %d", n, len(c.payload))
		}
	}
	var want []byte
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			want = append(want, []byte{c.R, c.G, c.B, c.A}[:bpp]...)
		}
	}
	if !bytes.Equal(raw, want) {
		t.Fatalf("payload decodes to %d bytes that differ from the %d straight-alpha pixel bytes", len(raw), len(want))
	}
	return chunks
}

func TestKittyChunksRGBA(t *testing.T) {
	img := noise(100, 100)
	var k Kitty
	for range 2 {
		chunks := checkKitty(t, img, k.Encode(nil, lines(img), Placement{Col: 1, Row: 2, Cols: 10, Rows: 5}, 7, 3), "32", 4)
		if len(chunks) < 10 {
			t.Fatalf("%d chunks for 40000 bytes of noise", len(chunks))
		}
	}
}

func TestKittyGradientAlphaAndOpaqueSubImage(t *testing.T) {
	var k Kitty
	img := gradient(100, 100, true)
	checkKitty(t, img, k.Encode(nil, lines(img), Placement{Col: 1, Row: 2, Cols: 10, Rows: 5}, 7, 3), "32", 4)
	tile := gradient(120, 110, false).SubImage(image.Rect(10, 5, 110, 105)).(*image.RGBA)
	checkKitty(t, tile, k.Encode(nil, lines(tile), Placement{Col: 1, Row: 2, Cols: 10, Rows: 5}, 7, 3), "24", 3)
}

func TestKittyFlatIsOneRawPixel(t *testing.T) {
	var k Kitty
	translucent := flatCard()
	for i := 0; i < len(translucent.Pix); i += 4 {
		copy(translucent.Pix[i:], []byte{61, 61, 61, 128})
	}
	for _, tc := range []struct {
		name   string
		img    *image.RGBA
		format string
		pixel  []byte
		size   int
	}{
		{"opaque", flatCard(), "24", []byte{244, 244, 245}, 62},
		{"translucent", translucent, "32", []byte{121, 121, 121, 128}, 66},
	} {
		out := k.Encode(k.Encode(nil, lines(noise(40, 40)), Placement{Cols: 4, Rows: 2}, 1, 1)[:0], lines(tc.img), Placement{Cols: 44, Rows: 8}, 1, 1)
		_, chunks, raw := kittyChunks(t, out)
		c := chunks[0].keys
		_, compressed := c["o"]
		_, more := c["m"]
		if len(chunks) != 1 || compressed || more || c["s"] != "1" || c["v"] != "1" || c["c"] != "44" || c["r"] != "8" || c["f"] != tc.format || !bytes.Equal(raw, tc.pixel) || len(out) != tc.size {
			t.Errorf("%s flat tile: %d bytes %v %v", tc.name, len(out), c, raw)
		}
	}
}

func TestKittyDelete(t *testing.T) {
	if got := string(KittyDelete(nil, 7)); got != "\x1b_Ga=d,d=I,i=7,q=2\x1b\\" {
		t.Fatalf("%q", got)
	}
}

func TestKittyPlacements(t *testing.T) {
	if got := string(KittyPlace(nil, Placement{Col: 8, Row: 2, Cols: 8, Rows: 1}, 7, 43)); got != "\x1b[3;9H\x1b_Ga=p,i=7,p=43,c=8,r=1,z=-1,C=1,q=2\x1b\\" {
		t.Errorf("place: %q", got)
	}
	if got := string(KittyUnplace(nil, 7, 43)); got != "\x1b_Ga=d,d=i,i=7,p=43,q=2\x1b\\" {
		t.Errorf("unplace: %q", got)
	}
}
