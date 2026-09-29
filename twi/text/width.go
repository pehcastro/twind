package text

import (
	"iter"
	"strings"
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
)

type props struct {
	class        breakClass
	conjunct     conjunct
	pictographic bool
	width        int
}

func lookup(r rune) props {
	if r >= ' ' && r <= '~' {
		return props{width: 1}
	}
	lo, hi := 0, len(table)/konst.RecordSize
	for lo < hi {
		m := int(uint(lo+hi) >> 1)
		record := table[m*konst.RecordSize:]
		switch {
		case r < rune(record[0])<<16|rune(record[1])<<8|rune(record[2]):
			hi = m
		case r > rune(record[3])<<16|rune(record[4])<<8|rune(record[5]):
			lo = m + 1
		default:
			flags := record[7]
			return props{
				class:        breakClass(record[6]),
				conjunct:     conjunct(flags >> konst.ConjunctShift & konst.ConjunctMask),
				pictographic: flags&konst.PictographicBit != 0,
				width:        int(flags & konst.WidthMask),
			}
		}
	}
	return props{width: 1}
}

func Graphemes(s string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for s != "" {
			n := clusterLen(s)
			if !yield(s[:n]) {
				return
			}
			s = s[n:]
		}
	}
}

func clusterLen(s string) int {
	var prev props
	var regionals int
	var emoji, indic sequence
	for pos := 0; pos < len(s); {
		r, n := utf8.DecodeRuneInString(s[pos:])
		cur := lookup(r)
		if pos > 0 && !joins(prev, cur, regionals, emoji, indic) {
			return pos
		}
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
		pos += n
		prev = cur
	}
	return len(s)
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
	total := 0
	for cluster := range Graphemes(s) {
		width := 0
		for _, r := range cluster {
			width = max(width, lookup(r).width)
		}
		switch {
		case width > 0 && strings.ContainsRune(cluster, emojiPresentation):
			width = 2
		case width > 0 && strings.ContainsRune(cluster, textPresentation):
			width = 1
		}
		total += width
	}
	return total
}
