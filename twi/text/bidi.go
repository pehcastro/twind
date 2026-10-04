package text

import (
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/text"
)

type Direction uint8

const (
	DirLTR Direction = iota
	DirRTL
	DirAuto
)

type bidiClass uint8

const (
	bL bidiClass = iota
	bR
	bAL
	bEN
	bES
	bET
	bAN
	bCS
	bNSM
	bBN
	bB
	bS
	bWS
	bON
	bLRE
	bLRO
	bRLE
	bRLO
	bPDF
	bLRI
	bRLI
	bFSI
	bPDI
)

func (c bidiClass) isolate() bool { return c == bLRI || c == bRLI || c == bFSI }

func (c bidiClass) neutral() bool {
	return c == bB || c == bS || c == bWS || c == bON || c.isolate() || c == bPDI
}

func (c bidiClass) strong() bidiClass {
	switch c {
	case bL:
		return bL
	case bR, bAL, bEN, bAN:
		return bR
	}
	return bON
}

func bidiOf(r rune) byte {
	return bidiBlocks[int(bidiIndex[r>>konst.BlockShift])<<konst.BlockShift|int(r&(konst.BlockSize-1))]
}

func pairOf(table string, r rune) rune {
	i := sort.Search(len(table)/konst.PairSize, func(m int) bool { return runeAt(table, m*konst.PairSize) >= r })
	return runeAt(table, i*konst.PairSize+konst.PairRuneBytes)
}

func runeAt(table string, at int) rune {
	return rune(table[at])<<16 | rune(table[at+1])<<8 | rune(table[at+2])
}

type paragraph struct {
	runes   []rune
	initial []bidiClass
	classes []bidiClass
	levels  []uint8
	match   []int
	level   uint8
}

type unit struct {
	from, to int
	level    uint8
}

type bracketPair struct {
	open, close int
}

type opener struct {
	key rune
	at  int
}

type entry struct {
	level    uint8
	override bidiClass
	isolate  bool
}

func (p *paragraph) resolve(s string, dir Direction) {
	p.runes = []rune(s)
	n := len(p.runes)
	p.initial = make([]bidiClass, n)
	for i, r := range p.runes {
		p.initial[i] = bidiClass(bidiOf(r) & konst.BidiClassMask)
	}
	p.classes = slices.Clone(p.initial)
	p.levels = make([]uint8, n)
	p.match = make([]int, n)
	var open []int
	for i, c := range p.initial {
		p.match[i] = -1
		switch {
		case c.isolate():
			open = append(open, i)
		case c == bPDI && len(open) > 0:
			p.match[i], p.match[open[len(open)-1]] = open[len(open)-1], i
			open = open[:len(open)-1]
		}
	}
	if dir == DirRTL || dir == DirAuto && p.strong(0, n) == bR {
		p.level = 1
	}
	p.explicit()
	kept := make([]int, 0, n)
	runAt := make([]int, n)
	var runs []int
	for i, c := range p.classes {
		if c == bBN {
			continue
		}
		if len(kept) == 0 || p.levels[i] != p.levels[kept[len(kept)-1]] {
			runs = append(runs, len(kept))
		}
		runAt[i] = len(runs) - 1
		kept = append(kept, i)
	}
	runs = append(runs, len(kept))
	var seq []int
	for r := range len(runs) - 1 {
		if first := kept[runs[r]]; p.initial[first] == bPDI && p.match[first] >= 0 {
			continue
		}
		seq = seq[:0]
		for run := r; ; {
			seq = append(seq, kept[runs[run]:runs[run+1]]...)
			last := seq[len(seq)-1]
			if !p.initial[last].isolate() || p.match[last] < 0 {
				break
			}
			run = runAt[p.match[last]]
		}
		p.sequence(seq)
	}
	level := p.level
	for i, c := range p.classes {
		odd := p.levels[i]&1 == 1
		switch {
		case c == bBN:
			p.levels[i] = level
		case !odd && c == bR, odd && (c == bL || c == bEN || c == bAN):
			p.levels[i]++
		case !odd && (c == bAN || c == bEN):
			p.levels[i] += 2
		}
		level = p.levels[i]
	}
}

func (p *paragraph) strong(from, to int) bidiClass {
	for i := from; i < to; i++ {
		switch c := p.initial[i]; {
		case c == bL, c == bR, c == bAL:
			return c.strong()
		case c.isolate() && p.match[i] < 0:
			return bON
		case c.isolate():
			i = p.match[i]
		}
	}
	return bON
}

