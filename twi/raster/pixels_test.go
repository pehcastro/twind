package raster

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

type pixelCase struct {
	name string
	img  *image.RGBA
}

func page(first Op, ops ...Op) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 480, 230))
	new(Raster).Draw(img, append([]Op{first}, ops...), img.Rect)
	return img
}

func pixelCases() []pixelCase {
	track := radius(20, 10, 160, 24, 1<<20)
	full := radius(0, 0, 200, 50, 0)
	plain := Op{Kind: Fill, Box: radius(0, 0, 480, 230, 0), Color: hex(0xfafafa, 255)}
	wash := Op{Kind: Fill, Box: radius(0, 0, 480, 230, 0), Angle: 180, Stops: []Stop{{Color: hex(0x0ea5e9, 255), At: 0}, {Color: hex(0xf43f5e, 255), At: 1}}}
	uneven := Box{Rect: Rect{20, 20, 440, 168}, Radii: [4]float64{4, 16, 0, 10}}
	top := Box{Rect: Rect{20.5, 30.25, 300, 120}, Radii: [4]float64{12, 12, 0, 0}}
	ops := append(sheet(), Op{Kind: Clip, Box: radius(300, 380, 150, 60, 0)},
		Op{Kind: Fill, Box: radius(250, 360, 300, 100, 20), Color: hex(0x10b981, 200)},
		Op{Kind: Pop},
		Op{Kind: Clip, Box: radius(260, 370, 200, 80, 30)},
		Op{Kind: Opacity, Opacity: 0.7},
		Op{Kind: Fill, Box: radius(240, 360, 150, 100, 0), Color: hex(0x3b82f6, 255)},
		Op{Kind: Pop},
		Op{Kind: Pop},
		Op{Kind: Border, Box: radius(40, 40, 200, 120, 4), Width: 1, Color: black, Dash: Dashed},
		Op{Kind: Fill, Box: radius(700, 200, 1, 300, 0), Color: black, Dash: Dotted})
	sheetImg, tiles := image.NewRGBA(image.Rect(0, 0, 940, 560)), image.NewRGBA(image.Rect(0, 0, 940, 560))
	new(Raster).Draw(sheetImg, ops, sheetImg.Rect)
	var shared Raster
	for _, t := range []image.Rectangle{image.Rect(37, 23, 211, 157), image.Rect(270, 350, 480, 470), image.Rect(700, 180, 940, 560)} {
		shared.Draw(tiles, ops, t)
	}
	cases := []pixelCase{
		{"card", page(plain, card(radius(20, 20, 440, 168, 12), white, hex(0xe4e4e7, 255), tailwind("md"))...)},
		{"sheet", sheetImg},
		{"tiles", tiles},
		{"uneven", page(plain, append(card(uneven, white, hex(0xe4e4e7, 255), tailwind("md")),
			Op{Kind: Shadow, Box: uneven, Color: hex(0, 40), Shadow: BoxShadow{Y: 2, Blur: 4, Inset: true}})...)},
		{"top", page(wash, append(card(top, hex(0xffffff, 230), hex(0x3f3f46, 160), tailwind("lg")),
			Op{Kind: Shadow, Box: top, Color: hex(0, 60), Shadow: BoxShadow{X: 3, Y: 4, Spread: 2}})...)},
		{"backdrop", page(plain, append(card(radius(40.5, 30, 300, 100, 8), white, hex(0xe4e4e7, 255), tailwind("xl")),
			Op{Kind: Fill, Box: radius(0, 0, 480, 230, 0), Color: hex(0, 128)},
			Op{Kind: Fill, Box: radius(100, 60, 200, 90, 1<<20), Color: hex(0x18181b, 255)})...)},
		{"clip-parent", render(200, 50, Op{Kind: Fill, Box: track, Color: black})},
		{"clip-filled", render(200, 50, Op{Kind: Clip, Box: track}, Op{Kind: Fill, Box: full, Color: black}, Op{Kind: Pop})},
		{"clip-bar", render(200, 50, Op{Kind: Clip, Box: track}, Op{Kind: Fill, Box: radius(20, 10, 96, 24, 0), Color: black}, Op{Kind: Pop})},
		{"clip-nested", render(200, 50, Op{Kind: Opacity, Opacity: 0.5}, Op{Kind: Clip, Box: track}, Op{Kind: Fill, Box: full, Color: black}, Op{Kind: Pop}, Op{Kind: Pop})},
		{"clip-faded", render(200, 50, Op{Kind: Opacity, Opacity: 0.5}, Op{Kind: Fill, Box: track, Color: black}, Op{Kind: Pop})},
		{"clip-inside", render(200, 50, Op{Kind: Clip, Box: track}, Op{Kind: Opacity, Opacity: 0.5}, Op{Kind: Fill, Box: full, Color: black}, Op{Kind: Pop}, Op{Kind: Pop})},
	}
	for _, d := range []Dash{Dashed, Dotted} {
		box := radius(20.5, 20.25, 120, 60, 8)
		cases = append(cases,
			pixelCase{fmt.Sprintf("dash%d-square", d), render(130, 70, Op{Kind: Border, Box: radius(10, 10, 101, 41, 0), Width: 1, Color: black, Dash: d})},
			pixelCase{fmt.Sprintf("dash%d-round", d), render(160, 100, Op{Kind: Border, Box: box, Width: 1, Color: black, Dash: d})},
			pixelCase{fmt.Sprintf("dash%d-thick", d), render(160, 100, Op{Kind: Border, Box: box, Width: 3, Color: black, Dash: d})},
			pixelCase{fmt.Sprintf("dash%d-line", d), render(130, 10, Op{Kind: Fill, Box: radius(10, 4, 101, 1, 0), Color: black, Dash: d})})
	}
	for i, s := range surfacesPage {
		img := image.NewRGBA(image.Rectangle{Max: s.size})
		shared.Draw(img, s.ops, img.Rect)
		cases = append(cases, pixelCase{fmt.Sprintf("surfaces%02d", i), img})
	}
	return cases
}

func TestPixels(t *testing.T) {
	out := os.Getenv("TWIND_RASTER_PIXELS")
	for _, c := range pixelCases() {
		if out != "" {
			var file bytes.Buffer
			if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&file, &image.NRGBA{Pix: c.img.Pix, Stride: c.img.Stride, Rect: c.img.Rect}); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(out, c.name+".png"), file.Bytes(), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		file, err := os.ReadFile(filepath.Join("testdata", "pixels", c.name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		old, err := png.Decode(bytes.NewReader(file))
		if err != nil {
			t.Fatal(err)
		}
		var want []uint8
		switch o := old.(type) {
		case *image.NRGBA:
			want = o.Pix
		case *image.RGBA:
			want = o.Pix
		default:
			t.Fatalf("%s: fixture decodes to %T, want 8-bit RGBA", c.name, old)
		}
		if old.Bounds() != c.img.Rect {
			t.Fatalf("%s: bounds %v, fixture %v", c.name, c.img.Rect, old.Bounds())
		}
		for y := range c.img.Rect.Dy() {
			for x := range c.img.Rect.Dx() {
				i := c.img.PixOffset(x, y)
				if got, w := c.img.Pix[i:i+4], want[i:i+4]; !bytes.Equal(got, w) {
					t.Fatalf("%s at (%d,%d): premultiplied %v, 9e1d4d3 drew %v", c.name, x, y, got, w)
				}
			}
		}
	}
}
