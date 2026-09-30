package text

import (
	"fmt"
	"math/bits"
	"strings"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/text"
)

type lineClass byte

const (
	alphabetic lineClass = iota
	numeric
	ideographic
	nonStarter
	space
	breakAfter
	hyphen
	breakBefore
	breakBoth
	glue
	wordJoiner
	zeroWidthSpace
	openPunct
	closePunct
	closeParen
	exclamation
	infix
	symbol
	quotation
	prefix
	postfix
	inseparable
)

func lineBreaks(before, after lineClass, spaced, leadingHyphen bool) bool {
	switch {
	case after == zeroWidthSpace:
		return false
	case before == zeroWidthSpace:
		return true
	case after == wordJoiner, before == openPunct,
		after == closePunct, after == closeParen, after == exclamation, after == infix, after == symbol,
		(before == closePunct || before == closeParen) && after == nonStarter,
		before == breakBoth && after == breakBoth:
		return false
	case spaced:
		return true
	case after == glue:
		return before == breakAfter || before == hyphen
	case before == wordJoiner, before == glue, before == quotation, after == quotation, before == breakBefore,
		after == breakAfter, after == hyphen, after == nonStarter, after == inseparable,
		leadingHyphen && after == alphabetic:
		return false
	}
	switch before {
	case alphabetic, numeric:
		return after != alphabetic && after != numeric && after != openPunct && after != prefix && after != postfix
	case closeParen:
		return after != alphabetic && after != numeric && after != prefix && after != postfix
	case closePunct:
		return after != prefix && after != postfix
	case infix:
		return after != alphabetic && after != numeric
	case symbol, hyphen:
		return after != numeric
	case prefix:
		return after != alphabetic && after != numeric && after != ideographic && after != openPunct
	case postfix:
		return after != alphabetic && after != numeric && after != openPunct
	case ideographic:
		return after != postfix
	}
	return true
}

type wrapper struct {
	w          Widths
	s          string
	width      int
	lines      []string
	start, end int
	copied     bool
	line       []byte
	used       int
	keep       bool
}

type WordBreak uint8

const (
	WordBreakNormal WordBreak = iota
	WordBreakAll
	WordBreakKeepAll
)

type OverflowWrap uint8

const (
	OverflowWrapNormal OverflowWrap = iota
	OverflowWrapBreakWord
	OverflowWrapAnywhere
)

type Space uint8

const (
	SpaceCollapse Space = iota
	SpacePreserve
)

type Wrapping struct {
	Widths   Widths
	Word     WordBreak
	Overflow OverflowWrap
	Space    Space
}

func Wrap(s string, width int) []string {
	return Wrapping{}.Wrap(s, width)
}

func (w Widths) Wrap(s string, width int) []string {
	return Wrapping{Widths: w}.Wrap(s, width)
}

func (b Wrapping) Wrap(s string, width int) []string {
	out := wrapper{w: b.Widths, s: s, width: width, keep: b.Space == SpacePreserve, lines: make([]string, 0, len(s)/max(width, 1)+1)}
	from := 0
	for paragraph := range strings.SplitSeq(s, "\n") {
		b.segments(s, from, from+len(paragraph), out.place)
		out.flush()
		from += len(paragraph) + 1
	}
	return out.lines
}

type segment struct {
	start, end, width int
	spaced, squeeze   bool
}

type coalescer struct {
	emit       func(segment) int
	room, need int
	held       segment
	holding    bool
}

func (c *coalescer) spare(seg *segment) int {
	if seg.squeeze || seg.start != c.held.end+b2i(seg.spaced) {
		return konst.NoRoom
	}
	return c.room - c.need - b2i(seg.spaced)
}

func (c *coalescer) push(seg segment) {
	if seg.width > c.spare(&seg) {
		c.flush()
		c.room = c.emit(seg)
		c.held, c.need = segment{end: seg.end}, 0
		return
	}
	if c.holding {
		c.held.end, c.held.width = seg.end, c.held.width+b2i(seg.spaced)+seg.width
	} else {
		c.held, c.holding = seg, true
	}
	c.need += b2i(seg.spaced) + seg.width
}

