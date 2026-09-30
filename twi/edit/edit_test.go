package edit

import (
	"os"
	"strconv"
	"strings"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/edit"
	"github.com/twind-dev/twind/twi/input"
)

const (
	acute = "\xcc\x81"
	zwj   = "\xe2\x80\x8d"
)

var (
	left      = input.KeyEvent{Key: input.KeyArrowLeft}
	right     = input.KeyEvent{Key: input.KeyArrowRight}
	up        = input.KeyEvent{Key: input.KeyArrowUp}
	down      = input.KeyEvent{Key: input.KeyArrowDown}
	home      = input.KeyEvent{Key: input.KeyHome}
	end       = input.KeyEvent{Key: input.KeyEnd}
	backspace = input.KeyEvent{Key: input.KeyBackspace}
	del       = input.KeyEvent{Key: input.KeyDelete}
	enter     = input.KeyEvent{Key: input.KeyEnter}
)

func with(k input.KeyEvent, m input.Modifiers) input.KeyEvent {
	k.Modifiers |= m
	return k
}

func ctrl(r rune) input.KeyEvent {
	return input.KeyEvent{Rune: r, Modifiers: input.ModCtrl}
}

func press(t *testing.T, b *Buffer, keys ...input.KeyEvent) {
	t.Helper()
	for _, k := range keys {
		if !b.Apply(k) {
			t.Fatalf("%+v not handled", k)
		}
	}
}

func typeText(t *testing.T, b *Buffer, s string) {
	t.Helper()
	for _, r := range s {
		press(t, b, input.KeyEvent{Rune: r})
	}
}

func want(t *testing.T, b *Buffer, value string, start, end int) {
	t.Helper()
	s, e := b.Selection()
	if b.Value() != value || s != start || e != end {
		t.Fatalf("got %q [%d,%d], want %q [%d,%d]", b.Value(), s, e, value, start, end)
	}
}

func wantCursor(t *testing.T, b *Buffer, row, column int) {
	t.Helper()
	if r, c := b.Cursor(); r != row || c != column {
		t.Fatalf("cursor at %d,%d, want %d,%d", r, c, row, column)
	}
}

func TestGraphemeBackspaceCombining(t *testing.T) {
	var b Buffer
	typeText(t, &b, "e"+acute)
	want(t, &b, "e"+acute, 3, 3)
	wantCursor(t, &b, 0, 1)
	press(t, &b, backspace)
	want(t, &b, "", 0, 0)
}

func TestGraphemeDeleteCombining(t *testing.T) {
	var b Buffer
	b.Insert("e" + acute + "x")
	press(t, &b, home, del)
	want(t, &b, "x", 0, 0)
}

func TestGraphemeLeftOverFamily(t *testing.T) {
	var b Buffer
	family := "\U0001F468" + zwj + "\U0001F469" + zwj + "\U0001F467"
	b.Insert("a" + family + "b")
	press(t, &b, left)
	want(t, &b, "a"+family+"b", 1+len(family), 1+len(family))
	wantCursor(t, &b, 0, 3)
	press(t, &b, left)
	want(t, &b, "a"+family+"b", 1, 1)
	press(t, &b, right)
	wantCursor(t, &b, 0, 3)
}

func TestGraphemeWideColumn(t *testing.T) {
	var b Buffer
	typeText(t, &b, "中a")
	wantCursor(t, &b, 0, 3)
}

func TestGraphemeJoinSnapsCursor(t *testing.T) {
	b := Buffer{Mode: MultiLine}
	b.Insert("a\n" + acute)
	press(t, &b, with(home, input.ModCtrl), right, right, backspace)
	want(t, &b, "a"+acute, 3, 3)
	wantCursor(t, &b, 0, 1)
}

func TestGraphemeTypingMarkStaysAfterCluster(t *testing.T) {
	var b Buffer
	b.Insert(acute)
	press(t, &b, home)
	typeText(t, &b, "e")
	want(t, &b, "e"+acute, 3, 3)
}

