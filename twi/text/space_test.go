package text

import (
	"math"
	"slices"
	"testing"
)

func TestSpace(t *testing.T) {
	pre := Wrapping{Space: SpacePreserve}
	cases := []struct {
		b     Wrapping
		s     string
		width int
		want  []string
		min   int
	}{
		{pre, "    x", math.MaxInt, []string{"    x"}, 1},
		{pre, "a ", math.MaxInt, []string{"a "}, 1},
		{pre, "a   b", math.MaxInt, []string{"a   b"}, 1},
		{pre, "   ", math.MaxInt, []string{"   "}, 0},
		{pre, "  if x {\n    return\n  }", math.MaxInt, []string{"  if x {", "    return", "  }"}, 6},
		{pre, "foo   bar", 4, []string{"foo ", "bar"}, 3},
		{pre, "  ab cd", 5, []string{"  ab ", "cd"}, 2},
		{pre, "    x", 2, []string{"  ", "x"}, 1},
		{pre, "go  (a)", 3, []string{"go ", "(a)"}, 3},
		{pre, "ab  cdefg", 3, []string{"ab ", "cde", "fg"}, 5},
		{Wrapping{}, "a   b", math.MaxInt, []string{"a b"}, 1},
		{Wrapping{}, "  a  b  ", math.MaxInt, []string{"a b"}, 1},
		{Wrapping{}, "a ", math.MaxInt, []string{"a"}, 1},
	}
	for _, c := range cases {
		if got := c.b.Wrap(c.s, c.width); !slices.Equal(got, c.want) {
			t.Errorf("space %d Wrap(%+q, %d) = %+q, want %+q", c.b.Space, c.s, c.width, got, c.want)
		}
		if got := c.b.MinContent(c.s); got != c.min {
			t.Errorf("space %d MinContent(%+q) = %d, want %d", c.b.Space, c.s, got, c.min)
		}
	}
}

func TestSpaceInARow(t *testing.T) {
	pre := Wrapping{Space: SpacePreserve}
	if lines := pre.Wrap("    x", math.MaxInt); len(lines) != 1 || Width(lines[0]) != 5 {
		t.Errorf("pre \"    x\" = %+q, want one line of width 5", lines)
	}
	a, b := pre.Wrap("a ", math.MaxInt), pre.Wrap("b", math.MaxInt)
	if row := a[0] + b[0]; row != "a b" || Width(a[0])+Width(b[0]) != 3 {
		t.Errorf("pre nodes \"a \" and \"b\" in a row draw %q, want \"a b\" over 3 cells", row)
	}
}

func TestSpaceUnknownMode(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("an unknown white-space mode did not panic")
		}
	}()
	Wrapping{Space: SpacePreserve + 1}.Wrap("a b", 1)
}