func (c *coalescer) flush() {
	if c.holding {
		c.emit(c.held)
		c.holding = false
	}
}

func (w Widths) MinContent(s string) int {
	return Wrapping{Widths: w}.MinContent(s)
}

func (b Wrapping) MinContent(s string) int {
	widest := 0
	emit := func(seg segment) int {
		widest = max(widest, seg.width)
		return konst.NoRoom
	}
	if b.Overflow == OverflowWrapAnywhere {
		emit = func(seg segment) int {
			for pos := seg.start; pos < seg.end; {
				n, cluster := b.Widths.next(s[pos:seg.end])
				widest = max(widest, cluster)
				pos += n
			}
			return konst.NoRoom
		}
	}
	from := 0
	for paragraph := range strings.SplitSeq(s, "\n") {
		b.segments(s, from, from+len(paragraph), emit)
		from += len(paragraph) + 1
	}
	return widest
}

func (b Wrapping) segments(s string, from, to int, emit func(segment) int) {
	if b.Word > WordBreakKeepAll || b.Overflow > OverflowWrapAnywhere || b.Space > SpacePreserve {
		panic(fmt.Sprintf("text: unknown word-break %d, overflow-wrap %d or white-space %d", b.Word, b.Overflow, b.Space))
	}
	keep := b.Space == SpacePreserve
	s = s[:to]
	out := coalescer{emit: emit, room: konst.NoRoom}
	var before lineClass
	seg, end, kept := segment{start: from}, from, 0
	var spaced, leadingHyphen bool
	for pos := from; pos < len(s); {
		n, width, after := 1, 1, printable(s, pos)
		if after == konst.Unprintable {
			n, width, after = b.Widths.cluster(s[pos:])
		}
		if n == 1 && s[pos] == ' ' {
			spaced = true
			pos++
			if keep {
				end, kept = pos, kept+1
			}
			continue
		}
		if b.Word == WordBreakAll && after <= numeric {
			after = ideographic
		}
		keptWhole := b.Word == WordBreakKeepAll && !spaced && before <= ideographic && after <= ideographic
		if end == from || !keptWhole && lineBreaks(before, after, spaced, leadingHyphen) {
			seg.end = end
			out.push(seg)
			seg = segment{start: pos, spaced: spaced && !keep}
			if e, w, class := b.leap(s, pos, out.spare(&seg)); e > 0 {
				seg.width, pos, end, before, leadingHyphen, spaced = w, e, e, class, false, false
				continue
			}
		} else if spaced {
			seg.width += max(kept, 1)
			seg.squeeze = seg.squeeze || !keep && pos-end > 1
		}
		seg.width += width
		seg.squeeze = seg.squeeze || n > 1 && (s[pos] == ' ' || s[pos+n-1] == ' ')
		leadingHyphen = after == hyphen && (spaced || end == from)
		pos += n
		before, spaced, kept = after, false, 0
		for before <= numeric && pos < len(s) {
			run, spacedWord := letters(s, pos)
			pos += run
			seg.width += run
			if run > 0 {
				before = lineClass(printableLines[s[pos-1]])
			}
			if spacedWord {
				seg.end = pos
				if keep {
					seg.end++
				}
				out.push(seg)
				pos++
				seg = segment{start: pos, spaced: !keep}
				if e, w, class := b.leap(s, pos, out.spare(&seg)); e > 0 {
					seg.width, pos, before = w, e, class
					break
				}
				continue
			}
			if run == konst.ByteLanes {
				continue
			}
			class := printable(s, pos)
			if class > numeric {
				break
			}
			before = class
			pos++
			seg.width++
		}
		for before == ideographic && b.Word != WordBreakKeepAll && pos < len(s) && s[pos] >= utf8.RuneSelf {
			r, size := utf8.DecodeRuneInString(s[pos:])
			head := record(r)
			if breakClass(head[0]) != other || lineClass(head[2]) != ideographic || attaches(s[pos+size:]) {
				break
			}
			seg.end = pos
			out.push(seg)
			seg = segment{start: pos, width: int(head[1] & konst.WidthMask)}
			pos += size
		}
		end = pos
	}
	seg.end = end
	out.push(seg)
	out.flush()
}

