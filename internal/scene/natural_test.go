package scene

import (
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi/text"
)

var naturalSamples = []string{
	"Getting started", "  leading and trailing  ", "two  spaces   inside", "tab\there", "a\nb\n\nc", "trailing\n",
	"中文排版 需要断行。日本語", "שלום עולם and ltr", "مرحبا بالعالم", "emoji 👩‍👩‍👧 family 🇧🇷 flag", "hyphen-ated - words -x",
	"commit 0123456789abcdef0123456789abcdef01234567", "é combining", "zero" + string(rune(0x200b)) + "width", " ", "x",
	"func main() {\n\tfmt.Println(\"hi\")\n}", "a/b/c.d?e=f&g=h", "(paren) [bracket] {brace}", "100% 3.14 1,000",
}

func naturalWrappings() []text.Wrapping {
	var out []text.Wrapping
	for _, word := range []text.WordBreak{text.WordBreakNormal, text.WordBreakAll, text.WordBreakKeepAll} {
		for _, over := range []text.OverflowWrap{text.OverflowWrapNormal, text.OverflowWrapBreakWord, text.OverflowWrapAnywhere} {
			for _, space := range []text.Space{text.SpaceCollapse, text.SpacePreserve} {
				for _, dir := range []text.Direction{text.DirLTR, text.DirRTL, text.DirAuto} {
					out = append(out, text.Wrapping{Word: word, Overflow: over, Space: space, Dir: dir})
				}
			}
		}
	}
	return out
}

func TestWideEnoughWrapKeepsUnboundedLines(t *testing.T) {
	pieces := []string{"a", "bc", "  ", " ", "中", "-", "\t", "\n", "👩‍👩‍👧", "ש", "word", "x-y", "。"}
	rng := rand.New(rand.NewPCG(1, 2))
	samples := slices.Clone(naturalSamples)
	for range 400 {
		var b strings.Builder
		for range 1 + rng.IntN(12) {
			b.WriteString(pieces[rng.IntN(len(pieces))])
		}
		samples = append(samples, b.String())
	}
	for _, b := range naturalWrappings() {
		for _, raw := range samples {
			clean := text.Sanitize(raw, text.RemoveBidi)
			unbounded := b.Wrap(clean, math.MaxInt)
			natural := 0
			for _, line := range unbounded {
				natural = max(natural, b.Widths.Width(line))
			}
			for _, width := range []int{natural, natural + 1, natural + 7} {
				if got := b.Wrap(clean, width); !slices.Equal(got, unbounded) {
					t.Fatalf("%+v %q at %d: %q, unbounded %q", b, raw, width, got, unbounded)
				}
			}
		}
	}
}

func TestCachedLinesMatchWrap(t *testing.T) {
	ascii := " !\"#$%&'()*+,-./0123456789:;<=>?@AZaz[\\]^_`{|}~"
	rng := rand.New(rand.NewPCG(3, 4))
	samples := slices.Clone(naturalSamples)
	for range 300 {
		var b strings.Builder
		for range 1 + rng.IntN(20) {
			b.WriteByte(ascii[rng.IntN(len(ascii))])
		}
		samples = append(samples, b.String())
	}
	for _, b := range naturalWrappings() {
		for _, raw := range samples {
			txt := Sanitize(raw)
			for _, width := range []int{math.MaxInt, 3, 40, 1, 3, 0, 12, math.MaxInt, 7} {
				want := b.Wrap(txt.clean, width)
				if got := txt.wrap(b, width); !slices.Equal(got, want) {
					t.Fatalf("%+v %q at %d: %q, want %q", b, raw, width, got, want)
				}
				w, h := txt.Size(b, width)
				ww := 0
				for _, line := range want {
					ww = max(ww, b.Widths.Width(line))
				}
				if w != ww || h != len(want) {
					t.Fatalf("%+v %q at %d: size %dx%d, want %dx%d", b, raw, width, w, h, ww, len(want))
				}
			}
		}
	}
}

func TestSanitizeMatchesText(t *testing.T) {
	var all strings.Builder
	for c := range 256 {
		all.WriteByte(byte(c))
	}
	inputs := append(slices.Clone(naturalSamples), all.String(), "\x1b[31mred\x1b[0m", "bell\a", "del\x7f", string(rune(0x202e))+"evil", "\xc2\x9b31m", "plain ascii ~!@#$%^&*()_+")
	for c := 0x20; c < 0x7f; c++ {
		inputs = append(inputs, string(rune(c)))
	}
	for _, raw := range inputs {
		if got, want := Sanitize(raw).clean, text.Sanitize(raw, text.RemoveBidi); got != want {
			t.Errorf("%q: %q, want %q", raw, got, want)
		}
	}
}
