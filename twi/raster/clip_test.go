package raster

import (
	"bytes"
	"image"
	"math"
	"testing"
)

func TestRoundClip(t *testing.T) {
	track := radius(20, 10, 160, 24, 1<<20)
	full := radius(0, 0, 200, 50, 0)
	parent := render(200, 50, Op{Kind: Fill, Box: track, Color: black})
	filled := render(200, 50, Op{Kind: Clip, Box: track}, Op{Kind: Fill, Box: full, Color: black}, Op{Kind: Pop})
	if !bytes.Equal(parent.Pix, filled.Pix) {
		t.Errorf("a child filling a rounded clip differs from the parent's own rounded fill")
	}

	bar := render(200, 50, Op{Kind: Clip, Box: track}, Op{Kind: Fill, Box: radius(20, 10, 96, 24, 0), Color: black}, Op{Kind: Pop})
	for y := range 50 {
		for x := range 200 {
			want := at(parent, x, y)
			if x >= 116 {
				want = white
			}
			if got := at(bar, x, y); got != want {
				t.Fatalf("unrounded indicator at (%d,%d): %v, want %v", x, y, got, want)
			}
		}
	}

	nested := render(200, 50, Op{Kind: Opacity, Opacity: 0.5}, Op{Kind: Clip, Box: track}, Op{Kind: Fill, Box: full, Color: black}, Op{Kind: Pop}, Op{Kind: Pop})
	faded := render(200, 50, Op{Kind: Opacity, Opacity: 0.5}, Op{Kind: Fill, Box: track, Color: black}, Op{Kind: Pop})
	if !bytes.Equal(nested.Pix, faded.Pix) {
		t.Errorf("a rounded clip inside an opacity group differs from the faded rounded fill")
	}
	inside := render(200, 50, Op{Kind: Clip, Box: track}, Op{Kind: Opacity, Opacity: 0.5}, Op{Kind: Fill, Box: full, Color: black}, Op{Kind: Pop}, Op{Kind: Pop})
	for y := range 50 {
		for x := range 200 {
			if got, want := at(inside, x, y), at(faded, x, y); !near(got, want, 1) {
				t.Fatalf("an opacity group inside a rounded clip at (%d,%d): %v, the faded rounded fill %v", x, y, got, want)
			}
		}
	}
}

func ring(img *image.RGBA, box image.Rectangle) []float64 {
	var out []float64
	x0, y0, x1, y1 := box.Min.X, box.Min.Y, box.Max.X-1, box.Max.Y-1
	for x := x0; x < x1; x++ {
		out = append(out, darkness(img, x, y0))
	}
	for y := y0; y < y1; y++ {
		out = append(out, darkness(img, x1, y))
	}
	for x := x1; x > x0; x-- {
		out = append(out, darkness(img, x, y1))
	}
	for y := y1; y > y0; y-- {
		out = append(out, darkness(img, x0, y))
	}
	return out
}

func runs(ink []float64) []float64 {
	start := 0
	for start < len(ink) && ink[start] > 0 {
		start++
	}
	var sums []float64
	on := false
	for i := range ink {
		v := ink[(start+i)%len(ink)]
		switch {
		case v > 0 && on:
			sums[len(sums)-1] += v
		case v > 0:
			sums, on = append(sums, v), true
		default:
			on = false
		}
	}
	return sums
}

func total(img *image.RGBA) float64 {
	sum := 0.0
	for y := range img.Rect.Dy() {
		for x := range img.Rect.Dx() {
			sum += darkness(img, x, y)
		}
	}
	return sum
}

func TestDashed(t *testing.T) {
	for _, c := range []struct {
		dash     Dash
		fraction float64
		run      float64
	}{{Dashed, 3.0 / 5, 3}, {Dotted, 0.5, 1}} {
		square := image.Rect(10, 10, 111, 51)
		img := render(130, 70, Op{Kind: Border, Box: radius(10, 10, 101, 41, 0), Width: 1, Color: black, Dash: c.dash})
		sums := runs(ring(img, square))
		if len(sums) < 2 {
			t.Errorf("dash %d on a square ring: %d ink runs, want a pattern", c.dash, len(sums))
		}
		for i, s := range sums {
			if math.Abs(s-sums[0]) > 0.15 || math.Abs(s-c.run) > 0.5 {
				t.Errorf("dash %d run %d of %d holds %.2f px of ink, run 0 holds %.2f, want about %v each, seam included", c.dash, i, len(sums), s, sums[0], c.run)
			}
		}

		box := radius(20.5, 20.25, 120, 60, 8)
		solid := render(160, 100, Op{Kind: Border, Box: box, Width: 1, Color: black})
		dashed := render(160, 100, Op{Kind: Border, Box: box, Width: 1, Color: black, Dash: c.dash})
		for y := range 100 {
			for x := range 160 {
				if darkness(dashed, x, y) > darkness(solid, x, y)+1.0/255 {
					t.Fatalf("dash %d inks (%d,%d) beyond the solid ring", c.dash, x, y)
				}
			}
		}
		if f := total(dashed) / total(solid); math.Abs(f-c.fraction) > 0.04 {
			t.Errorf("dash %d on a rounded ring inks %.3f of the solid ring, want %.3f", c.dash, f, c.fraction)
		}
		corner := 0.0
		for y := 20; y < 28; y++ {
			for x := 20; x < 28; x++ {
				corner += darkness(dashed, x, y)
			}
		}
		if corner == 0 {
			t.Errorf("dash %d leaves the top left arc bare", c.dash)
		}

		line := render(130, 10, Op{Kind: Fill, Box: radius(10, 4, 101, 1, 0), Color: black, Dash: c.dash})
		if darkness(line, 10, 4) < 0.99 || darkness(line, 110, 4) < 0.99 {
			t.Errorf("dash %d side line ends %.2f and %.2f, want a dash at each end", c.dash, darkness(line, 10, 4), darkness(line, 110, 4))
		}
		if f := total(line) / 101; math.Abs(f-c.fraction) > 0.06 {
			t.Errorf("dash %d side line inks %.3f of its length, want %.3f", c.dash, f, c.fraction)
		}
	}
}
