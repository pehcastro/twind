package text

import (
	"slices"
	"strings"
	"testing"
)

func BenchmarkWrap(b *testing.B) {
	paragraph := strings.Repeat("A well-known pangram: the quick brown fox jumps over the lazy dog, see https://example.com/fox 中文排版。 ", 20)
	b.ReportAllocs()
	for b.Loop() {
		Wrap(paragraph, 60)
	}
}

func BenchmarkMinContent(b *testing.B) {
	paragraph := strings.Repeat("A well-known pangram: the quick brown fox jumps over the lazy dog, see https://example.com/fox 中文排版。 ", 20)
	b.ReportAllocs()
	for b.Loop() {
		MinContent(paragraph)
	}
}

func TestWrap(t *testing.T) {
	cases := []struct {
		s     string
		width int
		want  []string
	}{
		{"Terminal DOM rocks", 8, []string{"Terminal", "DOM", "rocks"}},
		{"a b c d", 3, []string{"a b", "c d"}},
		{"abcdefghij", 4, []string{"abcd", "efgh", "ij"}},
		{"中文中文中", 5, []string{"中文", "中文", "中"}},
		{"中文", 1, []string{"中", "文"}},
		{"one\ntwo three", 5, []string{"one", "two", "three"}},
		{"e\U00000301e\U00000301e\U00000301", 2, []string{"e\U00000301e\U00000301", "e\U00000301"}},
		{"", 4, []string{""}},
	}
	for _, c := range cases {
		if got := Wrap(c.s, c.width); !slices.Equal(got, c.want) {
			t.Errorf("Wrap(%+q, %d) = %+q, want %+q", c.s, c.width, got, c.want)
		}
	}
}

func TestBreak(t *testing.T) {
	cases := []struct {
		s     string
		width int
		want  []string
	}{
		{"a package-level drive", 10, []string{"a package-", "level", "drive"}},
		{"hello 世界你好", 8, []string{"hello 世", "界你好"}},
		{"中文中文。", 8, []string{"中文中", "文。"}},
		{"see (a) now", 5, []string{"see", "(a)", "now"}},
		{"go (   b", 4, []string{"go", "( b"}},
		{"wait wait !", 9, []string{"wait", "wait !"}},
		{"wait !", 4, []string{"wait", "!"}},
		{"wait !", 5, []string{"wait", "!"}},
		{"https://example.com/docs/wrapping", 12, []string{"https://", "example.com/", "docs/", "wrapping"}},
		{"https://averyveryverylonghost.io/x", 10, []string{"https://", "averyveryv", "erylonghos", "t.io/x"}},
		{"use -verbose or --quiet", 5, []string{"use", "-verb", "ose", "or --", "quiet"}},
		{"x -5 3.14 1,000 $5 5%", 5, []string{"x -5", "3.14", "1,000", "$5 5%"}},
		{"ab-cd", 3, []string{"ab-", "cd"}},
		{"a" + brazil + family + "b", 2, []string{"a", brazil, family, "b"}},
		{"  lead  and   gaps  ", 20, []string{"lead and gaps"}},
		{"ab\U0000200Bcd", 3, []string{"ab\U0000200B", "cd"}},
		{"no\U000000A0break here", 8, []string{"no\U000000A0break", "here"}},
		{"ab-cd", 0, []string{"a", "b", "-", "c", "d"}},
	}
	for _, c := range cases {
		if got := Wrap(c.s, c.width); !slices.Equal(got, c.want) {
			t.Errorf("Wrap(%+q, %d) = %+q, want %+q", c.s, c.width, got, c.want)
		}
	}
}

func TestMinContent(t *testing.T) {
	cases := []struct {
		s    string
		want int
	}{
		{"Keyboard shortcuts", 9},
		{"Profile", 7},
		{"中文", 2},
		{"a" + brazil + family + "b", 2},
		{"e\U00000301e\U00000301e\U00000301", 3},
		{"a package-level drive", 8},
		{"no\U000000A0break here", 8},
		{"wait !", 6},
		{"  lead  and   gaps  ", 4},
		{"one\ntwo three", 5},
		{"https://example.com/docs/wrapping", 12},
		{"", 0},
	}
	for _, c := range cases {
		if got := MinContent(c.s); got != c.want {
			t.Errorf("MinContent(%+q) = %d, want %d", c.s, got, c.want)
		}
	}
	if got := (Widths{Flag: 1}).MinContent(brazil + " x"); got != 1 {
		t.Errorf("MinContent with a flag override = %d, want 1", got)
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		s     string
		width int
		want  string
	}{
		{"中文中文", 5, "中文…"},
		{"中文中文", 4, "中…"},
		{"中文中文", 8, "中文中文"},
		{"A very long project name", 23, "A very long project na…"},
		{"fits", 4, "fits"},
		{"e\U00000301\U00000301xyz", 2, "e\U00000301\U00000301…"},
		{"abc", 0, ""},
	}
	for _, c := range cases {
		got := Truncate(c.s, c.width)
		if got != c.want || Width(got) > c.width {
			t.Errorf("Truncate(%+q, %d) = %+q width %d, want %+q", c.s, c.width, got, Width(got), c.want)
		}
	}
}
