package raster

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/twind-dev/twind/twi/color"
)

func hex(v uint32, alpha uint8) color.RGBA {
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: alpha}
}

var (
	white = hex(0xffffff, 255)
	black = hex(0x000000, 255)
)

func radius(x, y, w, h, r float64) Box {
	return Box{Rect: Rect{x, y, w, h}, Radii: [4]float64{r, r, r, r}}
}

func render(w, h int, ops ...Op) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	page := Op{Kind: Fill, Box: radius(0, 0, float64(w), float64(h), 0), Color: white}
	new(Raster).Draw(img, append([]Op{page}, ops...), img.Bounds())
	return img
}

func at(img *image.RGBA, x, y int) color.RGBA {
	c := img.RGBAAt(x, y)
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

func darkness(img *image.RGBA, x, y int) float64 {
	return 1 - float64(img.RGBAAt(x, y).R)/255
}

func near(a, b color.RGBA, tolerance int) bool {
	d := func(p, q uint8) bool { return int(p)-int(q) <= tolerance && int(q)-int(p) <= tolerance }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B) && d(a.A, b.A)
}

func TestRect(t *testing.T) {
	aligned := render(120, 60, Op{Kind: Fill, Box: radius(10, 10, 100, 40, 8), Color: black})
	for _, p := range []struct {
		x, y int
		want float64
	}{{9, 30, 0}, {10, 30, 1}, {109, 30, 1}, {110, 30, 0}, {60, 9, 0}, {60, 10, 1}, {60, 49, 1}, {60, 50, 0}} {
		if got := darkness(aligned, p.x, p.y); got != p.want {
			t.Errorf("aligned rect at (%d,%d): coverage %.3f, want %v", p.x, p.y, got, p.want)
		}
	}

	origin := 12.5 - (8 - 8/math.Sqrt2)
	img := render(130, 70, Op{Kind: Fill, Box: radius(origin, origin, 100, 40, 8), Color: black})
	if got := at(img, int(origin+50), int(origin+20)); got != black {
		t.Errorf("centre %v, want opaque black", got)
	}
	right, bottom := origin+100, origin+40
	for _, p := range [][2]float64{{origin - 1, origin + 20}, {right + 1, origin + 20}, {origin + 50, origin - 1}, {origin + 50, bottom + 1}} {
		if got := at(img, int(p[0]), int(p[1])); got != white {
			t.Errorf("1 px outside the edge at (%v,%v): %v, want white", int(p[0]), int(p[1]), got)
		}
	}
	if c := darkness(img, 12, 12); c < 0.3 || c > 0.7 {
		t.Errorf("45 degree arc pixel (12,12): coverage %.3f, want 0.3 to 0.7", c)
	}
}

func TestShadow(t *testing.T) {
	box := radius(50, 40, 100, 40, 8)
	blur := 8.0
	sigma := blur / 2
	img := render(200, 120, Op{Kind: Shadow, Box: box, Color: black, Shadow: BoxShadow{Blur: blur}})
	previous := 1.0
	for x := 150; x < 180; x++ {
		d := float64(x) + 0.5 - 150
		got := darkness(img, x, 60)
		if got > previous {
			t.Errorf("x=%d: %.4f rises after %.4f", x, got, previous)
		}
		if want := 0.5 * math.Erfc(d/sigma/math.Sqrt2); math.Abs(got-want) > 0.02 {
			t.Errorf("x=%d, %.1f px out: alpha %.4f, want Gaussian sigma %v: %.4f", x, d, got, sigma, want)
		}
		previous = got
	}
	if edge := darkness(img, 150, 60); edge < 0.3 {
		t.Errorf("alpha at the edge %.3f, the shadow is missing", edge)
	}
	if far := darkness(img, 150+int(3*sigma), 60); far >= 0.02 {
		t.Errorf("alpha at 3 sigma %.4f, want under 0.02", far)
	}
	previous = 1.0
	for s := 6; s < 26; s++ {
		got := darkness(img, 142+s, 72+s)
		if got > previous {
			t.Errorf("corner diagonal step %d: %.4f rises after %.4f", s, got, previous)
		}
		previous = got
	}
	if got := at(img, 100, 60); got != white {
		t.Errorf("outer shadow painted inside the box: %v", got)
	}

	inset := render(200, 120, Op{Kind: Shadow, Box: box, Color: black, Shadow: BoxShadow{Blur: blur, Inset: true}})
	inside := 0
	for y := range 120 {
		for x := range 200 {
			in := x >= 50 && x < 150 && y >= 40 && y < 80
			got := at(inset, x, y)
			if !in && got != white {
				t.Fatalf("inset shadow outside the box at (%d,%d): %v", x, y, got)
			}
			if in && got != white {
				inside++
			}
		}
	}
	if darkness(inset, 50, 60) < 0.3 || inside == 0 {
		t.Errorf("inset shadow does not darken the inside edge: %.3f, %d pixels", darkness(inset, 50, 60), inside)
	}
}

