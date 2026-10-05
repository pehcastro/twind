package paint

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pehcastro/twind/internal/buffer"
	konst "github.com/pehcastro/twind/internal/konst/paint"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/icon"
)

func TestCoverageStandInsForLackedGlyphs(t *testing.T) {
	text := "☾ ⑂ ok 中 ∞ ✓"
	lacking := func(lacked string) func(string) bool {
		return func(cluster string) bool { return !strings.Contains(lacked, cluster) }
	}
	for _, c := range []struct {
		name   string
		covers func(string) bool
		want   string
	}{
		{"no coverage info", nil, "☾ ⑂ ok 中 ∞ ✓ "},
		{"covers everything", lacking(""), "☾ ⑂ ok 中 ∞ ✓ "},
		{"lacks the moon, an icon, a wide glyph and one with no stand-in", lacking("☾⑂中∞"), "● Y ok      ✓ "},
		{"lacks the moon's stand-in too", lacking("☾●"), "  ⑂ ok 中 ∞ ✓ "},
	} {
		p := Painter{Profile: color.TrueColor, Covers: c.covers}
		buf := buffer.New(14, 1)
		root := page(14, 1, text)
		p.Paint(buf, &root, Composited)
		if got := strings.Join(rows(buf), ""); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
		for x, cell := range buf.Row(0) {
			if cell.Width == buffer.Wide && c.covers != nil && !c.covers(cell.Grapheme) {
				t.Errorf("%s: cell %d keeps the lacked wide glyph %q", c.name, x, cell.Grapheme)
			}
		}
	}
}

func TestCoverageEveryIconGlyphHasAStandIn(t *testing.T) {
	const text = "−¤∕«»·∞"
	for n := icon.Name(1); n <= icon.Count; n++ {
		r := n.Glyph()
		if r < utf8.RuneSelf || strings.ContainsRune(konst.ConsoleGlyphs+text, r) {
			continue
		}
		if !strings.ContainsRune(konst.ConsoleMissing, r) {
			t.Errorf("icon %s draws %q, which has no console stand-in", n, r)
		}
	}
}
