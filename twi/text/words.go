package text

import (
	"iter"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/text"
)

type wordClass byte

const (
	wOther wordClass = iota
	wCR
	wLF
	wNewline
	wExtend
	wFormat
	wZWJ
	wRegional
	wKatakana
	wHebrew
	wLetter
	wSingleQuote
	wDoubleQuote
	wMidNumLet
	wMidLetter
	wMidNum
	wNumeric
	wExtendNumLet
	wSpace
)

func (c wordClass) ignored() bool { return c == wExtend || c == wFormat || c == wZWJ }

func (c wordClass) letter() bool { return c == wLetter || c == wHebrew }

func (c wordClass) alnum() bool { return c.letter() || c == wNumeric }

func (c wordClass) midLetter() bool { return c == wMidLetter || c == wMidNumLet || c == wSingleQuote }

func (c wordClass) midNum() bool { return c == wMidNum || c == wMidNumLet || c == wSingleQuote }

func wordOf(r rune) (wordClass, bool) {
	rec := record(r)
	return wordClass(rec[3]), rec[1]&konst.PictographicBit != 0
}

func Words(s string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for s != "" {
			n := word(s)
			if !yield(s[:n]) {
				return
			}
			s = s[n:]
		}
	}
}

func word(s string) int {
	r, n := utf8.DecodeRuneInString(s)
	last, _ := wordOf(r)
	switch {
	case last == wCR && n < len(s) && s[n] == '\n':
		return n + 1
	case last == wCR, last == wLF, last == wNewline:
		return n
	}
	before, raw, regionals := wOther, last, 0
	if last == wRegional {
		regionals = 1
	}
	for n < len(s) {
		r, size := utf8.DecodeRuneInString(s[n:])
		cur, pictographic := wordOf(r)
		switch {
		case cur == wCR, cur == wLF, cur == wNewline:
			return n
		case raw == wZWJ && pictographic, raw == wSpace && cur == wSpace, cur.ignored():
		case !wordJoins(before, last, cur, s[n+size:], regionals):
			return n
		}
		n += size
		raw = cur
		if cur.ignored() {
			continue
		}
		before, last = last, cur
		if cur == wRegional {
			regionals++
		} else {
			regionals = 0
		}
	}
	return n
}

func wordJoins(before, last, cur wordClass, rest string, regionals int) bool {
	return last.alnum() && cur.alnum() ||
		last.letter() && cur.midLetter() && following(rest).letter() ||
		before.letter() && last.midLetter() && cur.letter() ||
		last == wHebrew && cur == wSingleQuote ||
		last == wHebrew && cur == wDoubleQuote && following(rest) == wHebrew ||
		before == wHebrew && last == wDoubleQuote && cur == wHebrew ||
		before == wNumeric && last.midNum() && cur == wNumeric ||
		last == wNumeric && cur.midNum() && following(rest) == wNumeric ||
		last == wKatakana && cur == wKatakana ||
		(last.alnum() || last == wKatakana || last == wExtendNumLet) && cur == wExtendNumLet ||
		last == wExtendNumLet && (cur.alnum() || cur == wKatakana) ||
		last == wRegional && cur == wRegional && regionals%2 == 1
}

func following(s string) wordClass {
	for _, r := range s {
		if c, _ := wordOf(r); !c.ignored() {
			return c
		}
	}
	return wOther
}