func TestGradient(t *testing.T) {
	indigo, _ := color.Parse("oklch(51.1% 0.262 276.966)")
	pink, _ := color.Parse("oklch(59.2% 0.249 0.584)")
	img := render(101, 10, Op{Kind: Fill, Box: radius(0, 0, 101, 10, 0), Angle: 90,
		Stops: []Stop{{Color: indigo.RGBA, At: 0}, {Color: pink.RGBA, At: 1}}})
	lab := func(c color.RGBA) [3]float64 {
		var rgb, lms [3]float64
		for i, v := range []uint8{c.R, c.G, c.B} {
			rgb[i] = math.Pow((float64(v)/255+0.055)/1.055, 2.4)
		}
		toXYZ := [3][3]float64{{0.4124564, 0.3575761, 0.1804375}, {0.2126729, 0.7151522, 0.0721750}, {0.0193339, 0.1191920, 0.9503041}}
		toLMS := [3][3]float64{{0.8189330101, 0.3618667424, -0.1288597137}, {0.0329845436, 0.9293118715, 0.0361456387}, {0.0482003018, 0.2643662691, 0.6338517070}}
		toLab := [3][3]float64{{0.2104542553, 0.7936177850, -0.0040720468}, {1.9779984951, -2.4285922050, 0.4505937099}, {0.0259040371, 0.7827717662, -0.8086757660}}
		mul := func(m [3][3]float64, v [3]float64) (out [3]float64) {
			for i := range out {
				out[i] = m[i][0]*v[0] + m[i][1]*v[1] + m[i][2]*v[2]
			}
			return out
		}
		lms = mul(toLMS, mul(toXYZ, rgb))
		for i := range lms {
			lms[i] = math.Cbrt(lms[i])
		}
		return mul(toLab, lms)
	}
	p, q := lab(indigo.RGBA), lab(pink.RGBA)
	l, a, b := (p[0]+q[0])/2, (p[1]+q[1])/2, (p[2]+q[2])/2
	mid, err := color.Parse(fmt.Sprintf("oklch(%f %f %f)", l, math.Hypot(a, b), math.Atan2(b, a)*180/math.Pi))
	if err != nil {
		t.Fatal(err)
	}
	if got := at(img, 50, 5); !near(got, mid.RGBA, 1) {
		t.Errorf("centre column %v, want OKLab midpoint %v", got, mid.RGBA)
	}
	if got := at(img, 0, 5); !near(got, indigo.RGBA, 2) {
		t.Errorf("left column %v, want indigo %v", got, indigo.RGBA)
	}
}

func TestOpacity(t *testing.T) {
	img := render(100, 20,
		Op{Kind: Opacity, Opacity: 0.5},
		Op{Kind: Fill, Box: radius(0, 0, 60, 20, 0), Color: hex(0xff0000, 255)},
		Op{Kind: Fill, Box: radius(40, 0, 60, 20, 0), Color: hex(0x0000ff, 255)},
		Op{Kind: Pop},
	)
	for _, p := range []struct {
		x    int
		want color.RGBA
	}{{10, hex(0xff8080, 255)}, {50, hex(0x8080ff, 255)}, {90, hex(0x8080ff, 255)}} {
		if got := at(img, p.x, 10); !near(got, p.want, 1) {
			t.Errorf("x=%d: %v, want %v", p.x, got, p.want)
		}
	}
}

func TestMean(t *testing.T) {
	img := render(20, 10, Op{Kind: Fill, Box: radius(0, 0, 10, 10, 0), Color: black})
	if got := Mean(img, image.Rect(0, 0, 20, 10)); !near(got, hex(0x808080, 255), 1) {
		t.Errorf("mean of half black half white %v, want grey", got)
	}
	if got := Mean(img, image.Rect(10, 0, 20, 10)); got != white {
		t.Errorf("mean of white %v", got)
	}
}

func TestTile(t *testing.T) {
	ops := append(sheet(), Op{Kind: Clip, Box: radius(300, 380, 150, 60, 0)},
		Op{Kind: Fill, Box: radius(250, 360, 300, 100, 20), Color: hex(0x10b981, 200)},
		Op{Kind: Pop})
	bounds := image.Rect(0, 0, 940, 560)
	full := image.NewRGBA(bounds)
	new(Raster).Draw(full, ops, bounds)
	var r Raster
	for _, tile := range []image.Rectangle{image.Rect(37, 23, 211, 157), image.Rect(270, 350, 480, 470), image.Rect(700, 180, 940, 560)} {
		part := image.NewRGBA(bounds)
		r.Draw(part, ops, tile)
		for y := range 560 {
			for x := range 940 {
				want := color.RGBA{}
				if (image.Point{x, y}).In(tile) {
					want = at(full, x, y)
				}
				if c := at(part, x, y); c != want {
					t.Fatalf("tile %v at (%d,%d): %v, full raster %v", tile, x, y, c, want)
				}
			}
		}
	}
}