func TestWordMoves(t *testing.T) {
	var b Buffer
	b.Insert("foo  bar.baz")
	for _, at := range []int{9, 5, 0, 0} {
		press(t, &b, with(left, input.ModCtrl))
		want(t, &b, "foo  bar.baz", at, at)
	}
	for _, at := range []int{3, 8, 12, 12} {
		press(t, &b, with(right, input.ModAlt))
		want(t, &b, "foo  bar.baz", at, at)
	}
}

func TestWordWide(t *testing.T) {
	var b Buffer
	b.Insert("日本 語")
	press(t, &b, with(left, input.ModCtrl))
	wantCursor(t, &b, 0, 5)
}

func TestWordDelete(t *testing.T) {
	var b Buffer
	b.Insert("foo  bar")
	press(t, &b, ctrl('w'))
	want(t, &b, "foo  ", 5, 5)
	press(t, &b, with(backspace, input.ModAlt))
	want(t, &b, "", 0, 0)
	press(t, &b, ctrl('w'))
	want(t, &b, "", 0, 0)
	b.Insert("foo bar")
	press(t, &b, home, with(del, input.ModCtrl))
	want(t, &b, " bar", 0, 0)
	press(t, &b, with(del, input.ModAlt))
	want(t, &b, "", 0, 0)
}

func TestSelectShiftAndReplace(t *testing.T) {
	var b Buffer
	b.Insert("hello")
	press(t, &b, with(left, input.ModShift), with(left, input.ModShift))
	want(t, &b, "hello", 3, 5)
	typeText(t, &b, "X")
	want(t, &b, "helX", 4, 4)
	press(t, &b, with(with(left, input.ModShift), input.ModCtrl))
	want(t, &b, "helX", 0, 4)
	press(t, &b, backspace)
	want(t, &b, "", 0, 0)
}

func TestSelectCollapse(t *testing.T) {
	var b Buffer
	b.Insert("hello")
	press(t, &b, with(home, input.ModShift))
	want(t, &b, "hello", 0, 5)
	press(t, &b, right)
	want(t, &b, "hello", 5, 5)
	press(t, &b, with(left, input.ModShift), with(left, input.ModShift), left)
	want(t, &b, "hello", 3, 3)
	press(t, &b, with(end, input.ModShift), del)
	want(t, &b, "hel", 3, 3)
}

func TestSelectAll(t *testing.T) {
	b := Buffer{Mode: MultiLine}
	b.Insert("ab\ncd")
	press(t, &b, ctrl('a'))
	want(t, &b, "ab\ncd", 0, 5)
	b.Insert("z")
	want(t, &b, "z", 1, 1)
}

func TestUndoCoalescesTyping(t *testing.T) {
	var b Buffer
	typeText(t, &b, "abc")
	press(t, &b, ctrl('z'))
	want(t, &b, "", 0, 0)
	press(t, &b, ctrl('y'))
	want(t, &b, "abc", 3, 3)
	press(t, &b, backspace, backspace)
	want(t, &b, "a", 1, 1)
	press(t, &b, ctrl('z'))
	want(t, &b, "abc", 3, 3)
}

func TestUndoPasteIsOwnStep(t *testing.T) {
	var b Buffer
	typeText(t, &b, "ab")
	b.Insert(" pasted")
	typeText(t, &b, "c")
	for _, value := range []string{"ab pasted", "ab", ""} {
		press(t, &b, ctrl('z'))
		want(t, &b, value, len(value), len(value))
	}
}

func TestUndoMoveBreaksGroup(t *testing.T) {
	var b Buffer
	typeText(t, &b, "ab")
	press(t, &b, left)
	typeText(t, &b, "c")
	press(t, &b, ctrl('z'))
	want(t, &b, "ab", 1, 1)
}

func TestUndoNewEditDropsRedo(t *testing.T) {
	var b Buffer
	typeText(t, &b, "a")
	press(t, &b, ctrl('z'))
	typeText(t, &b, "b")
	press(t, &b, ctrl('y'))
	want(t, &b, "b", 1, 1)
}

func TestUndoIgnoresNoop(t *testing.T) {
	var b Buffer
	typeText(t, &b, "a")
	press(t, &b, home, backspace, ctrl('z'))
	want(t, &b, "", 0, 0)
}

