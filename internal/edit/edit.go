package edit

import (
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	konst "github.com/pehcastro/twind/internal/konst/edit"
	ukonst "github.com/pehcastro/twind/internal/konst/ui"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/text"
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

type Row struct{ Start, End int }

const (
	ChipHead = "[Text "
	ChipTail = " characters]"
)

type chip struct{ token, text string }

type entry struct {
	text  string
	chips []chip
}

type layout struct {
	value  string
	wrap   int
	widths text.Widths
	rows   []Row
}

type Buffer struct {
	Mode   Mode
	Wrap   int
	Widths text.Widths
	state
	undo, redo []state
	last       step
	now        time.Time
	edited     time.Time
	goal       int
	unit       Unit
	grabbed    [2]int
	history    []entry
	walk       int
	draft      entry
	chips      []chip
	laid       layout
}

func (b *Buffer) Value() string { return b.value }

func (b *Buffer) Selection() (start, end int) {
	return min(b.cursor, b.anchor), max(b.cursor, b.anchor)
}

func (b *Buffer) Cursor() (row, column int) {
	rows := b.Rows()
	for row+1 < len(rows) && rows[row+1].Start <= b.cursor {
		row++
	}
	return row, b.Widths.Width(b.value[rows[row].Start:b.cursor])
}

func (b *Buffer) Insert(s string) { b.insert(s, stepOther) }

func (b *Buffer) Set(s string) { b.replace(0, len(b.value), s, stepOther) }

func (b *Buffer) Paste(s string) {
	s = strings.NewReplacer("\r\n", "\n", "\r", "\n", "\t", strings.Repeat(" ", ukonst.PasteTabSpaces)).Replace(s)
	if runes := utf8.RuneCountInString(s); runes >= ukonst.PasteChipRunes {
		c := chip{ChipHead + strconv.Itoa(runes) + ChipTail, s}
		b.chips = append(b.chips, c)
		s = c.token
	}
	b.Insert(s)
}

func (b *Buffer) Expand(s string) string {
	for _, c := range b.chips {
		s = strings.Replace(s, c.token, c.text, 1)
	}
	return s
}

func (b *Buffer) Remember(s string) {
	var kept []chip
	for _, c := range b.chips {
		if strings.Contains(s, c.token) {
			kept = append(kept, c)
		}
	}
	b.history, b.chips = append(b.history, entry{s, kept}), nil
	b.walk = len(b.history)
}

func (b *Buffer) Apply(k input.KeyEvent, now time.Time) bool {
	if k.Release {
		return false
	}
	b.now = now
	shift := k.Modifiers&input.ModShift != 0
	word := k.Modifiers&(input.ModCtrl|input.ModAlt) != 0
	var ctrl rune
	if k.Key == input.KeyRune && k.Modifiers == input.ModCtrl {
		ctrl = k.Rune
	}
	redo := k.Key == input.KeyRune && k.Modifiers == input.ModCtrl|input.ModShift && unicode.ToLower(k.Rune) == 'z'
	multi := b.Mode == MultiLine
	var by int
	switch {
	case k.Key == input.KeyArrowUp, ctrl == 'p':
		by = -1
	case k.Key == input.KeyArrowDown, ctrl == 'n':
		by = 1
	}
	line, end := b.lineStart(b.cursor), b.lineEnd(b.cursor)
	floor, ceil := line, end
	if b.cursor == line {
		floor = max(line-1, 0)
	}
	if b.cursor == end {
		ceil = min(end+1, len(b.value))
	}
	switch {
	case k.Key == input.KeyRune && k.Modifiers&^input.ModShift == 0:
		b.insert(string(k.Rune), stepTyping)
	case ctrl == 'g':
		b.anchor, b.cursor, b.last = 0, len(b.value), stepOther
	case ctrl == 'k':
		b.erase(b.cursor, ceil, stepOther)
	case ctrl == 'u':
		b.erase(floor, b.cursor, stepOther)
	case ctrl == 't':
		b.transpose(line, end)
	case ctrl == 'w', k.Key == input.KeyBackspace && word:
		b.erase(b.wordLeft(floor), b.cursor, stepOther)
	case ctrl == 'z':
		b.restore(&b.undo, &b.redo)
	case ctrl == 'y', redo:
		b.restore(&b.redo, &b.undo)
	case multi && (ctrl == 'j' || k.Key == input.KeyEnter && shift):
		b.insert("\n", stepOther)
	case k.Key == input.KeyBackspace, ctrl == 'h':
		b.erase(b.prev(b.cursor), b.cursor, stepErasing)
	case k.Key == input.KeyDelete && word:
		b.erase(b.cursor, b.wordRight(ceil), stepOther)
	case k.Key == input.KeyDelete, ctrl == 'd':
		b.erase(b.cursor, b.next(b.cursor), stepErasing)
	case k.Key == input.KeyArrowLeft && word:
		b.move(b.wordLeft(0), shift)
	case k.Key == input.KeyArrowLeft, ctrl == 'b':
		b.move(b.prev(b.cursor), shift)
	case k.Key == input.KeyArrowRight && word:
		b.move(b.wordRight(len(b.value)), shift)
	case k.Key == input.KeyArrowRight, ctrl == 'f':
		b.move(b.next(b.cursor), shift)
	case k.Key == input.KeyHome && word:
		b.move(0, shift)
	case k.Key == input.KeyHome, ctrl == 'a':
		b.move(line, shift)
	case k.Key == input.KeyEnd && word:
		b.move(len(b.value), shift)
	case k.Key == input.KeyEnd, ctrl == 'e':
		b.move(end, shift)
	case by != 0 && k.Modifiers == 0 && b.recall(by):
	case by != 0 && multi:
		b.move(b.vertical(by), shift)
		b.last = stepVertical
	default:
		return false
	}
	return true
}

func (b *Buffer) insert(s string, kind step) {
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
	s = text.Sanitize(s, text.RemoveBidi)
	if b.Mode == SingleLine {
		s = strings.ReplaceAll(s, "\n", "")
	}
	if start == end && s == "" {
		return
	}
	if b.now.Sub(b.edited) >= konst.UndoPause {
		b.last = stepOther
	}
	b.edited = b.now
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
	b.walk, b.draft = len(b.history), entry{}
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

func (b *Buffer) recall(by int) bool {
	to := b.walk + by
	if strings.Contains(b.value, "\n") || to < 0 || to > len(b.history) {
		return false
	}
	if b.walk == len(b.history) {
		b.draft = entry{b.value, b.chips}
	}
	draft, recalled := b.draft, b.draft
	if to < len(b.history) {
		recalled = b.history[to]
	}
	b.Set(recalled.text)
	b.walk, b.draft, b.chips = to, draft, slices.Clone(recalled.chips)
	return true
}

func (b *Buffer) move(to int, shift bool) {
	b.cursor = to
	if !shift {
		b.anchor = to
	}
	b.last = stepOther
}

func (b *Buffer) transpose(line, end int) {
	if b.cursor == line || b.next(line) >= end {
		return
	}
	mid := b.cursor
	if mid == end {
		mid = b.prev(mid)
	}
	lo, hi := b.prev(mid), b.next(mid)
	b.replace(lo, hi, b.value[mid:hi]+b.value[lo:mid], stepOther)
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

func space(g string) bool {
	r, _ := utf8.DecodeRuneInString(g)
	return unicode.IsSpace(r)
}

func (b *Buffer) wordLeft(floor int) int {
	at := b.cursor
	for at > floor && space(b.value[b.prev(at):at]) {
		at = b.prev(at)
	}
	for at > floor && !space(b.value[b.prev(at):at]) {
		at = b.prev(at)
	}
	return at
}

func (b *Buffer) wordRight(ceil int) int {
	at := b.cursor
	for at < ceil && space(b.value[at:b.next(at)]) {
		at = b.next(at)
	}
	for at < ceil && !space(b.value[at:b.next(at)]) {
		at = b.next(at)
	}
	return at
}

func (b *Buffer) Rows() []Row {
	if b.laid.rows != nil && b.laid.value == b.value && b.laid.wrap == b.Wrap && b.laid.widths == b.Widths {
		return b.laid.rows
	}
	rows := b.laid.rows[:0]
	for at := 0; ; at++ {
		end := b.lineEnd(at)
		rows = b.wrap(rows, at, end)
		if end == len(b.value) {
			b.laid = layout{b.value, b.Wrap, b.Widths, rows}
			return rows
		}
		at = end
	}
}

func (b *Buffer) wrap(rows []Row, at, end int) []Row {
	if b.Wrap <= 0 {
		return append(rows, Row{at, end})
	}
	start, from, row, word := at, at, 0, 0
	for g := range text.Graphemes(b.value[at:end]) {
		w := b.Widths.Width(g)
		at += len(g)
		if space(g) {
			if row+word+w > b.Wrap && from > start {
				rows, start, row = append(rows, Row{start, from}), from, 0
			}
			row, word, from = row+word+w, 0, at
			continue
		}
		if word += w; word+w > b.Wrap {
			if from > start {
				rows, start, row = append(rows, Row{start, from}), from, 0
			}
			row, word, from = row+word, 0, at
		}
	}
	if row+word >= b.Wrap {
		rows, start = append(rows, Row{start, from}), from
	}
	return append(rows, Row{start, end})
}

func (b *Buffer) vertical(by int) int {
	row, column := b.Cursor()
	if b.last != stepVertical {
		b.goal = column
	}
	rows := b.Rows()
	row = min(max(row+by, 0), len(rows)-1)
	r := rows[row]
	soft := row+1 < len(rows) && rows[row+1].Start == r.End
	at, width := r.Start, 0
	for g := range text.Graphemes(b.value[r.Start:r.End]) {
		if width += b.Widths.Width(g); width > b.goal || soft && at+len(g) == r.End {
			break
		}
		at += len(g)
	}
	return at
}
