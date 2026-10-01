package highlight

import (
	"slices"
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/highlight"
)

type frameKind uint8

const (
	frameTop frameKind = iota
	frameParen
	frameBracket
	frameClass
	frameInterface
	frameBlock
	frameTypeLiteral
	frameObject
)

type bracket uint8

const (
	_ bracket = iota
	parenBracket
	braceBracket
	squareBracket
)

type frame struct {
	bracket bracket
	kind    frameKind
	enter   int
	parent  int
}

type frameSave struct {
	frame   int
	atStart bool
	qmarks  int
	flags   uint8
}

const (
	ternaryColon uint8 = 1
	varDecl      uint8 = 1
	ifaceHead    uint8 = 2
	aliasHead    uint8 = 4
)

func (w *work) caseLabel(j int) bool {
	for range konst.BackScanSteps {
		if j = w.prevSolid(j); j < 0 {
			return false
		}
		switch t := w.text(j); w.spans[j].Kind {
		case Keyword:
			if t == "case" || t == "default" {
				return true
			}
			if t != "this" && t != "null" && t != "undefined" {
				return false
			}
		case Identifier, String, Number, Boolean:
		case Operator:
			if t != "-" {
				return false
			}
		case Punctuation:
			if strings.Trim(t, ".") != "" {
				return false
			}
		default:
			return false
		}
		j--
	}
	return false
}

func (w *work) returnType(j int) bool {
	for range konst.BackScanSteps {
		if j = w.prevSolid(j); j < 0 {
			return false
		}
		switch t := w.text(j); w.spans[j].Kind {
		case Punctuation:
			if t[len(t)-1] == ':' {
				if len(t) >= 2 {
					return t[len(t)-2] == ')'
				}
				k := w.prevSolid(j - 1)
				return k >= 0 && w.src[w.spans[k].End-1] == ')'
			}
			if strings.Trim(t, ".[],") != "" {
				return false
			}
		case Identifier, Type, String, Number, Boolean:
		case Keyword:
			switch t {
			case "void", "null", "undefined", "this", "typeof", "keyof", "readonly", "infer", "is", "asserts":
			default:
				return false
			}
		case Operator:
			switch t {
			case "|", "&", "<", ">", ">>", ">>>":
			default:
				return false
			}
		default:
			return false
		}
		j--
	}
	return false
}

func (w *work) braceKind(prev, parent int, before byte, standIn bool) frameKind {
	if standIn {
		switch {
		case before == '$':
			return frameBlock
		case before == ':' && w.frames[parent].kind == frameBlock && w.caseLabel(prev):
			return frameBlock
		case before == ':':
			return frameTypeLiteral
		case before == ')':
			return frameBlock
		case before == ']' && w.returnType(prev):
			return frameBlock
		}
		return frameObject
	}
	if prev < 0 {
		return frameBlock
	}
	t := w.text(prev)
	last := t[len(t)-1]
	switch w.spans[prev].Kind {
	case Operator:
		if t == "=>" || (t == ">" || t == ">>" || t == ">>>") && w.returnType(prev-1) {
			return frameBlock
		}
	case Punctuation:
		switch {
		case last == '$':
			return frameBlock
		case t == ":" && w.frames[parent].kind == frameBlock && w.caseLabel(prev-1):
			return frameBlock
		case t == ":":
			return frameTypeLiteral
		case last == ')':
			return frameBlock
		case last == ']' && w.returnType(prev-1):
			return frameBlock
		}
	case Keyword:
		if t == "do" || t == "try" || t == "else" || t == "finally" ||
			(t == "void" || t == "null" || t == "undefined" || t == "this") && w.returnType(prev-1) {
			return frameBlock
		}
	case Identifier, Type:
		if w.returnType(prev - 1) {
			return frameBlock
		}
	}
	return frameObject
}

func memberLeading(t string) bool {
	switch t {
	case "get", "set", "async", "readonly", "public", "private", "protected", "static", "abstract", "override", "accessor", "declare":
		return true
	}
	return false
}

func statementStart(t string) bool {
	switch t {
	case "if", "else", "for", "while", "do", "switch", "case", "break", "continue", "throw", "try", "catch", "finally",
		"function", "class", "interface", "enum", "namespace", "module", "import", "export", "return":
		return true
	}
	return false
}

