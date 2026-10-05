package highlight

import "strings"

type scriptRules struct {
	reserved, functionValue, property, typePosition, parameters []rewriteRule
}

func newScriptRules() scriptRules {
	colon := anyOf(tok(Punctuation, ":"), tok(Operator, "?:"), seq(tok(Operator, "?"), tok(Punctuation, ":")))
	arrow := tok(Operator, "=>")
	async := tok(Keyword, "async")
	functionValue := seq(optional(async), anyOf(tok(Keyword, "function"), seq(tok(Identifier), arrow), seq(parens(), arrow)))
	assigned := seq(tok(Operator, "="), anyOf(
		anyOf(tok(Keyword, "function"), seq(async, tok(Keyword, "function"))),
		anyOf(seq(parens(), arrow), seq(async, parens(), arrow), seq(tok(Identifier), arrow), seq(async, tok(Identifier), arrow)),
	))
	notStatement := not(Keyword, "for", "while", "do", "if", "switch", "try", "with")
	named := []frameKind{frameObject, frameInterface, frameTypeLiteral}
	methods := []frameKind{frameClass, frameInterface, frameObject}
	members := []frameKind{frameClass, frameObject, frameInterface}
	call := parens()
	star := capture(tok(Operator, "*"))
	walk := tokenPattern{op: opParams}
	arrowWalk := tokenPattern{op: opParams, arrow: true}
	declared := seq(optional(tok(Operator, "*")), optional(anyOf(tok(Identifier), tok(Function))), walk)
	method := func(k Kind, texts ...string) rewriteRule {
		return rewriteRule{kind: k, texts: texts, atStart: true, frames: members, when: &walk, to: Parameter, captured: true}
	}
	annotation := tokenPattern{op: opSpan}
	returns := tokenPattern{op: opSpan, span: spanMode{braceExits: true}}
	heritage := tokenPattern{op: opSpan, span: spanMode{keepComma: true, braceExits: true}}
	cast := tokenPattern{op: opSpan, span: spanMode{exitQmark: true}}
	generics := tokenPattern{op: opSpan, span: spanMode{enterAngle: true, keepComma: true, keepEq: true, verifyAngle: true}}
	closeParen := tok(Punctuation, ")")
	afterLess := seq(tok(Operator, "<"), tok(Identifier))
	afterComma := seq(tok(Punctuation, ","), tok(Identifier))
	name := tok(Identifier)
	greater := tok(Operator, ">")
	typed := func(r rewriteRule, p *tokenPattern) rewriteRule {
		r.when, r.to, r.captured = p, Type, true
		return r
	}
	return scriptRules{
		reserved: []rewriteRule{
			{kind: Keyword, texts: []string{"new"}, atStart: true, frames: []frameKind{frameInterface}, when: &call, to: Keyword},
			{kind: Keyword, atStart: true, frames: methods, when: &call, to: Function},
			{kind: Keyword, atStart: true, frames: named, when: &colon, to: Identifier},
			{kind: Boolean, atStart: true, frames: named, when: &colon, to: Identifier},
			{kind: Keyword, texts: []string{"function", "yield", "static", "async"}, when: &star, to: Keyword, captured: true},
			{kind: Operator, texts: []string{"*"}, atStart: true, frames: []frameKind{frameClass, frameObject}, to: Keyword},
		},
		functionValue: []rewriteRule{{kind: Identifier, when: &assigned, to: Function}},
		property: []rewriteRule{
			{kind: Identifier, atStart: true, frames: []frameKind{frameObject}, when: ptr(seq(colon, functionValue)), to: Function},
			{kind: Identifier, atStart: true, frames: named, when: ptr(seq(colon, notStatement)), to: Property},
		},
		parameters: []rewriteRule{
			{kind: Keyword, texts: []string{"function"}, when: &declared, to: Parameter, captured: true},
			method(Identifier),
			method(Function),
			method(Keyword, "get", "set", "async", "readonly", "public", "private", "protected", "static", "abstract", "override",
				"accessor", "declare"),
			{kind: Punctuation, when: &arrowWalk, to: Parameter, captured: true},
			{kind: Identifier, when: &arrow, to: Parameter},
			{kind: Function, when: &arrow, to: Parameter},
		},
		typePosition: []rewriteRule{
			typed(rewriteRule{kind: Punctuation, suffixes: []string{"):"}, notTernary: true}, &returns),
			typed(rewriteRule{kind: Punctuation, texts: []string{":"}, notTernary: true, before: &closeParen}, &returns),
			typed(rewriteRule{kind: Punctuation, suffixes: []string{":"}, notTernary: true, frames: []frameKind{frameParen}}, &annotation),
			typed(rewriteRule{kind: Punctuation, suffixes: []string{":"}, notTernary: true, frames: []frameKind{frameClass, frameInterface}}, &annotation),
			typed(rewriteRule{kind: Punctuation, suffixes: []string{":"}, notTernary: true, frames: []frameKind{frameTop}, flags: varDecl}, &annotation),
			typed(rewriteRule{kind: Keyword, texts: []string{"as", "satisfies"}}, &cast),
			typed(rewriteRule{kind: Keyword, texts: []string{"extends"}, flags: ifaceHead}, &heritage),
			typed(rewriteRule{kind: Keyword, texts: []string{"extends"}, before: &afterLess}, &heritage),
			typed(rewriteRule{kind: Keyword, texts: []string{"extends"}, before: &afterComma}, &heritage),
			typed(rewriteRule{kind: Keyword, texts: []string{"implements"}}, &heritage),
			typed(rewriteRule{kind: Operator, texts: []string{"<"}, before: &name}, &generics),
			typed(rewriteRule{kind: Operator, texts: []string{"="}, flags: aliasHead, frames: []frameKind{frameTop}, before: &name}, &annotation),
			typed(rewriteRule{kind: Operator, texts: []string{"="}, flags: aliasHead, frames: []frameKind{frameTop}, before: &greater}, &annotation),
		},
	}
}