func tailwind(size string) []BoxShadow {
	return map[string][]BoxShadow{
		"sm": {{0, 1, 3, 0, false}, {0, 1, 2, -1, false}},
		"md": {{0, 4, 6, -1, false}, {0, 2, 4, -2, false}},
		"lg": {{0, 10, 15, -3, false}, {0, 4, 6, -4, false}},
		"xl": {{0, 20, 25, -5, false}, {0, 8, 10, -6, false}},
	}[size]
}

func card(box Box, fill, border color.RGBA, shadows []BoxShadow) []Op {
	var ops []Op
	for i := len(shadows) - 1; i >= 0; i-- {
		ops = append(ops, Op{Kind: Shadow, Box: box, Color: hex(0, 26), Shadow: shadows[i]})
	}
	return append(ops, Op{Kind: Fill, Box: box, Color: fill}, Op{Kind: Border, Box: box, Width: 1, Color: border})
}

func sheet() []Op {
	page, surface, line := hex(0xfafafa, 255), hex(0xffffff, 255), hex(0xe4e4e7, 255)
	ops := []Op{{Kind: Fill, Box: radius(0, 0, 940, 560, 0), Color: page}}
	for i, r := range []float64{4, 6, 8, 12} {
		ops = append(ops, card(radius(40+float64(i)*220, 40, 200, 120, r), surface, line, tailwind("sm"))...)
	}
	for i, s := range []string{"sm", "md", "lg", "xl"} {
		ops = append(ops, card(radius(40+float64(i)*220, 210, 200, 120, 12), surface, line, tailwind(s))...)
	}
	indigo, _ := color.Parse("oklch(51.1% 0.262 276.966)")
	pink, _ := color.Parse("oklch(59.2% 0.249 0.584)")
	ops = append(ops, Op{Kind: Fill, Box: radius(40, 400, 420, 80, 8), Angle: 90,
		Stops: []Stop{{Color: indigo.RGBA, At: 0}, {Color: pink.RGBA, At: 1}}})
	ops = append(ops,
		Op{Kind: Opacity, Opacity: 0.5},
		Op{Kind: Fill, Box: radius(490, 400, 90, 60, 6), Color: hex(0xef4444, 255)},
		Op{Kind: Fill, Box: radius(530, 420, 90, 60, 6), Color: hex(0x3b82f6, 255)},
		Op{Kind: Pop})
	ops = append(ops, card(radius(650, 400, 120, 36, 999), hex(0x18181b, 255), hex(0x18181b, 255), tailwind("sm"))...)
	ops = append(ops, Op{Kind: Fill, Box: radius(650, 450, 250, 36, 6), Color: surface},
		Op{Kind: Shadow, Box: radius(651, 451, 248, 34, 5), Color: hex(0, 20), Shadow: BoxShadow{Y: 2, Blur: 4, Inset: true}},
		Op{Kind: Border, Box: radius(650, 450, 250, 36, 6), Width: 1, Color: line})
	return append(ops, card(radius(800, 390, 110, 44, 8), hex(0x27272a, 255), hex(0x3f3f46, 255), tailwind("lg"))...)
}

func TestSheet(t *testing.T) {
	bounds := image.Rect(0, 0, 940, 560)
	img := image.NewRGBA(bounds)
	new(Raster).Draw(img, sheet(), bounds)
	dir := os.Getenv("TWIND_SHEET")
	if dir == "" {
		dir = t.TempDir()
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "raster-sheet.png"), encoded.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := at(img, 140, 100); got != white {
		t.Errorf("card centre %v, want white", got)
	}
}

func BenchmarkCard(b *testing.B) {
	bounds := image.Rect(0, 0, 480, 230)
	img := image.NewRGBA(bounds)
	ops := append([]Op{{Kind: Fill, Box: radius(0, 0, 480, 230, 0), Color: hex(0xfafafa, 255)}},
		card(radius(20, 20, 440, 168, 12), white, hex(0xe4e4e7, 255), tailwind("md"))...)
	var r Raster
	r.Draw(img, ops, bounds)
	b.ReportAllocs()
	for b.Loop() {
		r.Draw(img, ops, bounds)
	}
}
