package highlight

import "strings"

func isOpen(c byte) bool  { return c == '(' || c == '[' || c == '{' }
func isClose(c byte) bool { return c == ')' || c == ']' || c == '}' }

func anglePops(t string) int {
	if len(t) > 3 || strings.Trim(t, ">") != "" {
		return 0
	}
	return len(t)
}

func (w *work) paramList(i int, arrow bool) int {
	open, off := -1, 0
	if arrow {
		open, off = w.arrowParen(i - 1)
	} else if i = w.nextSolid(w.skipGenerics(i)); w.opensWith(i, '(') {
		open = i
	}
	if open < 0 {
		return -1
	}
	return w.walkParams(open, off)
}

func (w *work) skipGenerics(i int) int {
	i = w.nextSolid(i)
	if !w.is(i, Operator, "<") {
		return i
	}
	depth, m := 1, i+1
	for ; m < len(w.spans) && depth > 0; m++ {
		if w.spans[m].Kind != Operator {
			continue
		}
		if t := w.text(m); t == "<" {
			depth++
		} else {
			depth = max(0, depth-anglePops(t))
		}
	}
	return m
}

func (w *work) matchingClose(open, off int) (int, int) {
	depth := 1
	for k := open; k < len(w.spans); k++ {
		if w.spans[k].Kind != Punctuation {
			continue
		}
		from := w.spans[k].Start
		if k == open {
			from += off + 1
		}
		for p := from; p < w.spans[k].End; p++ {
			if isOpen(w.src[p]) {
				depth++
			} else if isClose(w.src[p]) {
				if depth--; depth == 0 {
					return k, p
				}
			}
		}
	}
	return -1, 0
}

func (w *work) arrowFollows(open, off int) bool {
	k, p := w.matchingClose(open, off)
	if k < 0 || p+1 < w.spans[k].End {
		return false
	}
	j := w.nextSolid(k + 1)
	if j == len(w.spans) {
		return false
	}
	if w.is(j, Punctuation, ":") {
		depth := 0
		for m := j + 1; m < len(w.spans); m++ {
			switch w.spans[m].Kind {
			case Punctuation:
				for q := w.spans[m].Start; q < w.spans[m].End; q++ {
					switch c := w.src[q]; {
					case isOpen(c):
						depth++
					case isClose(c) && depth == 0:
						return false
					case isClose(c):
						depth--
					case (c == ',' || c == ';') && depth == 0:
						return false
					}
				}
			case Operator:
				if depth == 0 && w.text(m) == "=>" {
					return true
				}
			}
		}
		return false
	}
	return w.is(j, Operator, "=>")
}

func (w *work) arrowParen(at int) (int, int) {
	if at < 0 || w.spans[at].Kind != Punctuation {
		return -1, 0
	}
	t := w.text(at)
	for off := range len(t) {
		if t[off] != '(' {
			continue
		}
		if prev := w.prevSolid(at - 1); off == 0 && prev >= 0 && w.spans[prev].Kind == Punctuation && w.src[w.spans[prev].End-1] == ':' {
			enclosing := w.frames[0]
			if at > 0 {
				enclosing = w.frames[w.frameOf[at-1]]
			}
			ternary := prev < len(w.signals) && w.signals[prev]&ternaryColon != 0
			if before := w.prevSolid(prev - 1); ternary && before >= 0 {
				ternary = !w.is(before, Operator, "?")
			}
			if (enclosing.bracket != braceBracket || enclosing.kind != frameObject) && !ternary {
				continue
			}
		}
		if w.arrowFollows(at, off) {
			return at, off
		}
	}
	return -1, 0
}

func (w *work) nameFollows(k int) bool {
	for ; k < len(w.spans); k++ {
		switch t := w.text(k); {
		case w.trivia(k):
		case w.spans[k].Kind == Identifier:
			return true
		case !w.paramModifier(k, t):
			return false
		}
	}
	return false
}

func (w *work) paramModifier(k int, t string) bool {
	return w.spans[k].Kind == Operator && t == "..." || w.spans[k].Kind == Keyword && memberLeading(t)
}

func (w *work) walkParams(open, off int) int {
	depth, expect, defaulted := 1, true, false
	k := open
	for ; k < len(w.spans) && depth > 0; k++ {
		s := w.spans[k]
		switch {
		case w.trivia(k):
		case s.Kind == Punctuation:
			from := s.Start
			if k == open {
				from += off + 1
			}
			for p := from; p < s.End && depth > 0; p++ {
				switch c := w.src[p]; {
				case isOpen(c):
					depth++
				case isClose(c):
					depth--
				case c == ',' && depth == 1:
					expect, defaulted = true, false
				}
			}
		case depth == 1 && expect && !defaulted:
			t := w.text(k)
			switch {
			case s.Kind == Identifier:
				w.caps = append(w.caps, [2]int{k, k + 1})
			case s.Kind == Operator && t == "=":
				defaulted = true
			case w.paramModifier(k, t) && w.nameFollows(k+1):
				continue
			}
			expect = false
		}
	}
	return k
}

func valueOperator(t string) bool {
	switch t {
	case "+", "-", "*", "/", "%", "**", "==", "!=", "===", "!==", "<=", ">=", "&&", "||", "??", "?.", "+=", "-=", "*=", "/=",
		"%=", "&=", "|=", "^=", "&&=", "||=", "??=", "<<", ">>", ">>>", "<<=", ">>=", ">>>=", "**=", "++", "--":
		return true
	}
	return false
}