func (p *paragraph) explicit() {
	var stack [konst.BidiMaxDepth + 2]entry
	depth := 0
	stack[0] = entry{level: p.level, override: bON}
	overflowIsolates, overflowEmbeddings, validIsolates := 0, 0, 0
	for i, c := range p.initial {
		top := stack[depth]
		p.levels[i] = top.level
		switch c {
		case bRLE, bLRE, bRLO, bLRO, bRLI, bLRI, bFSI:
			isolate := c.isolate()
			if !isolate {
				p.classes[i] = bBN
			} else if top.override != bON {
				p.classes[i] = top.override
			}
			to := len(p.runes)
			if p.match[i] >= 0 {
				to = p.match[i]
			}
			next := (top.level + 2) &^ 1
			if c == bRLE || c == bRLO || c == bRLI || c == bFSI && p.strong(i+1, to) == bR {
				next = (top.level + 1) | 1
			}
			override := bON
			switch c {
			case bRLO:
				override = bR
			case bLRO:
				override = bL
			}
			switch {
			case next <= konst.BidiMaxDepth && overflowIsolates == 0 && overflowEmbeddings == 0:
				validIsolates += b2i(isolate)
				depth++
				stack[depth] = entry{level: next, override: override, isolate: isolate}
			case isolate:
				overflowIsolates++
			case overflowIsolates == 0:
				overflowEmbeddings++
			}
		case bPDI:
			switch {
			case overflowIsolates > 0:
				overflowIsolates--
			case validIsolates > 0:
				overflowEmbeddings = 0
				for !stack[depth].isolate {
					depth--
				}
				depth--
				validIsolates--
			}
			top = stack[depth]
			p.levels[i] = top.level
			if top.override != bON {
				p.classes[i] = top.override
			}
		case bPDF:
			p.classes[i] = bBN
			switch {
			case overflowIsolates > 0:
			case overflowEmbeddings > 0:
				overflowEmbeddings--
			case !top.isolate && depth > 0:
				depth--
			}
		case bB:
			p.levels[i] = p.level
		case bBN:
		default:
			if top.override != bON {
				p.classes[i] = top.override
			}
		}
	}
}

func (p *paragraph) sequence(seq []int) {
	c := p.classes
	level := p.levels[seq[0]]
	prev, next := p.level, p.level
	for i := seq[0] - 1; i >= 0; i-- {
		if c[i] != bBN {
			prev = p.levels[i]
			break
		}
	}
	if last := seq[len(seq)-1]; !p.initial[last].isolate() {
		for i := last + 1; i < len(c); i++ {
			if c[i] != bBN {
				next = p.levels[i]
				break
			}
		}
	}
	sos, eos, e := bidiClass(max(prev, level)&1), bidiClass(max(next, level)&1), bidiClass(level&1)
	for k, i := range seq {
		if c[i] != bNSM {
			continue
		}
		switch {
		case k == 0:
			c[i] = sos
		case c[seq[k-1]].isolate() || c[seq[k-1]] == bPDI:
			c[i] = bON
		default:
			c[i] = c[seq[k-1]]
		}
	}
	strong := sos
	for _, i := range seq {
		switch c[i] {
		case bL, bR:
			strong = c[i]
		case bAL:
			strong, c[i] = bAL, bR
		case bEN:
			if strong == bAL {
				c[i] = bAN
			}
		}
	}
	for k := 1; k+1 < len(seq); k++ {
		before, after := c[seq[k-1]], c[seq[k+1]]
		switch c[seq[k]] {
		case bES:
			if before == bEN && after == bEN {
				c[seq[k]] = bEN
			}
		case bCS:
			if before == after && (before == bEN || before == bAN) {
				c[seq[k]] = before
			}
		}
	}
	for k := 0; k < len(seq); {
		end := k
		for end < len(seq) && c[seq[end]] == bET {
			end++
		}
		if end == k {
			k++
			continue
		}
		if k > 0 && c[seq[k-1]] == bEN || end < len(seq) && c[seq[end]] == bEN {
			for _, i := range seq[k:end] {
				c[i] = bEN
			}
		}
		k = end
	}
	strong = sos
	for _, i := range seq {
		switch c[i] {
		case bES, bET, bCS:
			c[i] = bON
		case bL, bR:
			strong = c[i]
		case bEN:
			if strong == bL {
				c[i] = bL
			}
		}
	}
	for _, pair := range p.brackets(seq) {
		dir := bON
		for _, i := range seq[pair.open+1 : pair.close] {
			if s := c[i].strong(); s == e {
				dir = e
				break
			} else if s != bON {
				dir = s
			}
		}
		if dir == bON {
			continue
		}
		if dir != e {
			before := sos
			for k := pair.open - 1; k >= 0; k-- {
				if s := c[seq[k]].strong(); s != bON {
					before = s
					break
				}
			}
			if before != dir {
				dir = e
			}
		}
		for _, k := range []int{pair.open, pair.close} {
			c[seq[k]] = dir
			for k++; k < len(seq) && p.initial[seq[k]] == bNSM; k++ {
				c[seq[k]] = dir
			}
		}
	}
	for k := 0; k < len(seq); {
		end := k
		for end < len(seq) && c[seq[end]].neutral() {
			end++
		}
		if end == k {
			k++
			continue
		}
		before, after := sos, eos
		if k > 0 {
			before = c[seq[k-1]].strong()
		}
		if end < len(seq) {
			after = c[seq[end]].strong()
		}
		dir := e
		if before == after {
			dir = before
		}
		for _, i := range seq[k:end] {
			c[i] = dir
		}
		k = end
	}
}

