package highlight

import (
	"slices"
	"strings"
)

const (
	paramScanTokens = 64
	callScanTokens  = 200
)

type work struct {
	src     string
	spans   []Span
	claims  []Kind
	chunk   []int
	pending []int
}

func (w *work) text(i int) string { return w.src[w.spans[i].Start:w.spans[i].End] }

func (w *work) next(i int) int {
	for i < len(w.spans) && (w.spans[i].Kind == Text || w.spans[i].Kind == Comment) {
		i++
	}
	return i
}

func (w *work) opens(i int, c byte) bool {
	return i < len(w.spans) && w.spans[i].Kind == Punctuation && w.src[w.spans[i].Start] == c
}

func (w *work) named(i int) bool {
	return i < len(w.spans) && (w.spans[i].Kind == Identifier || w.spans[i].Kind == Function)
}

func goPasses(w *work) {
	sp := w.spans
	w.claims = slices.Grow(w.claims[:0], len(sp))[:len(sp)]
	clear(w.claims)
	for i := range sp {
		if sp[i].Kind == Identifier {
			if n := w.next(i + 1); w.opens(n, '(') || w.opens(n, '[') && w.parenAfterClose(n) {
				w.claim(i, Function)
			} else if upperSnake(w.text(i)) {
				w.claim(i, Constant)
			}
			continue
		}
		if sp[i].Kind != Keyword {
			continue
		}
		switch w.text(i) {
		case "func":
			if n := w.next(i + 1); w.named(n) && w.params(n+1) >= 0 {
				continue
			}
			if end := w.params(i + 1); end >= 0 {
				if n := w.next(end); w.named(n) {
					w.params(n + 1)
				}
			}
		case "package":
			if n := w.next(i + 1); n < len(sp) && sp[n].Kind == Identifier {
				w.claim(n, Namespace)
			}
		case "import":
			w.imports(i)
		}
	}
	for i, k := range w.claims {
		if k != Text {
			sp[i].Kind = k
		}
	}
}

func (w *work) claim(i int, k Kind) {
	if precedence(k) > precedence(w.claims[i]) {
		w.claims[i] = k
	}
}

func precedence(k Kind) int {
	switch k {
	case Constant:
		return 1
	case Function:
		return 2
	case Parameter:
		return 3
	case Namespace:
		return 4
	}
	return 0
}