func (w *work) trackFrames(signals bool) {
	n := len(w.spans)
	w.frames = append(w.frames[:0], frame{kind: frameTop, enter: -1, parent: -1})
	w.frameOf = slices.Grow(w.frameOf[:0], n)[:n]
	w.atStart = slices.Grow(w.atStart[:0], n)[:n]
	w.signals = w.signals[:0]
	if signals {
		w.signals = slices.Grow(w.signals, n)[:n]
	}
	w.saves = w.saves[:0]
	cur, atStart, qmarks, flags := 0, true, 0, uint8(0)
	paren, brace, square, angle := 0, 0, 0, 0
	pending, pendingDepth := frameTop, -1
	prev, afterQmark := -1, false
	open := func(b bracket, kind frameKind, i int) {
		w.saves = append(w.saves, frameSave{cur, false, qmarks, flags})
		w.frames = append(w.frames, frame{bracket: b, kind: kind, enter: i, parent: cur})
		cur, atStart, qmarks, flags = len(w.frames)-1, b == braceBracket, 0, 0
	}
	closeFrame := func() bool {
		if len(w.saves) == 0 {
			return false
		}
		s := w.saves[len(w.saves)-1]
		w.saves = w.saves[:len(w.saves)-1]
		cur, atStart, qmarks, flags = s.frame, s.atStart, s.qmarks, s.flags
		return true
	}
	for i, sp := range w.spans {
		t := w.text(i)
		trivia := sp.Kind == Comment
		w.atStart[i] = atStart
		see := atStart && (sp.Kind == Keyword && (memberLeading(t) || t == "class" || t == "interface") || sp.Kind == Operator && t == "*")
		switch {
		case sp.Kind == Keyword && t == "class":
			pending, pendingDepth = frameClass, brace
		case sp.Kind == Keyword && t == "interface":
			pending, pendingDepth = frameInterface, brace
		case sp.Kind == Operator && t == "<":
			angle++
		case sp.Kind == Operator && strings.Trim(t, ">") == "" && len(t) <= 3:
			angle = max(0, angle-len(t))
		}
		signal, qmark := uint8(0), false
		if signals {
			switch {
			case sp.Kind == Operator && t == "?":
				qmarks++
				qmark = true
			case sp.Kind == Keyword && (t == "let" || t == "const" || t == "var"):
				flags |= varDecl
			case sp.Kind == Keyword && t == "interface":
				flags = (flags | ifaceHead) &^ (varDecl | aliasHead)
			case sp.Kind == Keyword && t == "type":
				flags |= aliasHead
			case sp.Kind == Keyword && statementStart(t):
				flags = 0
			}
		}
		if sp.Kind != Punctuation && !trivia && !see {
			atStart = false
		}
		for p := sp.Start; sp.Kind == Punctuation && p < sp.End; p++ {
			switch c := w.src[p]; c {
			case '(':
				open(parenBracket, frameParen, i)
				paren++
			case ')':
				closeFrame()
				paren = max(0, paren-1)
				atStart = false
			case '[':
				open(squareBracket, frameBracket, i)
				square++
			case ']':
				closeFrame()
				square = max(0, square-1)
				atStart = false
			case '{':
				var kind frameKind
				switch {
				case pending != frameTop && angle == 0 && paren == 0 && square == 0:
					kind, pending = pending, frameTop
				case pending != frameTop && angle > 0:
					kind = frameTypeLiteral
				default:
					var before byte
					if p > sp.Start {
						before = w.src[p-1]
					}
					kind = w.braceKind(prev, cur, before, p > sp.Start)
				}
				open(braceBracket, kind, i)
				brace++
			case '}':
				angle = 0
				popped := closeFrame()
				if popped && signals {
					flags = 0
				}
				brace = max(0, brace-1)
				atStart = popped && (w.frames[cur].kind == frameClass || w.frames[cur].kind == frameInterface)
			default:
				if signals && c == ':' && qmarks > 0 {
					qmarks--
					if !afterQmark || p != sp.Start {
						signal |= ternaryColon
					}
				} else if signals && c == ';' {
					flags = 0
				}
				if c == ';' {
					angle = 0
				}
				if pending != frameTop && brace == pendingDepth && strings.IndexByte(";,:", c) >= 0 {
					pending = frameTop
				}
				atStart = c == ',' || c == ';'
			}
		}
		if !trivia {
			prev, afterQmark = i, qmark
		}
		w.frameOf[i] = cur
		if signals {
			w.signals[i] = signal | flags<<1
		}
	}
}