func (p *paragraph) brackets(seq []int) []bracketPair {
	var pairs []bracketPair
	var openers []opener
scan:
	for k, i := range seq {
		if p.classes[i] != bON {
			continue
		}
		bits := bidiOf(p.runes[i])
		switch {
		case bits&konst.BidiOpenBit != 0 && len(openers) == konst.BracketDepth:
			break scan
		case bits&konst.BidiOpenBit != 0:
			openers = append(openers, opener{pairOf(brackets, p.runes[i]), k})
		case bits&konst.BidiCloseBit != 0:
			key := pairOf(brackets, p.runes[i])
			for d := len(openers) - 1; d >= 0; d-- {
				if openers[d].key == key {
					pairs = append(pairs, bracketPair{openers[d].at, k})
					openers = openers[:d]
					break
				}
			}
		}
	}
	slices.SortFunc(pairs, func(a, b bracketPair) int { return a.open - b.open })
	return pairs
}

func (p *paragraph) line(ks []int, levels []uint8) {
	trailing := true
	for j := len(ks) - 1; j >= 0; j-- {
		switch c := p.initial[ks[j]]; {
		case c == bS, c == bB:
			levels[j], trailing = p.level, true
		case trailing && (c == bWS || c == bBN || c.isolate() || c == bPDI || c >= bLRE && c <= bPDF):
			levels[j] = p.level
		default:
			levels[j], trailing = p.levels[ks[j]], false
		}
	}
}

func reorder(units []unit) bool {
	highest, lowestOdd := uint8(0), uint8(konst.BidiMaxDepth+2)
	for _, u := range units {
		highest = max(highest, u.level)
		if u.level&1 == 1 {
			lowestOdd = min(lowestOdd, u.level)
		}
	}
	for level := highest; level >= lowestOdd; level-- {
		for i := 0; i < len(units); {
			if units[i].level < level {
				i++
				continue
			}
			j := i
			for j < len(units) && units[j].level >= level {
				j++
			}
			slices.Reverse(units[i:j])
			i = j
		}
	}
	return lowestOdd <= highest
}

func (b Wrapping) visual(lines []string, s string) {
	var p paragraph
	p.resolve(s, b.Dir)
	var ks []int
	var levels []uint8
	var units []unit
	q, k := 0, 0
	for li, line := range lines {
		ks = ks[:0]
		for j := 0; j < len(line); {
			for s[q] != line[j] {
				_, skipped := utf8.DecodeRuneInString(s[q:])
				q, k = q+skipped, k+1
			}
			_, size := utf8.DecodeRuneInString(line[j:])
			ks = append(ks, k)
			q, k, j = q+size, k+1, j+size
		}
		levels = slices.Grow(levels[:0], len(ks))[:len(ks)]
		p.line(ks, levels)
		units = units[:0]
		for j, r := 0, 0; j < len(line); {
			n, _ := b.Widths.next(line[j:])
			units = append(units, unit{from: j, to: j + n, level: levels[r]})
			r += utf8.RuneCountInString(line[j : j+n])
			j += n
		}
		if !reorder(units) {
			continue
		}
		var out strings.Builder
		out.Grow(len(line))
		for _, u := range units {
			for j := u.from; j < u.to; {
				r, size := utf8.DecodeRuneInString(line[j:])
				if u.level&1 == 1 && bidiOf(r)&konst.BidiMirrorBit != 0 {
					out.WriteRune(pairOf(mirrors, r))
				} else {
					out.WriteString(line[j : j+size])
				}
				j += size
			}
		}
		lines[li] = out.String()
	}
}
