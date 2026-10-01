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
	Selector
	SelectorClass
	SelectorID
	SelectorPseudo
	Attribute
	CSSVariable
	Unit
	Null
	CharEscape
	HardBreak
	Heading
	HeadingMarker
	Bold
	Italic
	Strike
	Code
	LinkText
	Autolink
	URL
	URLLink
	URLTitle
	Entity
	CodeBlock
	CodeFence
	CodeLanguage
	RawCodeBlock
	FrontMatterMarker
	RawFrontMatter
	BlockquoteMarker
	ListMarker
	TaskMarker
	ThematicBreak
	Prompt
	PromptPrefix
	Output
	Template
	Decorator
	Type
	ClassName
	TagName
	AttrName
	Doctype
	boldOpen
	boldClose
	italicOpen
	italicClose
	strikeOpen
	strikeClose
	codeOpen
	codeClose
	linkTextOpen
	linkTextClose
	autolinkOpen
	autolinkClose
	rawShell
	rawScript
	rawStyle
	kindEnd
)

func (k Kind) String() string {
	return [kindEnd]string{
		Text: "text", Keyword: "keyword", String: "string", Escape: "string_escape", Number: "number",
		Comment: "comment", Function: "function", Operator: "operator", Punctuation: "punctuation",
		Identifier: "identifier", Property: "property", Boolean: "boolean", Variable: "variable",
		Builtin: "builtin", Regex: "regex", Datetime: "datetime", TableHeader: "array_table_header",
		Namespace: "namespace", Parameter: "parameter", Constant: "constant",
		Selector: "selector", SelectorClass: "selector_class", SelectorID: "selector_id", SelectorPseudo: "selector_pseudo",
		Attribute: "attribute", CSSVariable: "css_variable", Unit: "unit", Null: "null",
		CharEscape: "escape", HardBreak: "hard_break", Heading: "heading", HeadingMarker: "heading_marker",
		Bold: "bold", Italic: "italic", Strike: "strike", Code: "code", LinkText: "link_text", Autolink: "autolink",
		URL: "url", URLLink: "url_link", URLTitle: "url_title", Entity: "entity", CodeBlock: "code_block",
		CodeFence: "code_fence", CodeLanguage: "code_language", RawCodeBlock: "raw_code_block",
		FrontMatterMarker: "front_matter_marker", RawFrontMatter: "raw_front_matter", BlockquoteMarker: "blockquote_marker",
		ListMarker: "list_marker", TaskMarker: "task_marker", ThematicBreak: "hr",
		Prompt: "prompt", PromptPrefix: "prompt_prefix", Output: "output", Template: "template", Decorator: "decorator",
		Type: "type", ClassName: "class_name", TagName: "tag_name", AttrName: "attr_name", Doctype: "doctype",
		boldOpen: "bold_open", boldClose: "bold_close", italicOpen: "italic_open", italicClose: "italic_close",
		strikeOpen: "strike_open", strikeClose: "strike_close", codeOpen: "code_open", codeClose: "code_close",
		linkTextOpen: "link_text_open", linkTextClose: "link_text_close", autolinkOpen: "autolink_open",
		autolinkClose: "autolink_close", rawShell: "raw_shell", rawScript: "raw_script", rawStyle: "raw_style",
	}[k]
}

type Style uint8

const (
	StyleBold Style = 1 << iota
	StyleItalic
	StyleStrike
	StyleCode
	StyleLinkText
	StyleAutolink
)

type Span struct {
	Kind       Kind
	Start, End int
	Style      Style
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
	tokens *[]Span
	last   Span
}

func (o *spans) emit(k Kind, start, end int, sealed bool) {
	if !sealed && o.last.Kind == k && o.last.End == start {
		o.last.End = end
		return
	}
	o.flush()
	o.last = Span{Kind: k, Start: start, End: end}
}

func (o *spans) flush() {
	if o.last.End > o.last.Start {
		*o.tokens = append(*o.tokens, o.last)
	}
}

func Tokens(src string, g *Grammar) iter.Seq[Span] {
	return func(yield func(Span) bool) {
		w := g.borrow()
		g.fill(w, src)
		at := 0
		for _, s := range w.spans {
			if s.Start > at && !yield(Span{Kind: Text, Start: at, End: s.Start}) || !yield(s) {
				at = len(src)
				break
			}
			at = s.End
		}
		if at < len(src) {
			yield(Span{Kind: Text, Start: at, End: len(src)})
		}
		g.giveBack(w)
	}
}

func (g *Grammar) borrow() *work {
	if w, ok := g.works.Get().(*work); ok {
		return w
	}
	return &work{}
}

func (g *Grammar) giveBack(w *work) {
	w.src = ""
	g.works.Put(w)
}

func (g *Grammar) fill(w *work, src string) {
	w.src, w.spans = src, w.spans[:0]
	out := spans{tokens: &w.spans}
	g.run(src, &out)
	out.flush()
	if g.reclassify != nil {
		g.reclassify(w)
	}
}

func (g *Grammar) run(src string, out *spans) {
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
				return
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
			if ru.kind != Text {
				out.emit(ru.kind, pos, end, false)
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
			out.emit(ru.kind, pos, pos+width, n > 0 || ru.boundary)
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
