package edit

import (
	"testing"

	"github.com/pehcastro/twind/twi/input"
)

func TestClickBetweenLettersThenBackspace(t *testing.T) {
	var b Buffer
	typeText(t, &b, "Peedro")
	b.Press(b.At(0, 2), Grapheme, false)
	want(t, &b, "Peedro", 2, 2)
	press(t, &b, backspace)
	want(t, &b, "Pedro", 1, 1)
}

func TestClickWideHalves(t *testing.T) {
	var b Buffer
	b.Insert("a中b")
	for column, at := range []int{0, 1, 4, 4, 5, 5} {
		if got := b.At(0, column); got != at {
			t.Errorf("column %d: offset %d, want %d", column, got, at)
		}
	}
}

func TestClickClusters(t *testing.T) {
	var b Buffer
	family := "\U0001F468" + zwj + "\U0001F469" + zwj + "\U0001F467"
	b.Insert("e" + acute + family + "x")
	for column, at := range []int{0, 3, 3 + len(family), 3 + len(family), 4 + len(family)} {
		if got := b.At(0, column); got != at {
			t.Errorf("column %d: offset %d, want %d", column, got, at)
		}
	}
}

func TestClickOutsideLines(t *testing.T) {
	b := Buffer{Mode: MultiLine}
	b.Insert("ab\ncd\nef")
	for _, c := range []struct{ row, column, at int }{
		{0, 9, 2},
		{1, 0, 3},
		{1, 1, 4},
		{1, 7, 5},
		{2, -3, 6},
		{5, 1, 7},
		{-1, 1, 1},
	} {
		if got := b.At(c.row, c.column); got != c.at {
			t.Errorf("row %d column %d: offset %d, want %d", c.row, c.column, got, c.at)
		}
	}
}

func TestClickScrolledField(t *testing.T) {
	var b Buffer
	b.Insert("0123456789abcdefghij")
	scroll, x := 12, 3
	b.Press(b.At(0, scroll+x), Grapheme, false)
	press(t, &b, del)
	want(t, &b, "0123456789abcdeghij", 15, 15)
}

func TestClickDouble(t *testing.T) {
	b := Buffer{Mode: MultiLine}
	b.Insert("hello big world\nnext")
	b.Press(7, Word, false)
	want(t, &b, "hello big world\nnext", 6, 9)
	b.Press(5, Word, false)
	want(t, &b, "hello big world\nnext", 5, 6)
	b.Press(15, Word, false)
	want(t, &b, "hello big world\nnext", 10, 15)
	b.Press(20, Word, false)
	want(t, &b, "hello big world\nnext", 16, 20)
}

func TestClickDoubleEmptyLine(t *testing.T) {
	b := Buffer{Mode: MultiLine}
	b.Insert("a\n\nb")
	b.Press(2, Word, false)
	want(t, &b, "a\n\nb", 2, 2)
}

func TestClickTriple(t *testing.T) {
	b := Buffer{Mode: MultiLine}
	b.Insert("one two\nthree four\nfive")
	b.Press(11, Line, false)
	want(t, &b, "one two\nthree four\nfive", 8, 18)
}

func TestClickShiftExtends(t *testing.T) {
	var b Buffer
	b.Insert("hello world")
	b.Press(2, Grapheme, false)
	b.Press(8, Grapheme, true)
	want(t, &b, "hello world", 2, 8)
	b.Press(0, Grapheme, true)
	want(t, &b, "hello world", 0, 2)
	press(t, &b, backspace)
	want(t, &b, "llo world", 0, 0)
}

func TestClickShiftThenDrag(t *testing.T) {
	var b Buffer
	b.Insert("hello world")
	b.Press(4, Grapheme, false)
	b.Press(8, Grapheme, true)
	b.Drag(1)
	want(t, &b, "hello world", 1, 4)
	press(t, &b, with(right, input.ModShift))
	want(t, &b, "hello world", 2, 4)
}

func TestDragGraphemes(t *testing.T) {
	var b Buffer
	b.Insert("hello world")
	b.Press(3, Grapheme, false)
	b.Drag(8)
	want(t, &b, "hello world", 3, 8)
	b.Drag(1)
	want(t, &b, "hello world", 1, 3)
	press(t, &b, with(left, input.ModShift))
	want(t, &b, "hello world", 0, 3)
}

func TestDragWords(t *testing.T) {
	var b Buffer
	b.Insert("one two three four")
	b.Press(5, Word, false)
	b.Drag(15)
	want(t, &b, "one two three four", 4, 18)
	b.Drag(1)
	want(t, &b, "one two three four", 0, 7)
	press(t, &b, with(right, input.ModShift))
	want(t, &b, "one two three four", 1, 7)
}

func TestDragLines(t *testing.T) {
	b := Buffer{Mode: MultiLine}
	b.Insert("ab\ncd\nef")
	b.Press(4, Line, false)
	b.Drag(7)
	want(t, &b, "ab\ncd\nef", 3, 8)
	b.Drag(0)
	want(t, &b, "ab\ncd\nef", 0, 5)
}

func TestClickStartsUndoStep(t *testing.T) {
	var b Buffer
	typeText(t, &b, "abc")
	b.Press(1, Grapheme, false)
	typeText(t, &b, "x")
	press(t, &b, ctrl('z'))
	want(t, &b, "abc", 1, 1)
}
