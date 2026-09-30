package highlight

import (
	"iter"
	"slices"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/highlight"
)

type Kind uint8

const (
	Text Kind = iota
	Keyword
	String
	Escape
	Number
	Comment
	Function
	Operator
	Punctuation
	Identifier
	Property
	Boolean
	Variable
	Builtin
	Regex
	Datetime
	TableHeader
	Namespace
	Parameter
	Constant
	kindEnd
)

func (k Kind) String() string {
	return [kindEnd]string{
		Text: "text", Keyword: "keyword", String: "string", Escape: "string_escape", Number: "number",
		Comment: "comment", Function: "function", Operator: "operator", Punctuation: "punctuation",
		Identifier: "identifier", Property: "property", Boolean: "boolean", Variable: "variable",
		Builtin: "builtin", Regex: "regex", Datetime: "datetime", TableHeader: "array_table_header",
		Namespace: "namespace", Parameter: "parameter", Constant: "constant",
	}[k]
}

type Span struct {
	Kind       Kind
	Start, End int
}

type probeKey struct {
	state uint16
	rule  uint8
}

type probe struct {
	probeKey
	pos, depth int
}

type spans struct {
	yield func(Span) bool
	last  Span
}

func (o *spans) emit(k Kind, start, end int, sealed bool) bool {
	if !sealed && o.last.Kind == k && o.last.End == start {
		o.last.End = end
		return true
	}
	if !o.flush(start) {
		return false
	}
	o.last = Span{k, start, end}
	return true
}

func (o *spans) flush(upTo int) bool {
	if s := o.last; s.End > s.Start && !o.yield(s) {
		return false
	}
	return upTo <= o.last.End || o.yield(Span{Text, o.last.End, upTo})
}

func Tokens(src string, g *Grammar) iter.Seq[Span] {
	return func(yield func(Span) bool) {
		if g.reclassify == nil {
			out := spans{yield: yield}
			if g.run(src, &out) {
				out.flush(len(src))
			}
			return
		}
		g.reclassified(src, yield)
	}
}

func (g *Grammar) reclassified(src string, yield func(Span) bool) {
	w, _ := g.works.Get().(*work)
	if w == nil {
		w = &work{}
	}
	w.src, w.spans = src, w.spans[:0]
	out := spans{yield: func(s Span) bool {
		w.spans = append(w.spans, s)
		return true
	}}
	g.run(src, &out)
	out.flush(len(src))
	g.reclassify(w)
	for _, s := range w.spans {
		if !yield(s) {
			break
		}
	}
	w.src = ""
	g.works.Put(w)
}

func (g *Grammar) run(src string, out *spans) bool {
	var stack [konst.StackDepth]uint16
	depth, cur, pos := 0, uint16(0), 0
	enter := func(s uint16) {
		if depth < len(stack) {
			stack[depth] = s
			depth++
		}
	}
	var open probe
	failedAt, failed, nFailed := -1, [konst.FailedProbes]probeKey{}, 0
	failedHere := func(r uint8) bool {
		return pos == failedAt && slices.Contains(failed[:nFailed], probeKey{cur, r})
	}
	for {
		st := &g.states[cur]
		if pos >= len(src) {
			if !st.probe {
				return true
			}
			if st.atEnd != noState {
				depth = open.depth
				enter(open.state)
				cur = st.atEnd
			} else {
				if failedAt != open.pos {
					failedAt, nFailed = open.pos, 0
				}
				failed[nFailed] = open.probeKey
				nFailed++
				cur, depth = open.state, open.depth
			}
			pos = open.pos
			continue
		}
		c := src[pos]
		r, n, width := uint8(noRule), 0, 1
		if c < utf8.RuneSelf {
			for _, p := range st.patterns[st.first[c]:st.first[c+1]] {
				end := pos + len(p.text)
				if end > len(src) || src[pos+1] != p.text[1] || src[pos:end] != p.text ||
					st.rules[p.rule].boundary && end < len(src) && wordByte(src[end]) || failedHere(p.rule) {
					continue
				}
				r, n = p.rule, len(p.text)
				break
			}
			if r == noRule {
				r = st.byByte[c]
				if r != noRule && st.rules[r].boundary && pos+1 < len(src) && wordByte(src[pos+1]) {
					r = noRule
				}
			}
		} else {
			var ch rune
			ch, width = utf8.DecodeRuneInString(src[pos:])
			r = st.fallback
			for _, w := range st.wide {
				if ch >= w.lo && ch <= w.hi {
					r = w.rule
					break
				}
			}
		}
		if n == 0 && r != noRule && failedHere(r) {
			r = noRule
		}
		if r == noRule {
			pos += width
			continue
		}
		ru := st.rules[r]
		if ru.repeats && n == 0 {
			end := pos + width
			for end < len(src) && src[end] < utf8.RuneSelf && st.byByte[src[end]] == r && st.first[src[end]] == st.first[src[end]+1] {
				end++
			}
			if ru.kind != Text && !out.emit(ru.kind, pos, end, false) {
				return false
			}
			pos = end
			continue
		}
		if n > 0 {
			width = n
		}
		target := cur
		switch ru.step {
		case push, jump:
			target = ru.target
		case pop:
			if depth > 0 {
				target = stack[depth-1]
			}
		}
		if !st.probe && g.states[target].probe {
			open = probe{probeKey{cur, r}, pos, depth}
		}
		switch {
		case !st.probe && ru.kind != Text:
			if !out.emit(ru.kind, pos, pos+width, n > 0 || ru.boundary) {
				return false
			}
			pos += width
		case ru.step == stay || ru.step == push:
			pos += width
		case ru.step == jump:
			pos += n
		}
		switch ru.step {
		case push:
			enter(cur)
		case pop:
			depth = max(depth-1, 0)
		}
		cur = target
		if st.probe && !g.states[cur].probe {
			pos, depth = open.pos, open.depth
			if ru.step == push {
				enter(open.state)
			}
		}
	}
}

func wordByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '$'
}
