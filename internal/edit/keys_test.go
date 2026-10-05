package edit

import (
	"testing"
	"time"

	"github.com/pehcastro/twind/twi/input"
)

func multi(s string) *Buffer {
	b := &Buffer{Mode: MultiLine}
	b.Insert(s)
	return b
}

func TestKeyNewlineShiftEnterAndCtrlJ(t *testing.T) {
	b := &Buffer{Mode: MultiLine}
	typeText(t, b, "a")
	press(t, b, with(enter, input.ModShift))
	typeText(t, b, "b")
	press(t, b, ctrl('j'))
	typeText(t, b, "c")
	want(t, b, "a\nb\nc", 5, 5)
	if b.Apply(enter, time.Time{}) {
		t.Fatal("plain enter broke the line")
	}
	var single Buffer
	if single.Apply(ctrl('j'), time.Time{}) || single.Value() != "" {
		t.Fatalf("a one-line field took ctrl+j: %q", single.Value())
	}
}

func TestKeyEmacsMoves(t *testing.T) {
	b := multi("ab\ncd")
	for _, step := range []struct {
		key input.KeyEvent
		at  int
	}{
		{ctrl('a'), 3}, {ctrl('e'), 5}, {ctrl('b'), 4}, {ctrl('f'), 5},
		{ctrl('p'), 2}, {ctrl('b'), 1}, {ctrl('n'), 4}, {ctrl('a'), 3}, {ctrl('b'), 2}, {ctrl('f'), 3},
	} {
		press(t, b, step.key)
		want(t, b, "ab\ncd", step.at, step.at)
	}
}

func TestKeySelectAllIsCtrlG(t *testing.T) {
	b := multi("ab\ncd")
	press(t, b, ctrl('g'))
	want(t, b, "ab\ncd", 0, 5)
	press(t, b, ctrl('a'))
	want(t, b, "ab\ncd", 3, 3)
}

func TestKeyArrowDropsSelectionAndMovesFromCaret(t *testing.T) {
	var b Buffer
	b.Insert("hello")
	press(t, &b, with(left, input.ModShift), with(left, input.ModShift), left)
	want(t, &b, "hello", 2, 2)
	press(t, &b, with(right, input.ModShift), with(right, input.ModShift), right)
	want(t, &b, "hello", 5, 5)
}

func TestKeyKillToLineEdges(t *testing.T) {
	b := multi("ab\ncd")
	press(t, b, with(home, input.ModCtrl), right, right, right, right, ctrl('u'))
	want(t, b, "ab\nd", 3, 3)
	press(t, b, ctrl('u'))
	want(t, b, "abd", 2, 2)
	press(t, b, ctrl('u'))
	want(t, b, "d", 0, 0)
	b = multi("ab\ncd")
	press(t, b, with(home, input.ModCtrl), right, ctrl('k'))
	want(t, b, "a\ncd", 1, 1)
	press(t, b, ctrl('k'))
	want(t, b, "acd", 1, 1)
}

func TestKeyDeleteWordStaysInLine(t *testing.T) {
	b := multi("foo\nbar")
	press(t, b, ctrl('a'), ctrl('w'))
	want(t, b, "foobar", 3, 3)
	b = multi("foo\nbar")
	press(t, b, with(home, input.ModCtrl), ctrl('e'), with(del, input.ModCtrl))
	want(t, b, "foobar", 3, 3)
	b = multi("go foo-bar.baz")
	press(t, b, ctrl('w'))
	want(t, b, "go ", 3, 3)
	press(t, b, with(backspace, input.ModCtrl))
	want(t, b, "", 0, 0)
}

func TestKeyWordMovesByWhitespace(t *testing.T) {
	b := multi("a foo-bar\nbaz")
	for _, at := range []int{10, 2, 0, 0} {
		press(t, b, with(left, input.ModCtrl))
		want(t, b, "a foo-bar\nbaz", at, at)
	}
	for _, at := range []int{1, 9, 13, 13} {
		press(t, b, with(right, input.ModCtrl))
		want(t, b, "a foo-bar\nbaz", at, at)
	}
}

func TestKeyTranspose(t *testing.T) {
	var b Buffer
	b.Insert("abc")
	press(t, &b, ctrl('t'))
	want(t, &b, "acb", 3, 3)
	press(t, &b, home, ctrl('t'))
	want(t, &b, "acb", 0, 0)
	press(t, &b, right, ctrl('t'))
	want(t, &b, "cab", 2, 2)
	b = Buffer{}
	b.Insert("e" + acute + "b")
	press(t, &b, ctrl('t'))
	want(t, &b, "be"+acute, 4, 4)
	m := multi("x\nab")
	press(t, m, ctrl('a'), ctrl('t'))
	want(t, m, "x\nab", 2, 2)
	press(t, m, up, ctrl('t'))
	want(t, m, "x\nab", 0, 0)
}

func TestKeyCtrlHAndCtrlD(t *testing.T) {
	var b Buffer
	b.Insert("abc")
	press(t, &b, ctrl('h'), home, ctrl('d'))
	want(t, &b, "b", 0, 0)
}

