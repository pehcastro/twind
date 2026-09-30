package graphics

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strconv"
	"strings"
	"testing"
)

type decodedSixel struct {
	img       *image.RGBA
	registers int
	body      int
}

func decodeSixel(t *testing.T, out []byte) decodedSixel {
	t.Helper()
	s := string(out)
	start := strings.Index(s, "\x1bP0;1q\"1;1;")
	if start < 0 || !strings.HasSuffix(s, "\x1b\\") {
		t.Fatalf("not a sixel DCS: %q", s)
	}
	s = s[start+len("\x1bP0;1q\"1;1;") : len(s)-2]
	num := func() int {
		i := 0
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		n, err := strconv.Atoi(s[:i])
		if err != nil {
			t.Fatalf("number expected at %q", s)
		}
		s = s[i:]
		return n
	}
	w := num()
	s = strings.TrimPrefix(s, ";")
	h := num()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	palette := map[int]color.RGBA{}
	var d decodedSixel
	cur, x, y, repeat := 0, 0, 0, 1
	for len(s) > 0 {
		c := s[0]
		switch {
		case c == '#':
			s = s[1:]
			cur = num()
			if strings.HasPrefix(s, ";2;") {
				s = s[3:]
				var ch [3]uint8
				for i := range ch {
					ch[i] = uint8((num()*255 + 50) / 100)
					s = strings.TrimPrefix(s, ";")
				}
				palette[cur] = color.RGBA{ch[0], ch[1], ch[2], 255}
				d.registers = len(palette)
				d.body = len(s)
			}
		case c == '!':
			s = s[1:]
			repeat = num()
		case c == '$':
			x, s = 0, s[1:]
		case c == '-':
			x, y, s = 0, y+6, s[1:]
		case c >= '?' && c <= '~':
			bits := c - '?'
			for range repeat {
				for b := range 6 {
					if bits&(1<<b) == 0 {
						continue
					}
					if x >= w || y+b >= h {
						t.Fatalf("sixel bit outside %dx%d at %d,%d", w, h, x, y+b)
					}
					p, ok := palette[cur]
					if !ok {
						t.Fatalf("register %d used before it is defined", cur)
					}
					img.SetRGBA(x, y+b, p)
				}
				x++
			}
			repeat, s = 1, s[1:]
		default:
			t.Fatalf("unexpected sixel byte %q", c)
		}
	}
	d.img = img
	return d
}

func loadCard(t testing.TB) *image.RGBA {
	t.Helper()
	file, err := os.ReadFile("testdata/card.png")
	if err != nil {
		t.Fatal(err)
	}
	src, err := png.Decode(bytes.NewReader(file))
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(src.Bounds())
	draw.Draw(img, img.Rect, src, image.Point{}, draw.Src)
	return img
}

func flatAround(img *image.RGBA, x, y int) bool {
	c := img.RGBAAt(x, y)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			p := image.Pt(x+dx, y+dy)
			if !p.In(img.Rect) || img.RGBAAt(p.X, p.Y) != c {
				return false
			}
		}
	}
	return true
}

func assertFlatRegions(t *testing.T, want, got *image.RGBA) {
	t.Helper()
	flat := 0
	for y := range want.Rect.Dy() {
		for x := range want.Rect.Dx() {
			if !flatAround(want, x, y) {
				continue
			}
			flat++
			a, b := want.RGBAAt(x, y), got.RGBAAt(x, y)
			for i, d := range []int{int(a.R) - int(b.R), int(a.G) - int(b.G), int(a.B) - int(b.B), int(a.A) - int(b.A)} {
				if d < -2 || d > 2 {
					t.Fatalf("flat pixel %d,%d channel %d: want %v got %v", x, y, i, a, b)
				}
			}
		}
	}
	if flat < want.Rect.Dx()*want.Rect.Dy()/4 {
		t.Fatalf("only %d flat pixels compared", flat)
	}
}