func (b Wrapping) leap(s string, from, spare int) (end, width int, before lineClass) {
	if spare < konst.LeapMin {
		return 0, 0, 0
	}
	last, spacedNow := lineClass(konst.Unprintable), false
	run, runWidth := -1, 0
	for pos, w := from, 0; ; {
		more := pos < len(s) && w <= spare+1
		if more && pos+konst.ByteLanes <= len(s) && s[pos] < utf8.RuneSelf {
			word := lanes(s, pos)
			low := word &^ konst.HighBits
			space := equal(word, ' ')
			visible := ^word & (low + lane(utf8.RuneSelf-' ')) &^ (low + lane(utf8.RuneSelf-'~'-1)) &^ (space & equal(lanes(s, pos-1), ' '))
			if visible&konst.HighBits == konst.HighBits {
				if run < 0 {
					run, runWidth = pos, w
				}
				pos, w = pos+konst.ByteLanes, w+konst.ByteLanes
				continue
			}
		}
		if run >= 0 {
			if e, cells, class := landing(s, run, pos, spare-runWidth); e > 0 {
				end, width, before = e, runWidth+cells, class
			}
			spacedNow = s[pos-1] == ' '
			last, run = lineClass(printableLines[s[pos-1-b2i(spacedNow)]]), -1
		}
		if !more {
			return end, width, before
		}
		ascii := s[pos] < utf8.RuneSelf
		class, size, cells := lineClass(printableLines[s[pos]]), 1, 1
		if !ascii {
			r, n := utf8.DecodeRuneInString(s[pos:])
			head := record(r)
			if breakClass(head[0]) != other {
				break
			}
			class, size, cells = lineClass(head[2]), n, int(head[1]&konst.WidthMask)
		}
		if class == konst.Unprintable || class == space && spacedNow {
			break
		}
		switch {
		case spacedNow && ascii && class <= numeric && last != openPunct && w-1 <= spare:
			end, width, before = pos-1, w-1, last
		case !spacedNow && last == ideographic && class == ideographic && b.Word != WordBreakKeepAll && w <= spare:
			end, width, before = pos, w, ideographic
		}
		spacedNow = class == space
		if !spacedNow {
			last = class
		}
		pos, w = pos+size, w+cells
	}
	return end, width, before
}

func landing(s string, from, to, spare int) (end, width int, before lineClass) {
	for q := min(to-1, from+spare+1); q >= from+2; q-- {
		if s[q-1] == ' ' && lineClass(printableLines[s[q]]) <= numeric && lineClass(printableLines[s[q-2]]) != openPunct {
			return q - 1, q - 1 - from, lineClass(printableLines[s[q-2]])
		}
	}
	return 0, 0, 0
}

func letters(s string, pos int) (run int, spacedWord bool) {
	if pos+konst.ByteLanes >= len(s) {
		return 0, false
	}
	w, next := lanes(s, pos), lanes(s, pos+1)
	plain := alnum(w) &^ (w | next) & konst.HighBits
	run = bits.TrailingZeros64(^plain&konst.HighBits) / konst.ByteLanes
	return run, plain&(equal(w, ' ')<<konst.ByteLanes)>>(run*konst.ByteLanes+konst.ByteLanes)&utf8.RuneSelf != 0
}