func ptr(p tokenPattern) *tokenPattern { return &p }

const constBindingRank int8 = 25

type embeds struct {
	css, html, jsdoc *Grammar
}

func scriptPipeline(ts bool, rules scriptRules, sub embeds) func(*work) {
	return func(w *work) {
		w.trackFrames(ts)
		w.openClaims()
		w.rewrite(rules.reserved)
		w.applyClaims()
		w.openClaims()
		w.constants()
		if ts {
			w.namespaces()
			w.builtinTypes()
		}
		w.rewrite(rules.functionValue)
		w.constBindings()
		w.rewrite(rules.property)
		if !ts {
			w.classNames()
			w.rewrite(rules.parameters)
			w.namespaces()
			w.applyClaims()
			w.embedGroups(sub)
			return
		}
		w.tupleLabels()
		w.rewrite(rules.typePosition)
		w.applyClaims()
		w.typeOnlyBindings()
		w.genericCalls()
		w.openClaims()
		w.classNames()
		w.rewrite(rules.parameters)
		w.applyClaims()
		w.retagAngles()
		w.embedGroups(sub)
	}
}

func (w *work) solid(i int) int {
	if i < 0 {
		return -1
	}
	if i = w.nextSolid(i); i == len(w.spans) {
		return -1
	}
	return i
}

func (w *work) at(i int, k Kind, text string) bool {
	return i >= 0 && i < len(w.spans) && w.spans[i].Kind == k && w.text(i) == text
}

func (w *work) constants() {
	for i, s := range w.spans {
		if s.Kind == Identifier && upperSnake(w.text(i)) {
			w.emit(i, Constant, rank(Constant))
		}
	}
}

func (w *work) builtinTypes() {
	for i, s := range w.spans {
		if s.Kind != Identifier {
			continue
		}
		switch w.text(i) {
		case "number", "string", "boolean", "any", "never", "unknown", "object", "symbol", "bigint":
			w.emit(i, Type, rank(Type))
		}
	}
}

func (w *work) moduleStar(i int) int {
	j := w.solid(i + 1)
	if j < 0 {
		return -1
	}
	if k := w.solid(j + 1); w.at(k, Punctuation, ",") {
		j = w.solid(k + 1)
	} else if w.text(j) == "type" {
		j = k
	}
	if !w.at(j, Operator, "*") {
		return -1
	}
	return j
}

func (w *work) namespaces() {
	for i, s := range w.spans {
		if s.Kind != Keyword {
			continue
		}
		name := -1
		switch w.text(i) {
		case "import", "export":
			if star := w.moduleStar(i); star >= 0 {
				if as := w.solid(star + 1); w.at(as, Keyword, "as") {
					name = w.solid(as + 1)
				}
			}
		case "namespace", "module":
			name = w.solid(i + 1)
		}
		if name >= 0 && w.spans[name].Kind == Identifier {
			w.emit(name, Namespace, rank(Namespace))
		}
	}
}

