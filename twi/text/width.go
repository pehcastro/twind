package text

import (
	"iter"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/text"
)

//go:generate go run gen.go -version 17.0.0

type breakClass byte

const (
	other breakClass = iota
	cr
	lf
	control
	extend
	zwj
	regional
	prepend
	spacingMark
	hangulL
	hangulV
	hangulT
	hangulLV
	hangulLVT
)

type conjunct byte

const (
	conjunctNone conjunct = iota
	conjunctConsonant
	conjunctExtend
	conjunctLinker
)

type sequence byte

const (
	outside sequence = iota
	opened
	joining
)

const (
	textPresentation  = '\U0000FE0E'
	emojiPresentation = '\U0000FE0F'
	zeroWidthJoiner   = '\U0000200D'
	keycapMark        = '\U000020E3'
	modifierFirst     = '\U0001F3FB'
	modifierLast      = '\U0001F3FF'
)

type Class uint8

const (
	Flag Class = iota
	ZWJ
	VS16
	Modifier
	Keycap
	Classes
)

type Widths [Classes]int

type props struct {
	class        breakClass
	conjunct     conjunct
	pictographic bool
	width        int
	line         lineClass
}

func record(r rune) string {
	id := int(blocks[int(blockIndex[r>>konst.BlockShift])<<konst.BlockShift|int(r&(konst.BlockSize-1))])
	return records[id*konst.RecordSize:]
}

func lookup(r rune) props {
	record := record(r)
	flags := record[1]
	return props{
		class:        breakClass(record[0]),
		conjunct:     conjunct(flags >> konst.ConjunctShift & konst.ConjunctMask),
		pictographic: flags&konst.PictographicBit != 0,
		width:        int(flags & konst.WidthMask),
		line:         lineClass(record[2]),
	}
}

func Graphemes(s string) iter.Seq[string] {
	return func(yield func(string) bool) {
		var w Widths
		for s != "" {
			n, _, _ := w.next(s)
			if !yield(s[:n]) {
				return
			}
			s = s[n:]
		}
	}
}

func (w *Widths) next(s string) (n, width int, line lineClass) {
	if printableByte(s) {
		return 1, 1, lookup(rune(s[0])).line
	}
	return w.cluster(s)
}

func printableByte(s string) bool {
	return s[0] >= ' ' && s[0] <= '~' && (len(s) == 1 || s[1] < utf8.RuneSelf)
}

func (w *Widths) cluster(s string) (n, width int, line lineClass) {
	var first, prev props
	var runes, regionals int
	var emoji, indic sequence
	var vs15, vs16, joined, keycap, modifier bool
	for n < len(s) {
		r, size := utf8.DecodeRuneInString(s[n:])
		cur := lookup(r)
		if n > 0 && !joins(prev, cur, regionals, emoji, indic) {
			break
		}
		if n == 0 {
			first = cur
		}
		width = max(width, cur.width)
		switch r {
		case textPresentation:
			vs15 = true
		case emojiPresentation:
			vs16 = true
		case zeroWidthJoiner:
			joined = true
		case keycapMark:
			keycap = true
		}
		modifier = modifier || r >= modifierFirst && r <= modifierLast
		if cur.class == regional {
			regionals++
		} else {
			regionals = 0
		}
		switch {
		case cur.pictographic:
			emoji = opened
		case emoji == opened && cur.class == extend:
		case emoji == opened && cur.class == zwj:
			emoji = joining
		default:
			emoji = outside
		}
		switch {
		case cur.conjunct == conjunctConsonant:
			indic = opened
		case indic != outside && cur.conjunct == conjunctLinker:
			indic = joining
		case indic != outside && cur.conjunct == conjunctExtend:
		default:
			indic = outside
		}
		runes++
		n += size
		prev = cur
	}
	switch {
	case width > 0 && vs16:
		width = 2
	case width > 0 && vs15:
		width = 1
	}
	c := Classes
	switch {
	case runes == 1:
	case first.class == regional:
		c = Flag
	case first.pictographic && joined:
		c = ZWJ
	case keycap:
		c = Keycap
	case first.pictographic && modifier:
		c = Modifier
	case vs16:
		c = VS16
	}
	if c != Classes && w[c] > 0 {
		width = w[c]
	}
	return n, width, first.line
}

func joins(prev, cur props, regionals int, emoji, indic sequence) bool {
	switch {
	case prev.class == cr && cur.class == lf:
		return true
	case prev.class == cr, prev.class == lf, prev.class == control, cur.class == cr, cur.class == lf, cur.class == control:
		return false
	case prev.class == hangulL && (cur.class == hangulL || cur.class == hangulV || cur.class == hangulLV || cur.class == hangulLVT):
		return true
	case (prev.class == hangulLV || prev.class == hangulV) && (cur.class == hangulV || cur.class == hangulT):
		return true
	case (prev.class == hangulLVT || prev.class == hangulT) && cur.class == hangulT:
		return true
	case cur.class == extend, cur.class == zwj, cur.class == spacingMark, prev.class == prepend:
		return true
	case cur.conjunct == conjunctConsonant && indic == joining:
		return true
	case cur.pictographic && emoji == joining:
		return true
	}
	return prev.class == regional && cur.class == regional && regionals%2 == 1
}

func Width(s string) int {
	return Widths{}.Width(s)
}

func (w Widths) Width(s string) int {
	total := 0
	for s != "" {
		n, width, _ := w.next(s)
		total += width
		s = s[n:]
	}
	return total
}