func alnum(w uint64) uint64 {
	low := w &^ konst.HighBits
	folded := low | lane('a'-'A')
	letter := (folded + lane(utf8.RuneSelf-'a')) &^ (folded + lane(utf8.RuneSelf-'z'-1))
	digit := (low + lane(utf8.RuneSelf-'0')) &^ (low + lane(utf8.RuneSelf-'9'-1))
	return (letter | digit) & konst.HighBits
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func equal(w uint64, b byte) uint64 {
	diff := w ^ lane(b)
	return ^((diff&lane(utf8.RuneSelf-1) + lane(utf8.RuneSelf-1)) | diff) & konst.HighBits
}

func lane(b byte) uint64 {
	return uint64(b) * konst.LowBits
}

func lanes(s string, pos int) uint64 {
	s = s[pos : pos+konst.ByteLanes]
	return uint64(s[0]) | uint64(s[1])<<8 | uint64(s[2])<<16 | uint64(s[3])<<24 |
		uint64(s[4])<<32 | uint64(s[5])<<40 | uint64(s[6])<<48 | uint64(s[7])<<56
}

func (o *wrapper) place(seg segment) int {
	if seg.width <= o.width {
		o.put(seg)
		return o.room()
	}
	if o.length() > 0 {
		o.flush()
	}
	spaced := false
	for pos := seg.start; pos < seg.end; {
		n, cluster := o.w.next(o.s[pos:seg.end])
		switch {
		case n == 1 && o.s[pos] == ' ' && o.keep:
			o.put(segment{start: pos, end: pos + 1})
		case n == 1 && o.s[pos] == ' ':
			spaced = true
		default:
			o.put(segment{start: pos, end: pos + n, width: cluster, spaced: spaced})
			spaced = false
		}
		pos += n
	}
	return o.room()
}

func (o *wrapper) room() int {
	if o.keep || o.length() == 0 {
		return konst.NoRoom
	}
	return o.width - o.used
}

func (o *wrapper) length() int {
	if o.copied {
		return len(o.line)
	}
	return o.end - o.start
}

func (o *wrapper) put(seg segment) {
	gap := b2i(seg.spaced && o.length() > 0)
	if o.length() > 0 && o.used+gap+seg.width > o.width {
		o.flush()
		gap = 0
	}
	s := o.s[seg.start:seg.end]
	hanging := 0
	for o.keep && hanging < len(s) && s[len(s)-1-hanging] == ' ' {
		hanging++
	}
	o.used += gap + seg.width + hanging
	if o.length() == 0 {
		o.start, o.end = seg.start, seg.start
	}
	if !o.copied && o.end+gap == seg.start && (o.keep || !seg.squeeze) {
		o.end = seg.end
		return
	}
	if !o.copied {
		o.line = append(o.line[:0], o.s[o.start:o.end]...)
		o.copied = true
	}
	if gap == 1 {
		o.line = append(o.line, ' ')
	}
	for i := range len(s) {
		if i == 0 || s[i] != ' ' || s[i-1] != ' ' {
			o.line = append(o.line, s[i])
		}
	}
}

func (o *wrapper) flush() {
	for o.keep && o.used > o.width && o.end > o.start && o.s[o.end-1] == ' ' {
		o.end--
		o.used--
	}
	if o.copied {
		o.lines = append(o.lines, string(o.line))
	} else {
		o.lines = append(o.lines, o.s[o.start:o.end])
	}
	o.start, o.end, o.copied, o.used = 0, 0, false, 0
}

func MinContent(s string) int {
	return Wrapping{}.MinContent(s)
}

func Truncate(s string, width int) string {
	return Widths{}.Truncate(s, width)
}

func (w Widths) Truncate(s string, width int) string {
	if w.Width(s) <= width {
		return s
	}
	room := width - w.Width(konst.Ellipsis)
	if room < 0 {
		return ""
	}
	n, used := 0, 0
	for n < len(s) {
		size, cluster := w.next(s[n:])
		used += cluster
		if used > room {
			break
		}
		n += size
	}
	return s[:n] + konst.Ellipsis
}