func TestUndoDepthBounded(t *testing.T) {
	var b Buffer
	for range 2 * konst.UndoDepth {
		b.Insert("x ")
	}
	steps := 0
	for before := ""; before != b.Value() && steps <= 2*konst.UndoDepth; steps++ {
		before = b.Value()
		press(t, &b, ctrl('z'))
	}
	if steps-1 != konst.UndoDepth {
		t.Fatalf("%d undo steps, want %d", steps-1, konst.UndoDepth)
	}
}

func TestMultilineNewline(t *testing.T) {
	b := Buffer{Mode: MultiLine}
	typeText(t, &b, "ab")
	press(t, &b, with(enter, input.ModShift))
	typeText(t, &b, "c")
	want(t, &b, "ab\nc", 4, 4)
	wantCursor(t, &b, 1, 1)
	if b.Apply(enter) {
		t.Fatal("plain enter handled")
	}
}

func TestMultilineSingleLineRefuses(t *testing.T) {
	var b Buffer
	b.Insert("a\nb")
	want(t, &b, "ab", 2, 2)
	for _, k := range []input.KeyEvent{with(enter, input.ModShift), up, down} {
		if b.Apply(k) {
			t.Fatalf("%+v handled in single line", k)
		}
	}
}

func TestMultilinePreferredColumn(t *testing.T) {
	b := Buffer{Mode: MultiLine}
	b.Insert("abcdef\nx\nabcdef")
	press(t, &b, up)
	wantCursor(t, &b, 1, 1)
	press(t, &b, up)
	wantCursor(t, &b, 0, 6)
	press(t, &b, down, down)
	wantCursor(t, &b, 2, 6)
	press(t, &b, down)
	wantCursor(t, &b, 2, 6)
	press(t, &b, up, up, up)
	want(t, &b, "abcdef\nx\nabcdef", 0, 0)
}

func TestMultilineWideGoal(t *testing.T) {
	b := Buffer{Mode: MultiLine}
	b.Insert("中中\nabc")
	press(t, &b, up)
	wantCursor(t, &b, 0, 2)
	press(t, &b, down)
	wantCursor(t, &b, 1, 3)
}

func TestMultilineHomeEnd(t *testing.T) {
	b := Buffer{Mode: MultiLine}
	b.Insert("ab\ncd")
	press(t, &b, home)
	want(t, &b, "ab\ncd", 3, 3)
	press(t, &b, up, end)
	want(t, &b, "ab\ncd", 2, 2)
	press(t, &b, with(end, input.ModCtrl))
	want(t, &b, "ab\ncd", 5, 5)
	press(t, &b, with(with(home, input.ModCtrl), input.ModShift))
	want(t, &b, "ab\ncd", 0, 5)
}

func TestKeyReleaseIgnored(t *testing.T) {
	var b Buffer
	if b.Apply(input.KeyEvent{Rune: 'a', Release: true}) || b.Value() != "" {
		t.Fatalf("release edited %q", b.Value())
	}
}

func TestPasteHostile(t *testing.T) {
	corpus, err := os.ReadFile("../text/testdata/hostile.txt")
	if err != nil {
		t.Fatal(err)
	}
	cases := 0
	for line := range strings.Lines(string(corpus)) {
		fields := strings.Split(strings.TrimSuffix(line, "\n"), "\t")
		var pasted strings.Builder
		for _, field := range fields[1 : len(fields)-1] {
			s, err := strconv.Unquote(field)
			if err != nil {
				t.Fatalf("%s: %v", fields[0], err)
			}
			pasted.WriteString(s)
		}
		for _, mode := range []Mode{SingleLine, MultiLine} {
			b := Buffer{Mode: mode}
			b.Insert(pasted.String())
			for _, r := range b.Value() {
				if (r < ' ' && r != '\t' && r != '\n') || (r >= 0x7f && r <= 0x9f) || (r >= 0x202a && r <= 0x202e) || (r >= 0x2066 && r <= 0x2069) || (r == '\n' && mode == SingleLine) {
					t.Errorf("%s: %q keeps %U", fields[0], b.Value(), r)
				}
			}
		}
		cases++
	}
	if cases == 0 {
		t.Fatal("empty corpus")
	}
	t.Logf("%d hostile pastes", cases)
}
