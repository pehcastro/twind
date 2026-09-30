package text

import (
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
	w     Widths
	width int
	lines []string
	line  []byte
	used  int
}

func Wrap(s string, width int) []string {
	return Widths{}.Wrap(s, width)
}

func (w Widths) Wrap(s string, width int) []string {
	out := wrapper{w: w, width: width}
	for paragraph := range strings.SplitSeq(s, "\n") {
		w.segments(paragraph, out.place)
		out.flush()
	}
	return out.lines
}

func (w Widths) MinContent(s string) int {
	widest := 0
	for paragraph := range strings.SplitSeq(s, "\n") {
		w.segments(paragraph, func(_ string, width int, _ bool) { widest = max(widest, width) })
	}
	return widest
}

func (w Widths) segments(paragraph string, emit func(segment string, width int, spaced bool)) {
	var before lineClass
	var pos, start, end, segWidth int
	var segSpaced, spaced, leadingHyphen bool
	for cluster := range Graphemes(paragraph) {
		at := pos
		pos += len(cluster)
		if cluster == " " {
			spaced = true
			continue
		}
		r, _ := utf8.DecodeRuneInString(cluster)
		after := lookup(r).line
		if end == 0 || lineBreaks(before, after, spaced, leadingHyphen) {
			emit(paragraph[start:end], segWidth, segSpaced)
			start, segWidth, segSpaced = at, 0, spaced
		} else if spaced {
			segWidth++
		}
		segWidth += w.clusterWidth(cluster)
		leadingHyphen = after == hyphen && (spaced || end == 0)
		end, before, spaced = pos, after, false
	}
	emit(paragraph[start:end], segWidth, segSpaced)
}

func (o *wrapper) place(segment string, width int, spaced bool) {
	if width <= o.width {
		o.put(segment, width, spaced)
		return
	}
	if len(o.line) > 0 {
		o.flush()
	}
	spaced = false
	for cluster := range Graphemes(segment) {
		if cluster == " " {
			spaced = true
			continue
		}
		o.put(cluster, o.w.clusterWidth(cluster), spaced)
		spaced = false
	}
}

func (o *wrapper) put(s string, width int, spaced bool) {
	gap := 0
	if spaced && len(o.line) > 0 {
		gap = 1
	}
	if len(o.line) > 0 && o.used+gap+width > o.width {
		o.flush()
		gap = 0
	}
	if gap == 1 {
		o.line = append(o.line, ' ')
	}
	for i := range len(s) {
		if i == 0 || s[i] != ' ' || s[i-1] != ' ' {
			o.line = append(o.line, s[i])
		}
	}
	o.used += gap + width
}

func (o *wrapper) flush() {
	o.lines = append(o.lines, string(o.line))
	o.line = o.line[:0]
	o.used = 0
}

func MinContent(s string) int {
	return Widths{}.MinContent(s)
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
	for cluster := range Graphemes(s) {
		used += w.clusterWidth(cluster)
		if used > room {
			break
		}
		n += len(cluster)
	}
	return s[:n] + konst.Ellipsis
}