func (w *work) constBindings() {
	for i, s := range w.spans {
		if s.Kind != Keyword {
			continue
		}
		if t := w.text(i); t == "import" || t == "export" {
			if star := w.moduleStar(i); star >= 0 {
				w.emit(star, Constant, rank(Constant))
			}
			continue
		} else if t != "const" {
			continue
		}
		w.objects = w.objects[:0]
		depth, angle, binding, skipping, skipBase := 0, 0, true, false, 0
		inObject := func() bool { return depth > 0 && w.objects[len(w.objects)-1] }
	walk:
		for k := w.nextSolid(i + 1); k < len(w.spans); k++ {
			t := w.text(k)
			switch w.spans[k].Kind {
			case Comment:
				continue
			case Punctuation:
				for p := 0; p < len(t); p++ {
					switch c := t[p]; c {
					case '{', '[', '(':
						depth++
						w.objects = append(w.objects, c == '{')
						binding = c != '('
					case '}', ']', ')':
						if depth == 0 {
							break walk
						}
						depth--
						w.objects = w.objects[:len(w.objects)-1]
						if skipping && depth < skipBase {
							skipping = false
						}
						binding = false
					case ',':
						if skipping && depth == skipBase && angle == 0 {
							skipping = false
						}
						if !skipping {
							binding = true
						}
					case ';':
						if depth == 0 {
							break walk
						}
					case ':':
						if inObject() {
							binding = true
						} else if !skipping {
							skipping, skipBase = true, depth
						}
					default:
						binding = false
					}
				}
				continue
			case Operator:
				switch {
				case t == "<":
					angle++
				case t == ">" || t == ">>" || t == ">>>":
					angle = max(0, angle-len(t))
				case t == "=" && !skipping:
					skipping, skipBase = true, depth
				case t == "..." && !skipping:
					binding = true
					continue
				}
				binding = false
				continue
			case Keyword:
				if (t == "of" || t == "in") && depth == 0 {
					break walk
				}
			case Identifier:
				if !skipping && binding && (!inObject() || !w.at(w.solid(k+1), Punctuation, ":")) {
					w.emit(k, Constant, constBindingRank)
				}
			}
			binding = false
		}
	}
}

func (w *work) nameLike(i int) bool {
	return i >= 0 && i < len(w.spans) && (w.spans[i].Kind == Identifier || w.spans[i].Kind == Type || w.spans[i].Kind == Function)
}

func (w *work) skipAngles(from int) int {
	if !w.at(from, Operator, "<") {
		return from
	}
	depth, j := 1, from+1
	for ; j < len(w.spans) && depth > 0; j++ {
		if w.spans[j].Kind == Operator {
			switch w.text(j) {
			case "<":
				depth++
			case ">":
				depth--
			}
		}
	}
	return j
}

func (w *work) chainLast(from int) int {
	j, last := w.solid(from), -1
	for w.nameLike(j) {
		last = j
		if j = w.solid(j + 1); !w.at(j, Punctuation, ".") {
			break
		}
		j = w.solid(j + 1)
	}
	if last >= 0 {
		w.emit(last, ClassName, rank(ClassName))
	}
	if j < 0 {
		return len(w.spans)
	}
	return j
}

func (w *work) classList(from int) int {
	j := w.solid(from)
	for w.nameLike(j) {
		j = w.solid(w.chainLast(j))
		if w.at(j, Operator, "<") {
			j = w.solid(w.skipAngles(j))
		}
		if !w.at(j, Punctuation, ",") {
			break
		}
		j = w.solid(j + 1)
	}
	if j < 0 {
		return len(w.spans)
	}
	return j
}

func (w *work) classNames() {
	for i, s := range w.spans {
		if s.Kind != Keyword {
			continue
		}
		switch w.text(i) {
		case "class", "interface":
			j := w.solid(i + 1)
			if w.nameLike(j) {
				w.emit(j, ClassName, rank(ClassName))
				j = w.solid(j + 1)
			}
			if w.at(j, Operator, "<") {
				j = w.solid(w.skipAngles(j))
			}
			for range 2 {
				if j = w.solid(j); j >= 0 && (w.at(j, Keyword, "extends") || w.at(j, Keyword, "implements")) {
					j = w.classList(j + 1)
					continue
				}
				break
			}
		case "new", "instanceof":
			w.chainLast(i + 1)
		}
	}
}

func (w *work) typeOnlyBindings() {
	for i, s := range w.spans {
		if s.Kind != Keyword {
			continue
		}
		switch w.text(i) {
		case "type":
			if prev := w.prevSolid(i - 1); prev >= 0 && !w.statementEnd(prev) {
				continue
			}
			if j := w.solid(i + 1); j >= 0 && w.spans[j].Kind == Identifier {
				w.spans[j].Kind = Type
			}
		case "import", "export":
			j := w.solid(i + 1)
			if !w.at(j, Keyword, "type") {
				continue
			}
			k := w.solid(j + 1)
			switch {
			case k < 0 || w.at(k, Operator, "*"):
			case w.spans[k].Kind == Punctuation && w.src[w.spans[k].Start] == '{':
				w.typeList(k)
			case w.spans[k].Kind == Identifier && w.text(i) == "import":
				w.spans[k].Kind = Type
			}
		}
	}
}