func upperSnake(t string) bool {
	if len(t) < 2 || t[0] < 'A' || t[0] > 'Z' {
		return false
	}
	for i := 1; i < len(t); i++ {
		if c := t[i]; (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}

func (w *work) parenAfterClose(n int) bool {
	depth := 0
	for k := n; k < len(w.spans); k++ {
		s := w.spans[k]
		if s.Kind != Punctuation {
			continue
		}
		for p := s.Start; p < s.End; p++ {
			switch w.src[p] {
			case '[':
				depth++
			case ']':
				if depth--; depth == 0 {
					if p+1 < s.End {
						return w.src[p+1] == '('
					}
					return w.opens(w.next(k+1), '(')
				}
			}
		}
	}
	return false
}

func (w *work) params(start int) int {
	sp := w.spans
	k := w.next(start)
	if k == len(sp) || sp[k].Kind != Punctuation {
		return -1
	}
	bracket, brace := 0, 0
	for budget := paramScanTokens; k < len(sp) && budget > 0; k++ {
		if sp[k].Kind == Text {
			continue
		}
		budget--
		if sp[k].Kind != Punctuation {
			continue
		}
		for p := sp[k].Start; p < sp[k].End; p++ {
			switch w.src[p] {
			case '[':
				bracket++
			case '{':
				brace++
			case ']':
				if bracket == 0 {
					return -1
				}
				bracket--
			case '}':
				if brace == 0 {
					return -1
				}
				brace--
			case '(':
				if bracket == 0 && brace == 0 {
					return w.walk(k, p+1)
				}
			case ')':
				if bracket == 0 && brace == 0 {
					return -1
				}
			}
		}
	}
	return -1
}

func (w *work) walk(k, from int) int {
	sp := w.spans
	paren, bracket, brace := 1, 0, 0
	w.pending = w.pending[:0]
	for ; k < len(sp) && paren > 0; k++ {
		switch sp[k].Kind {
		case Text, Comment:
			continue
		case Punctuation:
		default:
			w.chunk = append(w.chunk, k)
			continue
		}
		include := false
		for p := max(from, sp[k].Start); p < sp[k].End; p++ {
			c := w.src[p]
			if c == ',' && paren == 1 && bracket == 0 && brace == 0 {
				if include {
					w.chunk = append(w.chunk, k)
				}
				w.endChunk()
				include = false
				continue
			}
			switch c {
			case '(':
				paren++
			case ')':
				paren--
			case '[':
				bracket++
			case ']':
				bracket = max(0, bracket-1)
			case '{':
				brace++
			case '}':
				brace = max(0, brace-1)
			}
			if paren == 0 {
				break
			}
			include = true
		}
		if include {
			w.chunk = append(w.chunk, k)
		}
	}
	w.endChunk()
	return k
}

func (w *work) endChunk() {
	chunk := w.chunk
	w.chunk = w.chunk[:0]
	if len(chunk) == 0 {
		return
	}
	switch first := chunk[0]; {
	case !w.named(first):
		w.pending = w.pending[:0]
	case w.typed(chunk):
		for _, i := range w.pending {
			w.claim(i, Parameter)
		}
		w.claim(first, Parameter)
		w.pending = w.pending[:0]
	case len(chunk) == 1:
		w.pending = append(w.pending, first)
	default:
		w.pending = w.pending[:0]
	}
}

func (w *work) typed(chunk []int) bool {
	if len(chunk) < 2 {
		return false
	}
	if w.spans[chunk[1]].Kind != Punctuation {
		return true
	}
	switch w.src[w.spans[chunk[1]].Start] {
	case '.':
		return false
	case '[':
		depth := 0
		for pos, k := range chunk[1:] {
			if w.spans[k].Kind != Punctuation {
				continue
			}
			t := w.text(k)
			for off := range len(t) {
				switch {
				case t[off] == '[':
					depth++
				case t[off] == ']' && depth > 0:
					if depth--; depth == 0 {
						return strings.Trim(t[off+1:], "),") != "" || pos+1 < len(chunk)-1
					}
				}
			}
		}
	}
	return true
}

func (w *work) imports(i int) {
	sp := w.spans
	j := w.next(i + 1)
	group := j < len(sp) && sp[j].Kind == Punctuation && w.text(j) == "("
	if group {
		j = w.next(j + 1)
	}
	for j < len(sp) {
		k, t, after := sp[j].Kind, w.text(j), w.next(j+1)
		switch {
		case k == Punctuation && t == ")":
			return
		case k == Punctuation && t == ";" && group:
			j = after
		case k == Punctuation && t == ";":
			return
		case k == Identifier && after < len(sp) && sp[after].Kind == String:
			w.claim(j, Namespace)
			j = w.next(after + 1)
		case k == Identifier && !group:
			return
		case k == String:
			j = after
		case group:
			j++
		default:
			return
		}
	}
}

func bashPasses(words map[string]Kind) func(*work) {
	return func(w *work) {
		w.extendVariables()
		w.mergeNumbers()
		for i, s := range w.spans {
			if s.Kind != Identifier {
				continue
			}
			if k, ok := words[w.text(i)]; ok {
				w.spans[i].Kind = k
			} else if w.balanced(i + 1) {
				w.spans[i].Kind = Function
			}
		}
	}
}

func identByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_'
}

func (w *work) extendVariables() {
	sp, out := w.spans, w.spans[:0]
	for i := 0; i < len(sp); i++ {
		s := sp[i]
		if s.Kind != Variable || i+1 == len(sp) || sp[i+1].Kind == Text {
			out = append(out, s)
			continue
		}
		next := sp[i+1]
		p := next.Start
		for p < next.End && identByte(w.src[p]) {
			p++
		}
		if p == next.Start {
			out = append(out, s)
			continue
		}
		s.End, next.Start = p, p
		out = append(out, s)
		if p < next.End {
			out = append(out, next)
		}
		i++
	}
	w.spans = out
}

func (w *work) mergeNumbers() {
	sp, out := w.spans, w.spans[:0]
	joins := func(i int) bool { return i < len(sp) && (sp[i].Kind == Number || sp[i].Kind == Identifier) }
	for i := 0; i < len(sp); i++ {
		s := sp[i]
		switch t := w.src[s.Start:s.End]; {
		case s.Kind != Number:
		case (strings.HasPrefix(t, "0x") || strings.HasPrefix(t, "0X")) && joins(i+1):
			for joins(i + 1) {
				i++
			}
			s.End = sp[i].End
		case w.opens(i+1, '#') && sp[i+1].End-sp[i+1].Start == 1 && joins(i+2):
			i += 2
			s.End = sp[i].End
		}
		out = append(out, s)
	}
	w.spans = out
}

func (w *work) balanced(j int) bool {
	sp := w.spans
	if j < len(sp) && sp[j].Kind == Text {
		j++
	}
	if !w.opens(j, '(') {
		return false
	}
	depth := 0
	for budget := 1 + callScanTokens; j < len(sp) && budget > 0; j++ {
		if sp[j].Kind == Text {
			continue
		}
		budget--
		if sp[j].Kind != Punctuation {
			continue
		}
		for p := sp[j].Start; p < sp[j].End; p++ {
			switch w.src[p] {
			case '(':
				depth++
			case ')':
				if depth--; depth == 0 {
					return true
				}
			}
		}
	}
	return false
}
