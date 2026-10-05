package scene

import (
	"slices"
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/text"
)

func TestWordBreakReachesLines(t *testing.T) {
	const hash = "commit 0123456789abcdef0123456789abcdef01234567"
	for _, c := range []struct {
		name     string
		raw      string
		word     style.WordBreak
		overflow style.OverflowWrap
		want     []string
	}{
		{"normal hash", hash, style.WordBreakNormal, style.OverflowWrapNormal, []string{"commit", "0123456789ab", "cdef01234567", "89abcdef0123", "4567"}},
		{"break-all hash", hash, style.WordBreakAll, style.OverflowWrapNormal, []string{"commit 01234", "56789abcdef0", "123456789abc", "def01234567"}},
		{"anywhere hash", hash, style.WordBreakNormal, style.OverflowWrapAnywhere, []string{"commit", "0123456789ab", "cdef01234567", "89abcdef0123", "4567"}},
		{"normal cjk", "中文排版 需要断行。日本語", style.WordBreakNormal, style.OverflowWrapNormal, []string{"中文排版 需", "要断行。日本", "語"}},
		{"keep-all cjk", "中文排版 需要断行。日本語", style.WordBreakKeepAll, style.OverflowWrapNormal, []string{"中文排版", "需要断行。", "日本語"}},
	} {
		s := style.ComputedStyle{Opacity: 1, WordBreak: c.word, OverflowWrap: c.overflow}
		n := New(&layout.Box{ContentBox: layout.Rect{W: 12, H: 5}}, s, Sanitize(c.raw))
		got := n.Lines(text.Widths{})
		t.Logf("%s, 12 cells:\n|%s|", c.name, strings.Join(got, "|\n|"))
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
		w, h := n.text.Size(Wrapping(&s), 12)
		if h != len(c.want) || w > 12 {
			t.Errorf("%s: measured %dx%d, want at most 12 wide and %d lines", c.name, w, h, len(c.want))
		}
	}
}

func TestWordBreakSharedTextKeepsEachMode(t *testing.T) {
	shared := Sanitize("commit 0123456789abcdef0123456789abcdef01234567")
	box := &layout.Box{ContentBox: layout.Rect{W: 12, H: 5}}
	normal := New(box, style.ComputedStyle{Opacity: 1}, shared)
	all := New(box, style.ComputedStyle{Opacity: 1, WordBreak: style.WordBreakAll}, shared)
	for range 2 {
		if got := normal.Lines(text.Widths{})[0]; got != "commit" {
			t.Errorf("normal after break-all on the same text: first line %q, want commit", got)
		}
		if got := all.Lines(text.Widths{})[0]; got != "commit 01234" {
			t.Errorf("break-all after normal on the same text: first line %q, want commit 01234", got)
		}
	}
	if _, lines := shared.Size(Wrapping(&style.ComputedStyle{OverflowWrap: style.OverflowWrapAnywhere}), 1); lines != 46 {
		t.Errorf("anywhere at 1 cell: %d lines, want 46", lines)
	}
	if got := shared.MinContent(Wrapping(&style.ComputedStyle{WordBreak: style.WordBreakAll})); got != 1 {
		t.Errorf("break-all min-content %d, want 1", got)
	}
}

func TestWordBreakUnknownValuePanics(t *testing.T) {
	for _, s := range []style.ComputedStyle{{WordBreak: style.WordBreakKeepAll + 1}, {OverflowWrap: style.OverflowWrapAnywhere + 1}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("word-break %d overflow-wrap %d mapped without a panic", s.WordBreak, s.OverflowWrap)
				}
			}()
			Wrapping(&s)
		}()
	}
}
