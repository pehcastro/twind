package icon_test

import (
	"bytes"
	"cmp"
	"image"
	"image/png"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/pehcastro/twind/twi/icon"
	"github.com/pehcastro/twind/twi/text"
)

func reference(t *testing.T, size int) *image.Gray {
	t.Helper()
	raw, err := os.ReadFile("testdata/lucide-" + strconv.Itoa(size) + ".png")
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return img.(*image.Gray)
}

func compare(t *testing.T, size int, dst image.Rectangle, at image.Point) {
	ref := reference(t, size)
	const cols, maxError, meanError, meanOverAll = 16, 80, 9.0, 2.5
	var s icon.Stroker
	worst, total := 0, 0
	for n := icon.Name(1); n <= icon.Count; n++ {
		got := image.NewAlpha(dst)
		s.Draw(got, n)
		sum, i := 0, int(n)-1
		for y := range size {
			for x := range size {
				want := int(ref.GrayAt(i%cols*size+x, i/cols*size+y).Y)
				have := int(got.AlphaAt(dst.Min.X+at.X+x, dst.Min.Y+at.Y+y).A)
				d := max(want-have, have-want)
				sum += d
				if d > maxError {
					t.Errorf("%s at %dpx: pixel %d,%d is %d, lucide through resvg is %d", n, size, x, y, have, want)
				}
				worst = max(worst, d)
			}
		}
		if mean := float64(sum) / float64(size*size); mean > meanError {
			t.Errorf("%s at %dpx: mean error %.2f over %.2f", n, size, mean, meanError)
		}
		total += sum
	}
	mean := float64(total) / float64(size*size*int(icon.Count))
	if mean > meanOverAll {
		t.Errorf("%dpx: mean error %.3f over every icon, want at most %.1f", size, mean, meanOverAll)
	}
	t.Logf("%dpx in %v: worst pixel %d/255, mean %.3f/255 over %d icons", size, dst, worst, mean, icon.Count)
}

func TestDrawOneCellAt8x16(t *testing.T) { compare(t, 16, image.Rect(0, 0, 16, 16), image.Point{}) }

func TestDrawOneCellAt10x20(t *testing.T) { compare(t, 20, image.Rect(0, 0, 20, 20), image.Point{}) }

func TestDrawCentresInAnOffsetWideBox(t *testing.T) {
	compare(t, 20, image.Rect(7, 3, 7+26, 3+20), image.Point{X: 3})
}

func TestDrawEmptyBox(t *testing.T) {
	var s icon.Stroker
	s.Draw(image.NewAlpha(image.Rect(4, 4, 4, 9)), icon.Search)
}

func TestDrawAllocatesNothingOnceWarm(t *testing.T) {
	var s icon.Stroker
	dst := image.NewAlpha(image.Rect(0, 0, 20, 20))
	s.Draw(dst, icon.LoaderCircle)
	if n := testing.AllocsPerRun(100, func() { s.Draw(dst, icon.CircleAlert) }); n != 0 {
		t.Fatalf("%v allocations per draw", n)
	}
}

func BenchmarkDrawEveryIcon(b *testing.B) {
	for _, size := range []int{16, 20, 40} {
		b.Run(image.Pt(size, size).String(), func(b *testing.B) {
			var s icon.Stroker
			dst := image.NewAlpha(image.Rect(0, 0, size, size))
			b.ReportAllocs()
			for b.Loop() {
				for n := icon.Name(1); n <= icon.Count; n++ {
					s.Draw(dst, n)
				}
			}
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(icon.Count), "ns/icon")
		})
	}
}

const emoji = "00A9 00AE 203C 2049 2122 2139 2194..2199 21A9..21AA 231A..231B 2328 23CF 23E9..23F3 23F8..23FA 24C2 25AA..25AB 25B6 25C0 25FB..25FE " +
	"2600..2604 260E 2611 2614..2615 2618 261D 2620 2622..2623 2626 262A 262E..262F 2638..263A 2640 2642 2648..2653 265F..2660 2663 2665..2666 " +
	"2668 267B 267E..267F 2692..2697 2699 269B..269C 26A0..26A1 26A7 26AA..26AB 26B0..26B1 26BD..26BE 26C4..26C5 26C8 26CE..26CF 26D1 26D3..26D4 " +
	"26E9..26EA 26F0..26F5 26F7..26FA 26FD 2702 2705 2708..270D 270F 2712 2714 2716 271D 2721 2728 2733..2734 2744 2747 274C 274E 2753..2755 2757 " +
	"2763..2764 2795..2797 27A1 27B0 27BF 2934..2935 2B05..2B07 2B1B..2B1C 2B50 2B55 3030 303D 3297 3299"

func TestFallbackIsOneCell(t *testing.T) {
	for n := icon.Name(1); n <= icon.Count; n++ {
		g := n.Glyph()
		if w := text.Width(string(g)); w != 1 || !unicode.IsGraphic(g) || unicode.IsSpace(g) {
			t.Errorf("%s: fallback %q is %d cells wide or not a plain graphic", n, g, w)
		}
		for field := range strings.FieldsSeq(emoji) {
			lo, hi, _ := strings.Cut(field, "..")
			from, _ := strconv.ParseUint(lo, 16, 32)
			to, _ := strconv.ParseUint(cmp.Or(hi, lo), 16, 32)
			if uint64(g) >= from && uint64(g) <= to {
				t.Errorf("%s: fallback %q has the Emoji property, and Windows Terminal draws it as a colour emoji", n, g)
			}
		}
	}
}

func TestNamesResolve(t *testing.T) {
	kebab := regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	seen := map[string]bool{}
	for n := icon.Name(1); n <= icon.Count; n++ {
		name := n.String()
		if !kebab.MatchString(name) || seen[name] {
			t.Errorf("%d: name %q is not a unique lucide file name", n, name)
		}
		seen[name] = true
	}
	if icon.Count < 60 {
		t.Errorf("%d icons, want at least 60", icon.Count)
	}
	defer func() {
		if recover() == nil {
			t.Error("the zero name did not panic")
		}
	}()
	_ = icon.Name(0).String()
}
