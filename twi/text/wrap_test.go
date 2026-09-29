package text

import (
	"slices"
	"testing"
)

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
