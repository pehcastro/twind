package text

import (
	"fmt"
	"strings"

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

func (w Widths) MinContent(s string) int {
	return Wrapping{Widths: w}.MinContent(s)
}

func (b Wrapping) MinContent(s string) int {
	widest := 0
	emit := func(_, _, width int, _ bool) { widest = max(widest, width) }
	if b.Overflow == OverflowWrapAnywhere {
		emit = func(start, end, _ int, _ bool) {
			for pos := start; pos < end; {
				n, cluster, _ := b.Widths.next(s[pos:end])
				widest = max(widest, cluster)
				pos += n
			}
		}
	}
	from := 0
	for paragraph := range strings.SplitSeq(s, "\n") {
		b.segments(s, from, from+len(paragraph), emit)
		from += len(paragraph) + 1
	}
	return widest
}

func (b Wrapping) segments(s string, from, to int, emit func(start, end, width int, spaced bool)) {
	if b.Word > WordBreakKeepAll || b.Overflow > OverflowWrapAnywhere || b.Space > SpacePreserve {
		panic(fmt.Sprintf("text: unknown word-break %d, overflow-wrap %d or white-space %d", b.Word, b.Overflow, b.Space))
	}
	keep := b.Space == SpacePreserve
	var before lineClass
	start, end, segWidth, kept := from, from, 0, 0
	var segSpaced, spaced, leadingHyphen bool
	for pos := from; pos < to; {
		n, width, after := b.Widths.next(s[pos:to])
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
			emit(start, end, segWidth, segSpaced)
			start, segWidth, segSpaced = pos, 0, spaced && !keep
		} else if spaced {
			segWidth += max(kept, 1)
		}
		segWidth += width
		leadingHyphen = after == hyphen && (spaced || end == from)
		pos += n
		before, spaced, kept = after, false, 0
		for before <= numeric && pos < to && printableByte(s[pos:to]) {
			class := lookup(rune(s[pos])).line
			if class > numeric {
				break
			}
			before = class
			pos++
			segWidth++
		}
		end = pos
	}
	emit(start, end, segWidth, segSpaced)
}

func (o *wrapper) place(start, end, width int, spaced bool) {
	if width <= o.width {
		o.put(start, end, width, spaced)
		return
	}
	if o.length() > 0 {
		o.flush()
	}
	spaced = false
	for pos := start; pos < end; {
		n, cluster, _ := o.w.next(o.s[pos:end])
		switch {
		case n == 1 && o.s[pos] == ' ' && o.keep:
			o.put(pos, pos+1, 0, false)
		case n == 1 && o.s[pos] == ' ':
			spaced = true
		default:
			o.put(pos, pos+n, cluster, spaced)
			spaced = false
		}
		pos += n
	}
}

func (o *wrapper) length() int {
	if o.copied {
		return len(o.line)
	}
	return o.end - o.start
}

func (o *wrapper) put(start, end, width int, spaced bool) {
	gap := 0
	if spaced && o.length() > 0 {
		gap = 1
	}
	if o.length() > 0 && o.used+gap+width > o.width {
		o.flush()
		gap = 0
	}
	s := o.s[start:end]
	hanging := 0
	for o.keep && hanging < len(s) && s[len(s)-1-hanging] == ' ' {
		hanging++
	}
	o.used += gap + width + hanging
	if o.length() == 0 {
		o.start, o.end = start, start
	}
	if !o.copied && o.end+gap == start && (o.keep || !strings.Contains(s, "  ")) {
		o.end = end
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
		size, cluster, _ := w.next(s[n:])
		used += cluster
		if used > room {
			break
		}
		n += size
	}
	return s[:n] + konst.Ellipsis
}
