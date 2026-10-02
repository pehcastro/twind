package edit

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/edit"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/text"
)

type Mode uint8

const (
	SingleLine Mode = iota
	MultiLine
)

type step uint8

const (
	stepOther step = iota
	stepTyping
	stepErasing
	stepVertical
)

type state struct {
	value          string
	cursor, anchor int
}

type Buffer struct {
	Mode   Mode
	Widths text.Widths
	Now    func() time.Time
	state
	undo, redo []state
	last       step
	edited     time.Time
	goal       int
	unit       Unit
	grabbed    [2]int
}

func (b *Buffer) Value() string { return b.value }

func (b *Buffer) Selection() (start, end int) {
	return min(b.cursor, b.anchor), max(b.cursor, b.anchor)
}

func (b *Buffer) Cursor() (row, column int) {
	line := b.lineStart(b.cursor)
	return strings.Count(b.value[:line], "\n"), b.Widths.Width(b.value[line:b.cursor])
}

func (b *Buffer) Insert(s string) { b.insert(s, stepOther) }

func (b *Buffer) Apply(k input.KeyEvent) bool {
	if k.Release {
		return false
	}
	shift := k.Modifiers&input.ModShift != 0
	word := k.Modifiers&(input.ModCtrl|input.ModAlt) != 0
	ctrl := k.Key == input.KeyRune && k.Modifiers == input.ModCtrl
	redo := k.Key == input.KeyRune && k.Modifiers == input.ModCtrl|input.ModShift && unicode.ToLower(k.Rune) == 'z'
	multi := b.Mode == MultiLine
	start, end := b.Selection()
	switch {
	case k.Key == input.KeyRune && k.Modifiers&^input.ModShift == 0:
		b.insert(string(k.Rune), stepTyping)
	case ctrl && k.Rune == 'a':
		b.anchor, b.cursor, b.last = 0, len(b.value), stepOther
	case ctrl && k.Rune == 'w':
		b.erase(b.wordLeft(), b.cursor, stepOther)
	case ctrl && k.Rune == 'z':
		b.restore(&b.undo, &b.redo)
	case ctrl && k.Rune == 'y', redo:
		b.restore(&b.redo, &b.undo)
	case k.Key == input.KeyEnter && shift && multi:
		b.insert("\n", stepOther)
	case k.Key == input.KeyBackspace && word:
		b.erase(b.wordLeft(), b.cursor, stepOther)
	case k.Key == input.KeyBackspace:
		b.erase(b.prev(b.cursor), b.cursor, stepErasing)
	case k.Key == input.KeyDelete && word:
		b.erase(b.cursor, b.wordRight(), stepOther)
	case k.Key == input.KeyDelete:
		b.erase(b.cursor, b.next(b.cursor), stepErasing)
	case k.Key == input.KeyArrowLeft && word:
		b.move(b.wordLeft(), shift)
	case k.Key == input.KeyArrowLeft && start != end && !shift:
		b.move(start, shift)
	case k.Key == input.KeyArrowLeft:
		b.move(b.prev(b.cursor), shift)
	case k.Key == input.KeyArrowRight && word:
		b.move(b.wordRight(), shift)
	case k.Key == input.KeyArrowRight && start != end && !shift:
		b.move(end, shift)
	case k.Key == input.KeyArrowRight:
		b.move(b.next(b.cursor), shift)
	case k.Key == input.KeyHome && word:
		b.move(0, shift)
	case k.Key == input.KeyHome:
		b.move(b.lineStart(b.cursor), shift)
	case k.Key == input.KeyEnd && word:
		b.move(len(b.value), shift)
	case k.Key == input.KeyEnd:
		b.move(b.lineEnd(b.cursor), shift)
	case (k.Key == input.KeyArrowUp || k.Key == input.KeyArrowDown) && multi:
		b.move(b.vertical(k.Key), shift)
		b.last = stepVertical
	default:
		return false
	}
	return true
}

func (b *Buffer) insert(s string, kind step) {
	s = text.Sanitize(s, text.RemoveBidi)
	if b.Mode == SingleLine {
		s = strings.ReplaceAll(s, "\n", "")
	}
	start, end := b.Selection()
	if start != end {
		b.last = stepOther
	}
	b.replace(start, end, s, kind)
}

func (b *Buffer) erase(from, to int, kind step) {
	if start, end := b.Selection(); start != end {
		from, to, kind = start, end, stepOther
	}
	b.replace(from, to, "", kind)
}

func (b *Buffer) replace(start, end int, s string, kind step) {
	if start == end && s == "" {
		return
	}
	if b.Now != nil {
		now := b.Now()
		if now.Sub(b.edited) >= konst.UndoPause {
			b.last = stepOther
		}
		b.edited = now
	}
	if kind == stepOther || kind != b.last {
		b.undo = append(b.undo, b.state)
		if len(b.undo) > konst.UndoDepth {
			b.undo = b.undo[1:]
		}
	}
	b.redo = nil
	b.value = b.value[:start] + s + b.value[end:]
	b.cursor = start + len(s)
	if b.cursor > 0 {
		b.cursor = b.next(b.cursor - 1)
	}
	b.anchor = b.cursor
	b.last = kind
}

func (b *Buffer) restore(from, to *[]state) {
	if len(*from) == 0 {
		return
	}
	*to = append(*to, b.state)
	b.state = (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	b.last = stepOther
}

func (b *Buffer) move(to int, shift bool) {
	b.cursor = to
	if !shift {
		b.anchor = to
	}
	b.last = stepOther
}

func (b *Buffer) lineStart(i int) int {
	return strings.LastIndexByte(b.value[:i], '\n') + 1
}

func (b *Buffer) lineEnd(i int) int {
	if n := strings.IndexByte(b.value[i:], '\n'); n >= 0 {
		return i + n
	}
	return len(b.value)
}

func (b *Buffer) next(i int) int {
	at := b.lineStart(i)
	for g := range text.Graphemes(b.value[at:]) {
		if at += len(g); at > i {
			return at
		}
	}
	return len(b.value)
}

func (b *Buffer) prev(i int) int {
	at := b.lineStart(i)
	if at == i {
		return max(i-1, 0)
	}
	for g := range text.Graphemes(b.value[at:]) {
		if at+len(g) >= i {
			return at
		}
		at += len(g)
	}
	return at
}

func isWord(g string) bool {
	r, _ := utf8.DecodeRuneInString(g)
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func (b *Buffer) wordLeft() int {
	left, at := 0, 0
	for w := range text.Words(b.value) {
		if at >= b.cursor {
			break
		}
		if isWord(w) {
			left = at
		}
		at += len(w)
	}
	return b.prev(b.next(left))
}

func (b *Buffer) wordRight() int {
	at := 0
	for w := range text.Words(b.value) {
		if at += len(w); at > b.cursor && isWord(w) {
			return b.next(at - 1)
		}
	}
	return len(b.value)
}

func (b *Buffer) vertical(k input.Key) int {
	if b.last != stepVertical {
		_, b.goal = b.Cursor()
	}
	line, lineEnd := b.lineStart(b.cursor), b.lineEnd(b.cursor)
	switch {
	case k == input.KeyArrowUp && line == 0:
		return 0
	case k == input.KeyArrowUp:
		line = b.lineStart(line - 1)
	case lineEnd == len(b.value):
		return lineEnd
	default:
		line = lineEnd + 1
	}
	width := 0
	for g := range text.Graphemes(b.value[line:b.lineEnd(line)]) {
		if width += b.Widths.Width(g); width > b.goal {
			break
		}
		line += len(g)
	}
	return line
}
