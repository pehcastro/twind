package scene

import (
	"slices"
	"testing"

	"github.com/twind-dev/twind/twi/text"
)

func TestNoWrapClipsInsteadOfWrapping(t *testing.T) {
	n := box(0, 0, 10, 1, paint(0, 0, 0, 0))
	n.text = Sanitize("hello wide world\nsecond line here")
	for _, c := range []struct {
		nowrap, truncate bool
		want             []string
	}{
		{false, false, []string{"hello wide", "world", "second", "line here"}},
		{true, false, []string{"hello wide world", "second line here"}},
		{true, true, []string{"hello wid…", "second li…"}},
	} {
		n.NoWrap, n.Truncate = c.nowrap, c.truncate
		if got := n.Lines(text.Widths{}); !slices.Equal(got, c.want) {
			t.Errorf("nowrap %v truncate %v: %q, want %q", c.nowrap, c.truncate, got, c.want)
		}
	}
}

func TestNoWrapWidthsReachTheWrapCache(t *testing.T) {
	n := box(0, 0, 3, 1, paint(0, 0, 0, 0))
	n.text = Sanitize("🇧🇷🇧🇷🇧🇷")
	for _, c := range []struct {
		widths text.Widths
		want   []string
	}{
		{text.Widths{}, []string{"🇧🇷", "🇧🇷", "🇧🇷"}},
		{text.Widths{text.Flag: 1}, []string{"🇧🇷🇧🇷🇧🇷"}},
		{text.Widths{}, []string{"🇧🇷", "🇧🇷", "🇧🇷"}},
	} {
		if got := n.Lines(c.widths); !slices.Equal(got, c.want) {
			t.Errorf("widths %v: %q, want %q", c.widths, got, c.want)
		}
	}
	n.Content.W, n.NoWrap, n.Truncate = 2, true, true
	if got, want := n.Lines(text.Widths{text.Flag: 1}), []string{"🇧🇷…"}; !slices.Equal(got, want) {
		t.Errorf("truncated under flag 1: %q, want %q", got, want)
	}
}