func (w *work) statementEnd(i int) bool {
	t := w.text(i)
	switch w.spans[i].Kind {
	case Punctuation:
		return t[len(t)-1] == ';' || t[len(t)-1] == '}'
	case Keyword:
		return t == "export" || t == "declare"
	}
	return false
}

func (w *work) typeList(start int) {
	depth := strings.Count(w.text(start), "{") - strings.Count(w.text(start), "}")
	for m := start + 1; m < len(w.spans) && depth > 0; m++ {
		switch w.spans[m].Kind {
		case Punctuation:
			for p := w.spans[m].Start; p < w.spans[m].End; p++ {
				if c := w.src[p]; c == '{' {
					depth++
				} else if c == '}' {
					if depth--; depth == 0 {
						break
					}
				}
			}
		case Identifier:
			w.spans[m].Kind = Type
		}
	}
}

func (w *work) nextLess(from int) int {
	for i := from; i < len(w.spans); i++ {
		if w.at(i, Operator, "<") {
			return i
		}
	}
	return -1
}

func (w *work) genericCalls() {
	for lt := w.nextLess(0); lt >= 0; lt = w.nextLess(lt + 1) {
		i := w.prevSolid(lt - 1)
		if i < 0 || w.spans[i].Kind != Identifier {
			continue
		}
		if end := w.angleCloses(lt+1, true); end >= 0 && w.opensWith(w.solid(end+1), '(') {
			w.spans[i].Kind = Function
		}
	}
}

func (w *work) retagAngles() {
	for i := w.nextLess(0); i >= 0; {
		prev := w.prevSolid(i - 1)
		k := Text
		if prev >= 0 {
			k = w.spans[prev].Kind
		}
		end := -1
		if k == Identifier || k == Type || k == ClassName || k == Function {
			end = w.angleCloses(i+1, true)
		}
		if end < 0 || !w.typeArgsEnd(w.solid(end+1)) {
			i = w.nextLess(i + 1)
			continue
		}
		w.spans[i].Kind = Punctuation
		for _, a := range w.angles {
			w.spans[a].Kind = Punctuation
		}
		i = w.nextLess(end + 1)
	}
}

func (w *work) typeArgsEnd(after int) bool {
	if after < 0 {
		return true
	}
	t := w.text(after)
	switch w.spans[after].Kind {
	case Punctuation:
		return strings.IndexByte("(){}[],;.:", t[0]) >= 0
	case Operator:
		switch t {
		case "=", "=>", "?:", "|", "&", ">", ">>", ">>>", "?", "!":
			return true
		}
	case Keyword:
		return t == "extends" || t == "implements"
	}
	return false
}

func (w *work) labelColon(i int) bool {
	next := w.solid(i + 1)
	if next < 0 {
		return false
	}
	t := w.text(next)
	switch w.spans[next].Kind {
	case Punctuation:
		return t == ":"
	case Operator:
		return t == "?:" || t == "?" && w.opensWith(w.solid(next+1), ':')
	}
	return false
}

func (w *work) indexSignature(f frame, i int) bool {
	if f.parent < 0 || w.frames[f.parent].bracket != braceBracket {
		return false
	}
	if w.src[w.spans[f.enter].Start] == '[' && w.at(w.prevSolid(f.enter-1), Operator, "?") {
		return false
	}
	depth := 1
	for j := i + 1; j < len(w.spans); j++ {
		if w.spans[j].Kind != Punctuation {
			continue
		}
		for p := w.spans[j].Start; p < w.spans[j].End; p++ {
			switch w.src[p] {
			case '[':
				depth++
			case ']':
				if depth--; depth == 0 {
					if p+1 < w.spans[j].End {
						return w.src[p+1] == ':'
					}
					return w.opensWith(w.solid(j+1), ':')
				}
			}
		}
	}
	return false
}

func (w *work) tupleLabels() {
	for i, s := range w.spans {
		f := w.frames[w.frameOf[i]]
		if s.Kind != Identifier || f.bracket != squareBracket || !w.labelColon(i) {
			continue
		}
		prev := w.prevSolid(i - 1)
		if prev < 0 {
			continue
		}
		t := w.text(prev)
		switch {
		case w.spans[prev].Kind == Punctuation && t[len(t)-1] == '[':
			if w.indexSignature(f, i) {
				continue
			}
		case w.spans[prev].Kind == Punctuation && (t[len(t)-1] == ',' || strings.HasSuffix(t, "...")):
		case w.spans[prev].Kind == Operator && t == "...":
		default:
			continue
		}
		w.emit(i, Property, rank(Property))
	}
}