func TestSixelTwoColourExact(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 12, 12))
	for y := range 12 {
		for x := range 12 {
			c := color.RGBA{0, 0, 255, 255}
			if x < 3 || y >= 9 {
				c = color.RGBA{255, 0, 0, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	var s Sixel
	got := string(s.Encode(nil, lines(img), Placement{Col: 2, Row: 3, Cols: 2, Rows: 1}))
	want := "\x1b[4;3H\x1bP0;1q\"1;1;12;12#0;2;100;0;0#1;2;0;0;100#0~~~$#1???!9~-#0~~~!9w$#1???!9F\x1b\\"
	if got != want {
		t.Fatalf("\ngot  %q\nwant %q", got, want)
	}
}

func TestSixelCardRoundTrip(t *testing.T) {
	card := loadCard(t)
	var s Sixel
	out := s.Encode(nil, lines(card), Placement{Cols: 44, Rows: 8})
	d := decodeSixel(t, out)
	if d.registers > 256 || d.registers < 2 {
		t.Fatalf("registers %d", d.registers)
	}
	assertFlatRegions(t, card, d.img)
	t.Logf("card 440x168: %d bytes, %d registers", len(out), d.registers)
}

func TestSixelQuantisesPastRegisters(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 128, 96))
	for y := range 96 {
		for x := range 128 {
			c := color.RGBA{uint8(x * 2), uint8(y * 2), uint8(x + y), 255}
			if y >= 48 {
				c = color.RGBA{30, 60, 90, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	var s Sixel
	d := decodeSixel(t, s.Encode(nil, lines(img), Placement{}))
	if d.registers != 256 {
		t.Fatalf("registers %d, want 256", d.registers)
	}
	assertFlatRegions(t, img, d.img)
	worst := 0
	for y := range 48 {
		for x := range 128 {
			a, b := img.RGBAAt(x, y), d.img.RGBAAt(x, y)
			for _, dd := range []int{int(a.R) - int(b.R), int(a.G) - int(b.G), int(a.B) - int(b.B)} {
				worst = max(worst, dd, -dd)
			}
		}
	}
	t.Logf("gradient: worst channel error %d", worst)
	if worst > 16 {
		t.Fatalf("gradient worst channel error %d", worst)
	}
}

func TestSixelQuantiserCountsRepeatedRows(t *testing.T) {
	encode := func(odd uint8) string {
		img := image.NewRGBA(image.Rect(0, 0, 128, 96))
		for y := range 96 {
			for x := range 128 {
				c := color.RGBA{uint8(x * 2), uint8(y * 2), uint8(x + y), 255}
				if y >= 48 {
					c = color.RGBA{30 + odd*uint8(y%2), 60, 90, 255}
				}
				img.SetRGBA(x, y, c)
			}
		}
		var s Sixel
		return string(s.Encode(nil, lines(img), Placement{}))
	}
	if repeated, alternating := encode(0), encode(1); repeated != alternating {
		t.Fatalf("48 equal rows and 48 rows alternating between two inputs of one register encode differently: %d and %d bytes", len(repeated), len(alternating))
	}
}

func TestSixelRowRLE(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 400, 1))
	for x := range 400 {
		img.SetRGBA(x, 0, color.RGBA{240, 240, 240, 255})
	}
	img.SetRGBA(399, 0, color.RGBA{10, 10, 10, 255})
	var s Sixel
	d := decodeSixel(t, s.Encode(nil, lines(img), Placement{}))
	if d.body >= 20 {
		t.Fatalf("400 px row body is %d bytes", d.body)
	}
	img.SetRGBA(399, 0, color.RGBA{240, 240, 240, 255})
	flat := decodeSixel(t, s.Encode(nil, lines(img), Placement{}))
	if flat.body >= 20 {
		t.Fatalf("flat 400 px row body is %d bytes", flat.body)
	}
}

func TestSixelPartialBandSubImageAndTransparency(t *testing.T) {
	full := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for y := range 20 {
		for x := range 20 {
			full.SetRGBA(x, y, color.RGBA{uint8(x * 10), uint8(y * 10), 50, 255})
		}
	}
	full.SetRGBA(6, 4, color.RGBA{})
	tile := full.SubImage(image.Rect(5, 3, 15, 10)).(*image.RGBA)
	var s Sixel
	for range 2 {
		d := decodeSixel(t, s.Encode(nil, lines(tile), Placement{}))
		if d.img.Rect.Dx() != 10 || d.img.Rect.Dy() != 7 {
			t.Fatalf("decoded size %v", d.img.Rect)
		}
		for y := range 7 {
			for x := range 10 {
				want := full.RGBAAt(x+5, y+3)
				got := d.img.RGBAAt(x, y)
				if want.A == 0 {
					if got.A != 0 {
						t.Fatalf("transparent pixel painted %v", got)
					}
					continue
				}
				for _, dd := range []int{int(want.R) - int(got.R), int(want.G) - int(got.G), int(want.B) - int(got.B)} {
					if dd < -2 || dd > 2 {
						t.Fatalf("tile pixel %d,%d: want %v got %v", x, y, want, got)
					}
				}
			}
		}
	}
}

func TestSixelEmpty(t *testing.T) {
	var s Sixel
	if out := s.Encode(nil, lines(image.NewRGBA(image.Rect(0, 0, 0, 0))), Placement{}); len(out) != 0 {
		t.Fatalf("empty image encoded to %q", out)
	}
	clear := image.NewRGBA(image.Rect(0, 0, 9, 7))
	for i := range clear.Pix {
		if i%4 != 3 {
			clear.Pix[i] = uint8(i)
		}
	}
	if out := s.Encode(nil, lines(clear), Placement{}); len(out) != 0 {
		t.Fatalf("transparent image encoded to %q", out)
	}
}