func statementKeyword(t string) bool {
	switch t {
	case "return", "if", "else", "for", "while", "do", "switch", "case", "break", "continue", "throw", "try", "catch", "finally",
		"function", "class", "interface", "enum", "namespace", "module", "let", "const", "var", "import", "export", "type":
		return true
	}
	return false
}

func terminalKeyword(t string) bool {
	switch t {
	case "this", "void", "undefined", "null", "never", "unknown", "any", "object":
		return true
	}
	return false
}

func (w *work) closesType(i int) bool {
	prev := w.prevSolid(i - 1)
	if prev < 0 {
		return false
	}
	switch t := w.text(prev); w.spans[prev].Kind {
	case Punctuation:
		return t[len(t)-1] == ']' || t[len(t)-1] == ')'
	case Identifier, Type:
		return true
	case Operator:
		return t == ">"
	case Keyword:
		return terminalKeyword(t)
	}
	return false
}

func (w *work) elementStart(i int) bool {
	prev := w.prevSolid(i - 1)
	if prev < 0 {
		return false
	}
	t := w.text(prev)
	if w.spans[prev].Kind == Punctuation {
		return t[len(t)-1] == '[' || t[len(t)-1] == ',' || strings.HasSuffix(t, "...")
	}
	return w.spans[prev].Kind == Operator && t == "..."
}

func (w *work) angleCloses(i int, braces bool) int {
	depth, brace := 1, 0
	w.angles = w.angles[:0]
	for j := i; j < len(w.spans); j++ {
		switch t := w.text(j); w.spans[j].Kind {
		case Operator:
			if t == "<" {
				depth++
				w.angles = append(w.angles, j)
			} else if pops := anglePops(t); pops > 0 && (brace == 0 || !braces) {
				w.angles = append(w.angles, j)
				if depth -= pops; depth <= 0 {
					return j
				}
			}
		case Punctuation:
			for p := 0; p < len(t); p++ {
				switch c := t[p]; {
				case !braces && (c == ';' || c == '}'):
					return -1
				case c == '{':
					brace++
				case c == '}' && brace == 0:
					return -1
				case c == '}':
					brace--
				case c == ';' && brace == 0:
					return -1
				}
			}
		}
	}
	return -1
}

func (w *work) genericCall(i int) bool {
	j := w.angleCloses(i+1, false)
	return j >= 0 && w.opensWith(w.nextSolid(j+1), '(')
}

func (w *work) opensWith(i int, c byte) bool {
	return i < len(w.spans) && w.spans[i].Kind == Punctuation && w.src[w.spans[i].Start] == c
}

func (w *work) typeArgsFollow(i int) bool {
	j := w.angleCloses(i, true)
	if j < 0 {
		return false
	}
	after := w.nextSolid(j + 1)
	if after == len(w.spans) {
		return true
	}
	switch t := w.text(after); w.spans[after].Kind {
	case Punctuation:
		return strings.IndexByte("(){}[],;.:", t[0]) >= 0
	case Operator:
		switch t {
		case "=", "|", "&", ">", "?", "!", "=>", "?:":
			return true
		}
	case Keyword:
		return t == "extends" || t == "implements"
	}
	return false
}

func (w *work) typeSpan(i int, m spanMode) int {
	if m.verifyAngle && !w.typeArgsFollow(i) {
		return -1
	}
	paren, brace, square, angle := 0, 0, 0, 0
	level := func() bool { return paren == 0 && brace == 0 && square == 0 && angle == 0 }
	for ; i < len(w.spans); i++ {
		t := w.text(i)
		switch w.spans[i].Kind {
		case Comment:
		case Punctuation:
			for p := 0; p < len(t); p++ {
				switch t[p] {
				case '(':
					paren++
				case ')':
					if paren--; paren < 0 {
						return i
					}
				case '{':
					if m.braceExits && brace == 0 && paren == 0 && w.closesType(i) {
						return i
					}
					brace++
				case '}':
					if brace--; brace < 0 {
						return i
					}
				case '[':
					square++
				case ']':
					if square--; square < 0 {
						return i
					}
				case ';':
					if level() {
						return i
					}
				}
			}
			if !m.keepComma && level() && strings.IndexByte(t, ',') >= 0 {
				return i
			}
		case Operator:
			if t == "<" {
				angle++
				continue
			}
			if pops := anglePops(t); pops > 0 {
				if angle >= pops {
					angle -= pops
					continue
				}
				angle = 0
				if m.enterAngle {
					return i
				}
			}
			if !level() {
				continue
			}
			switch {
			case t == "=" && !m.keepEq:
				return i
			case t == "=":
			case t == "=>":
				if prev := w.prevSolid(i - 1); prev < 0 || w.src[w.spans[prev].End-1] != ')' {
					return i
				}
			case t == "?" && m.exitQmark:
				return i
			case t == "?":
			case valueOperator(t):
				return i
			}
		case Keyword:
			if level() && statementKeyword(t) {
				return i
			}
		case Identifier:
			if !w.memberName(i, paren > 0 || brace > 0, square > 0) {
				w.caps = append(w.caps, [2]int{i, i + 1})
			}
		}
	}
	return i
}

func (w *work) memberName(i int, member, inBracket bool) bool {
	if !member && (!inBracket || !w.elementStart(i)) {
		return false
	}
	next := w.nextSolid(i + 1)
	if next == len(w.spans) {
		return false
	}
	t := w.text(next)
	switch w.spans[next].Kind {
	case Punctuation:
		return t == ":" || member && t[0] == '('
	case Operator:
		switch {
		case t == "?:":
			return true
		case t == "?":
			return w.opensWith(w.nextSolid(next+1), ':')
		case member && t == "<":
			return w.genericCall(next)
		}
	}
	return false
}
