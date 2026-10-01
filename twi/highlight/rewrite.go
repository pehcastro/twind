package highlight

import (
	"slices"
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/highlight"
)

type tokenPatternOp uint8

const (
	opType tokenPatternOp = iota
	opSeq
	opAny
	opOptional
	opCapture
	opBalanced
	opNot
	opParams
	opSpan
)

type spanMode struct {
	keepComma, keepEq, exitQmark, enterAngle, braceExits, verifyAngle bool
}

type tokenPattern struct {
	op    tokenPatternOp
	kind  Kind
	texts []string
	parts []tokenPattern
	arrow bool
	span  spanMode
}

func tok(k Kind, texts ...string) tokenPattern {
	return tokenPattern{op: opType, kind: k, texts: texts}
}

func seq(parts ...tokenPattern) tokenPattern { return tokenPattern{op: opSeq, parts: parts} }

func anyOf(parts ...tokenPattern) tokenPattern { return tokenPattern{op: opAny, parts: parts} }

func optional(p tokenPattern) tokenPattern {
	return tokenPattern{op: opOptional, parts: []tokenPattern{p}}
}

func capture(p tokenPattern) tokenPattern {
	return tokenPattern{op: opCapture, parts: []tokenPattern{p}}
}

func parens() tokenPattern { return tokenPattern{op: opBalanced} }

func not(k Kind, texts ...string) tokenPattern { return tokenPattern{op: opNot, kind: k, texts: texts} }

type rewriteRule struct {
	kind       Kind
	texts      []string
	suffixes   []string
	atStart    bool
	frames     []frameKind
	notTernary bool
	flags      uint8
	before     *tokenPattern
	when       *tokenPattern
	to         Kind
	captured   bool
}

func (w *work) trivia(i int) bool { return w.spans[i].Kind == Comment }

func (w *work) nextSolid(i int) int {
	for i < len(w.spans) && w.trivia(i) {
		i++
	}
	return i
}

func (w *work) prevSolid(i int) int {
	for i >= 0 && w.trivia(i) {
		i--
	}
	return i
}

func (w *work) matches(i int, k Kind, texts []string) bool {
	return w.spans[i].Kind == k && (texts == nil || slices.Contains(texts, w.text(i)))
}

func (w *work) match(p *tokenPattern, i int) int {
	switch p.op {
	case opType:
		i = w.nextSolid(i)
		if i == len(w.spans) || !w.matches(i, p.kind, p.texts) {
			return -1
		}
		return i + 1
	case opSeq:
		for k := range p.parts {
			if i = w.match(&p.parts[k], i); i < 0 {
				return -1
			}
		}
		return i
	case opAny:
		mark := len(w.caps)
		for k := range p.parts {
			if end := w.match(&p.parts[k], i); end >= 0 {
				return end
			}
			w.caps = w.caps[:mark]
		}
		return -1
	case opOptional:
		mark := len(w.caps)
		if end := w.match(&p.parts[0], i); end >= 0 {
			return end
		}
		w.caps = w.caps[:mark]
		return i
	case opCapture:
		start := w.nextSolid(i)
		end := w.match(&p.parts[0], start)
		if end >= 0 {
			w.caps = append(w.caps, [2]int{start, end})
		}
		return end
	case opBalanced:
		return w.balanced(w.nextSolid(i))
	case opNot:
		i = w.nextSolid(i)
		if i < len(w.spans) && w.matches(i, p.kind, p.texts) {
			return -1
		}
		return i
	case opParams:
		return w.paramList(i, p.arrow)
	}
	return w.typeSpan(i, p.span)
}

func endsWithAny(t string, suffixes []string) bool {
	for _, s := range suffixes {
		if strings.HasSuffix(t, s) {
			return true
		}
	}
	return false
}

func (w *work) stepBack(p *tokenPattern, i int) int {
	if i = w.prevSolid(i); i < 0 || w.spans[i].Kind != p.kind || p.texts != nil && !endsWithAny(w.text(i), p.texts) {
		return -2
	}
	return i - 1
}

func (w *work) matchesBack(p *tokenPattern, i int) bool {
	if p.op != opSeq {
		return w.stepBack(p, i) >= -1
	}
	for k := len(p.parts) - 1; k >= 0 && i >= -1; k-- {
		i = w.stepBack(&p.parts[k], i)
	}
	return i >= -1
}

func (w *work) balanced(i int) int {
	if !w.opensWith(i, '(') {
		return -1
	}
	depth := 0
	for k := i; k < min(len(w.spans), i+1+konst.CallScanTokens); k++ {
		if w.spans[k].Kind != Punctuation {
			continue
		}
		for p := w.spans[k].Start; p < w.spans[k].End; p++ {
			switch w.src[p] {
			case '(':
				depth++
			case ')':
				if depth--; depth == 0 {
					return k + 1
				}
			}
		}
	}
	return -1
}

func rank(k Kind) int8 {
	switch k {
	case Parameter:
		return 15
	case Property:
		return 20
	case Function:
		return 30
	case Type:
		return 45
	case Namespace:
		return 46
	case ClassName:
		return 50
	case Constant:
		return 55
	case Keyword:
		return 70
	}
	return 0
}

func (w *work) openClaims() {
	n := len(w.spans)
	w.claims = slices.Grow(w.claims[:0], n)[:n]
	w.ranks = slices.Grow(w.ranks[:0], n)[:n]
	clear(w.claims)
}

func (w *work) emit(i int, k Kind, r int8) {
	if w.claims[i] == Text || r > w.ranks[i] {
		w.claims[i], w.ranks[i] = k, r
	}
}

func (w *work) applyClaims() {
	for i, k := range w.claims {
		if k != Text {
			w.spans[i].Kind = k
		}
	}
}

func (w *work) anchored(r *rewriteRule, i int) bool {
	t := w.text(i)
	switch {
	case r.texts != nil && !slices.Contains(r.texts, t):
		return false
	case r.suffixes != nil && !endsWithAny(t, r.suffixes):
		return false
	case r.atStart && !w.atStart[i]:
		return false
	case r.frames != nil && !slices.Contains(r.frames, w.frames[w.frameOf[i]].kind):
		return false
	case !r.notTernary && r.flags == 0:
		return true
	case i >= len(w.signals):
		return false
	}
	s := w.signals[i]
	return (!r.notTernary || s&ternaryColon == 0) && s&(r.flags<<1) == r.flags<<1
}

func (w *work) rewrite(rules []rewriteRule) {
	for i := range w.spans {
		if w.trivia(i) {
			continue
		}
		for k := range rules {
			r := &rules[k]
			if r.kind != w.spans[i].Kind || !w.anchored(r, i) || r.before != nil && !w.matchesBack(r.before, i-1) {
				continue
			}
			w.caps = w.caps[:0]
			if r.when != nil && w.match(r.when, i+1) < 0 {
				continue
			}
			if !r.captured {
				w.caps = append(w.caps[:0], [2]int{i, i + 1})
			}
			for _, c := range w.caps {
				for t := c[0]; t < c[1]; t++ {
					w.emit(t, r.to, rank(r.to))
				}
			}
			break
		}
	}
}
