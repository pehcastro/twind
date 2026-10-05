package graphics

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strconv"
	"strings"
	"testing"
)

func TestITermHeaderAndPNG(t *testing.T) {
	full := gradient(60, 50, true)
	tile := full.SubImage(image.Rect(10, 10, 40, 30)).(*image.RGBA)
	var enc ITerm
	enc.Encode(nil, lines(flatCard()), Placement{})
	s := string(enc.Encode(nil, lines(tile), Placement{Col: 4, Row: 0, Cols: 3, Rows: 1}))
	head, data, ok := strings.Cut(s, ":")
	if !ok || !strings.HasSuffix(data, "\x07") {
		t.Fatalf("not an OSC 1337 File: %q", s)
	}
	wantHead := "\x1b[1;5H\x1b]1337;File=inline=1;size=%d;width=3;height=1;preserveAspectRatio=0;doNotMoveCursor=1"
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSuffix(data, "\x07"))
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Replace(wantHead, "%d", strconv.Itoa(len(raw)), 1); head != want {
		t.Fatalf("\nhead %q\nwant %q", head, want)
	}
	got, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds().Dx() != 30 || got.Bounds().Dy() != 20 {
		t.Fatalf("png size %v", got.Bounds())
	}
	for y := range 20 {
		for x := range 30 {
			want := color.NRGBAModel.Convert(tile.At(x+10, y+10))
			if got := color.NRGBAModel.Convert(got.At(x, y)); got != want {
				t.Fatalf("pixel %d,%d: want %v got %v", x, y, want, got)
			}
		}
	}
}

func TestITermFlatIsOnePixel(t *testing.T) {
	var enc ITerm
	out := enc.Encode(nil, lines(flatCard()), Placement{Cols: 44, Rows: 8})
	_, data, _ := strings.Cut(string(out), ":")
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSuffix(data, "\x07"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds().Dx() != 1 || got.Bounds().Dy() != 1 || !strings.Contains(string(out), "width=44;height=8;") {
		t.Fatalf("flat tile sent as %v: %q", got.Bounds(), out[:60])
	}
}