func TestKeySetIsOneStep(t *testing.T) {
	var b Buffer
	typeText(t, &b, "abc")
	b.Set("xy")
	want(t, &b, "xy", 2, 2)
	press(t, &b, ctrl('z'))
	want(t, &b, "abc", 3, 3)
}

func TestKeyWrapRows(t *testing.T) {
	for _, c := range []struct {
		value string
		wrap  int
		rows  []Row
	}{
		{"hello world", 6, []Row{{0, 6}, {6, 11}}},
		{"hello world!", 6, []Row{{0, 6}, {6, 12}, {12, 12}}},
		{"中中中", 4, []Row{{0, 6}, {6, 9}}},
		{"abcdefgh", 5, []Row{{0, 5}, {5, 8}}},
		{"ab cd efghij", 8, []Row{{0, 6}, {6, 12}}},
		{"ab\n\ncd", 0, []Row{{0, 2}, {3, 3}, {4, 6}}},
		{"hi\nhello world", 6, []Row{{0, 2}, {3, 9}, {9, 14}}},
	} {
		b := Buffer{Mode: MultiLine, Wrap: c.wrap}
		b.Insert(c.value)
		got := b.Rows()
		if len(got) != len(c.rows) {
			t.Errorf("%q at %d: rows %v, want %v", c.value, c.wrap, got, c.rows)
			continue
		}
		for i := range got {
			if got[i] != c.rows[i] {
				t.Errorf("%q at %d: rows %v, want %v", c.value, c.wrap, got, c.rows)
				break
			}
		}
	}
}

func TestKeyWrappedCaretAndClicks(t *testing.T) {
	b := Buffer{Mode: MultiLine, Wrap: 6}
	b.Insert("hello world")
	wantCursor(t, &b, 1, 5)
	b.Press(6, Grapheme, false)
	wantCursor(t, &b, 1, 0)
	for _, c := range []struct{ row, column, at int }{{0, 9, 6}, {1, 2, 8}, {0, 2, 2}, {4, 0, 6}} {
		if got := b.At(c.row, c.column); got != c.at {
			t.Errorf("row %d column %d: offset %d, want %d", c.row, c.column, got, c.at)
		}
	}
}

func TestKeyWrappedUpDown(t *testing.T) {
	b := Buffer{Mode: MultiLine, Wrap: 8}
	b.Insert("ab cd efghij")
	wantCursor(t, &b, 1, 6)
	press(t, &b, up)
	wantCursor(t, &b, 0, 5)
	press(t, &b, up)
	wantCursor(t, &b, 0, 5)
	press(t, &b, down)
	wantCursor(t, &b, 1, 6)
	press(t, &b, down)
	wantCursor(t, &b, 1, 6)
	press(t, &b, home, down)
	wantCursor(t, &b, 1, 0)
}

func TestHistoryWalksBackAndRestoresTheDraft(t *testing.T) {
	var b Buffer
	b.Remember("first")
	b.Remember("second")
	typeText(t, &b, "draft")
	press(t, &b, up)
	want(t, &b, "second", 6, 6)
	press(t, &b, up)
	want(t, &b, "first", 5, 5)
	if b.Apply(up, time.Time{}) || b.Value() != "first" {
		t.Fatalf("up past the oldest holds %q", b.Value())
	}
	press(t, &b, down)
	want(t, &b, "second", 6, 6)
	press(t, &b, down)
	want(t, &b, "draft", 5, 5)
	if b.Apply(down, time.Time{}) || b.Value() != "draft" {
		t.Fatalf("down past the draft holds %q", b.Value())
	}
}

func TestHistoryEditRestartsTheWalk(t *testing.T) {
	var b Buffer
	b.Remember("a")
	b.Remember("b")
	press(t, &b, up)
	typeText(t, &b, "x")
	press(t, &b, up)
	want(t, &b, "b", 1, 1)
	press(t, &b, down)
	want(t, &b, "bx", 2, 2)
}

func TestHistoryOnlyOnOneLogicalLine(t *testing.T) {
	b := multi("a\nb")
	b.Remember("old")
	press(t, b, up)
	want(t, b, "a\nb", 1, 1)
	w := Buffer{Mode: MultiLine, Wrap: 4}
	w.Remember("old")
	w.Insert("abcdefgh")
	press(t, &w, up)
	want(t, &w, "old", 3, 3)
}

func TestHistoryNotOnShiftOrCtrlP(t *testing.T) {
	var b Buffer
	b.Remember("old")
	b.Insert("ab")
	for _, k := range []input.KeyEvent{with(up, input.ModShift), ctrl('p'), ctrl('n')} {
		if b.Apply(k, time.Time{}) || b.Value() != "ab" {
			t.Fatalf("%+v walked history: %q", k, b.Value())
		}
	}
}

func TestHistoryNoneLeavesTheArrows(t *testing.T) {
	var b Buffer
	b.Insert("ab")
	if b.Apply(up, time.Time{}) || b.Apply(down, time.Time{}) {
		t.Fatal("a one-line field with no history took up or down")
	}
}
