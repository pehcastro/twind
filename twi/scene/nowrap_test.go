package scene

import (
	"math"
	"slices"
	"testing"

	"github.com/twind-dev/twind/twi/style"
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
		{true, false, []string{"hello wide world", "second line here"}},
	} {
		n.NoWrap, n.Truncate = c.nowrap, c.truncate
		if got := n.Lines(text.Widths{}); !slices.Equal(got, c.want) {
			t.Errorf("nowrap %v truncate %v: %q, want %q", c.nowrap, c.truncate, got, c.want)
		}
	}
}

func TestNoWrapPaintsWhatItMeasures(t *testing.T) {
	for _, c := range []struct {
		white  style.WhiteSpace
		nowrap bool
		want   []string
	}{
		{style.WhiteSpaceNowrap, true, []string{"a b", "c"}},
		{style.WhiteSpacePre, true, []string{"  a   b  ", " c"}},
		{style.WhiteSpacePreLine, false, []string{"a b", "c"}},
		{style.WhiteSpacePreWrap, false, []string{"  a   b  ", " c"}},
		{style.WhiteSpaceNormal, false, []string{"a b", "c"}},
	} {
		n := box(0, 0, 20, 2, paint(0, 0, 0, 0))
		n.text, n.NoWrap = Sanitize("  a   b  \n c"), c.nowrap
		n.wrapping = Wrapping(&style.ComputedStyle{WhiteSpace: c.white})
		got := n.Lines(text.Widths{})
		width, height := n.text.Size(n.wrapping, math.MaxInt)
		if !slices.Equal(got, c.want) || width != text.Width(c.want[0]) || height != len(c.want) {
			t.Errorf("white-space %d: paints %q, measures %dx%d, want %q", c.white, got, width, height, c.want)
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
