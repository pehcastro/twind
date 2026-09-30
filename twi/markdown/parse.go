package markdown

import (
	"slices"
	"strconv"
	"strings"
	"unicode"

	konst "github.com/twind-dev/twind/internal/konst/markdown"
	"github.com/twind-dev/twind/twi/text"
)

const item = Tag + 1

type node struct {
	kind      Kind
	parent    *node
	children  []*node
	open      bool
	lastBlank bool
	line      int
	lines     []string
	level     int
	ordered   bool
	marker    byte
	start     int
	tight     bool
	offset    int
	fence     byte
	fenceLen  int
	indent    int
	info      string
	align     []Align
	tag       Block
}

type parser struct {
	file       string
	components map[string]Component
	root       node
	line       string
	offset     int
	number     int
}

func Parse(file, src string, components map[string]Component) (Page, error) {
	p := parser{file: file, components: components, root: node{open: true}}
	lines := strings.Split(text.Sanitize(src, text.RemoveBidi), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for i, line := range lines {
		p.number = i + 1
		if err := p.add(detab(line)); err != nil {
			return Page{}, err
		}
	}
	p.closeLast(&p.root)
	page := Page{File: file, Blocks: blocks(p.root.children)}
	page.Anchors = anchors(page.Blocks, map[string]int{}, nil)
	return page, nil
}

func detab(line string) string {
	end := strings.IndexFunc(line, func(r rune) bool { return r != ' ' && r != '\t' && r != '>' })
	if end < 0 {
		end = len(line)
	}
	if !strings.Contains(line[:end], "\t") {
		return line
	}
	var b strings.Builder
	for _, c := range line[:end] {
		if c == '\t' {
			b.WriteString(strings.Repeat(" ", konst.TabStop-b.Len()%konst.TabStop))
			continue
		}
		b.WriteRune(c)
	}
	return b.String() + line[end:]
}

func (p *parser) indent() int {
	rest := p.line[p.offset:]
	return len(rest) - len(strings.TrimLeft(rest, " "))
}

func (p *parser) rest() string { return p.line[p.offset:] }

func (p *parser) blank() bool { return strings.Trim(p.rest(), " \t") == "" }

func openChild(n *node) *node {
	if len(n.children) == 0 || !n.children[len(n.children)-1].open {
		return nil
	}
	return n.children[len(n.children)-1]
}

func (p *parser) add(line string) error {
	p.line, p.offset = line, 0
	c := &p.root
	for next := openChild(c); next != nil; next = openChild(c) {
		ok, done := p.continues(next)
		if done {
			return nil
		}
		if !ok {
			break
		}
		c = next
	}
	matched := c
	if c.kind == Code {
		c.lines = append(c.lines, p.rest()[min(c.indent, p.indent()):])
		return nil
	}
	c, err := p.start(c)
	if err != nil {
		return err
	}
	blank := p.blank()
	if blank && len(c.children) > 0 {
		c.children[len(c.children)-1].lastBlank = true
	}
	openedEmpty := c.kind == item && len(c.children) == 0 && c.line == p.number
	c.lastBlank = blank && c.kind != Quote && c.kind != Code && !openedEmpty
	for t := c.parent; t != nil; t = t.parent {
		t.lastBlank = false
	}
	tip := c
	for next := openChild(tip); next != nil; next = openChild(tip) {
		tip = next
	}
	if c == matched && tip != c && !blank && tip.kind == Paragraph {
		tip.lines = append(tip.lines, strings.TrimLeft(p.rest(), " "))
		return nil
	}
	if c == matched {
		p.closeLast(c)
	}
	content := strings.TrimLeft(p.rest(), " ")
	switch {
	case blank || c.kind == Code || c.kind == Heading || c.kind == Break || c.kind == Tag:
	case c.kind == Paragraph && len(c.lines) == 1 && strings.Contains(content, "|"):
		if align, ok := delimiterRow(content, len(cells(c.lines[0]))); ok {
			c.kind, c.align = Table, align
			return nil
		}
		c.lines = append(c.lines, content)
	case c.kind == Paragraph || c.kind == Table:
		c.lines = append(c.lines, content)
	default:
		p.open(c, &node{kind: Paragraph, lines: []string{content}})
	}
	return nil
}

func (p *parser) start(c *node) (*node, error) {
	for {
		indent := p.indent()
		rest := p.line[p.offset+indent:]
		if indent >= konst.CodeIndent || rest == "" {
			return c, nil
		}
		if rest[0] == '>' {
			p.offset += indent + 1
			if strings.HasPrefix(p.rest(), " ") {
				p.offset++
			}
			c = p.open(c, &node{kind: Quote})
			continue
		}
		if level, content, ok := atx(rest); ok {
			return p.open(c, &node{kind: Heading, level: level, lines: []string{content}}), nil
		}
		if n := fenceOpen(rest); n > 0 {
			return p.open(c, &node{kind: Code, fence: rest[0], fenceLen: n, indent: indent, info: language(rest[n:])}), nil
		}
		if thematicBreak(rest) {
			return p.open(c, &node{kind: Break}), nil
		}
		if next, ok := p.listItem(c, rest, indent); ok {
			c = next
			continue
		}
		if rest[0] != '<' || len(rest) < 2 || rest[1] < 'A' || rest[1] > 'Z' {
			return c, nil
		}
		tag, err := p.tag(rest)
		if err != nil {
			return nil, err
		}
		return p.open(c, &node{kind: Tag, tag: tag}), nil
	}
}

func (p *parser) continues(n *node) (ok, done bool) {
	indent := p.indent()
	switch n.kind {
	case Quote:
		if indent < konst.CodeIndent && strings.HasPrefix(p.line[p.offset+indent:], ">") {
			p.offset += indent + 1
			if strings.HasPrefix(p.rest(), " ") {
				p.offset++
			}
			return true, false
		}
		return false, false
	case item:
		switch {
		case indent >= n.offset:
			p.offset += n.offset
		case p.blank() && len(n.children) > 0:
			p.offset = len(p.line)
		default:
			return false, false
		}
		return true, false
	case List:
		return true, false
	case Code:
		rest := p.line[p.offset+indent:]
		closing := strings.TrimLeft(rest, string(n.fence))
		if indent < konst.CodeIndent && len(rest)-len(closing) >= n.fenceLen && strings.Trim(closing, " \t") == "" {
			p.close(n)
			return false, true
		}
		return true, false
	case Paragraph, Table:
		return !p.blank(), false
	case Heading, Break, Tag:
		return false, false
	}
	panic("markdown: unknown block kind " + strconv.Itoa(int(n.kind)))
}

func accepts(parent, child Kind) bool {
	switch parent {
	case 0, Quote, item:
		return child != item
	case List:
		return child == item
	}
	return false
}

func (p *parser) open(parent, child *node) *node {
	for !accepts(parent.kind, child.kind) {
		p.close(parent)
		parent = parent.parent
	}
	p.closeLast(parent)
	child.parent, child.open, child.line = parent, true, p.number
	parent.children = append(parent.children, child)
	return child
}

func (p *parser) closeLast(n *node) {
	if last := openChild(n); last != nil {
		p.close(last)
	}
}

func (p *parser) close(n *node) {
	p.closeLast(n)
	n.open = false
	if n.kind == List {
		n.tight = tight(n)
	}
}

func tight(list *node) bool {
	for i, it := range list.children {
		last := i == len(list.children)-1
		if it.lastBlank && !last {
			return false
		}
		for j, sub := range it.children {
			if !last || j < len(it.children)-1 {
				for n := sub; ; n = n.children[len(n.children)-1] {
					if n.lastBlank {
						return false
					}
					if n.kind != List && n.kind != item || len(n.children) == 0 {
						break
					}
				}
			}
		}
	}
	return true
}

func atx(rest string) (int, string, bool) {
	content := strings.TrimLeft(rest, "#")
	level := len(rest) - len(content)
	if level == 0 || level > konst.MaxHeadingLevel || content != "" && content[0] != ' ' && content[0] != '\t' {
		return 0, "", false
	}
	content = strings.Trim(content, " \t")
	switch unclosed := strings.TrimRight(content, "#"); {
	case unclosed == "":
		content = ""
	case strings.HasSuffix(unclosed, " ") || strings.HasSuffix(unclosed, "\t"):
		content = strings.TrimRight(unclosed, " \t")
	}
	return level, content, true
}

func fenceOpen(rest string) int {
	if rest[0] != '`' && rest[0] != '~' {
		return 0
	}
	n := len(rest) - len(strings.TrimLeft(rest, rest[:1]))
	if n < konst.MinFence || rest[0] == '`' && strings.Contains(rest[n:], "`") {
		return 0
	}
	return n
}

func language(info string) string {
	var b strings.Builder
	for i := 0; i < len(info); i++ {
		if info[i] == '\\' && i+1 < len(info) && asciiPunct(info[i+1]) {
			i++
		}
		b.WriteByte(info[i])
	}
	if words := strings.Fields(b.String()); len(words) > 0 {
		return words[0]
	}
	return ""
}

func asciiPunct(c byte) bool { return strings.IndexByte("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", c) >= 0 }

func thematicBreak(rest string) bool {
	mark := rest[0]
	if mark != '*' && mark != '-' && mark != '_' {
		return false
	}
	marks := strings.Count(rest, string(mark))
	return marks >= konst.MinBreakMarks && marks+strings.Count(rest, " ")+strings.Count(rest, "\t") == len(rest)
}

func (p *parser) listItem(c *node, rest string, indent int) (*node, bool) {
	var start, width int
	ordered := false
	switch rest[0] {
	case '-', '+', '*':
		width = 1
	default:
		digits := len(rest) - len(strings.TrimLeft(rest, "0123456789"))
		if digits == 0 || digits > konst.MaxOrderedDigits || digits == len(rest) || rest[digits] != '.' && rest[digits] != ')' {
			return nil, false
		}
		start, _ = strconv.Atoi(rest[:digits])
		ordered, width = true, digits+1
	}
	marker, after := rest[width-1], rest[width:]
	if after != "" && after[0] != ' ' && after[0] != '\t' {
		return nil, false
	}
	empty := strings.Trim(after, " \t") == ""
	if c.kind == Paragraph && (empty || ordered && start != 1) {
		return nil, false
	}
	pad := len(after) - len(strings.TrimLeft(after, " \t"))
	if empty || pad > konst.MaxItemPadding {
		pad = 1
	}
	p.offset = min(p.offset+indent+width+pad, len(p.line))
	if c.kind != List || c.ordered != ordered || c.marker != marker {
		c = p.open(c, &node{kind: List, ordered: ordered, marker: marker, start: start})
	}
	return p.open(c, &node{kind: item, offset: indent + width + pad}), true
}

func (p *parser) tag(rest string) (Block, error) {
	s := strings.TrimRight(rest, " \t")
	word := func(r rune) bool { return r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) }
	name := s[1 : len(s)-len(strings.TrimLeftFunc(s[1:], word))]
	fail := func(problem Problem, what string) (Block, error) {
		return Block{}, &Error{File: p.file, Line: p.number, Problem: problem, Name: what}
	}
	body, closed := strings.CutSuffix(s[1+len(name):], "/>")
	if !closed {
		return fail(MalformedTag, name)
	}
	attrs := map[string]string{}
	var keys []string
	for {
		trimmed := strings.TrimLeft(body, " \t")
		if trimmed == "" {
			break
		}
		key, value, ok := strings.Cut(trimmed, "=\"")
		end := strings.IndexByte(value, '"')
		if len(trimmed) == len(body) || !ok || end < 0 || key == "" || strings.ContainsFunc(key, func(r rune) bool { return !word(r) && r != '-' }) {
			return fail(MalformedTag, name)
		}
		if _, dup := attrs[key]; dup {
			return fail(DuplicateAttribute, key)
		}
		attrs[key], body = value[:end], value[end+1:]
		keys = append(keys, key)
	}
	component, known := p.components[name]
	if !known {
		return fail(UnknownTag, name)
	}
	for _, key := range keys {
		if !slices.Contains(component.Attrs, key) {
			return fail(UnknownAttribute, key)
		}
	}
	return Block{Kind: Tag, Name: name, Attrs: attrs, build: component.Build}, nil
}

func cells(row string) []string {
	row = strings.TrimPrefix(strings.Trim(row, " \t"), "|")
	if strings.HasSuffix(row, "|") && !strings.HasSuffix(row, "\\|") {
		row = row[:len(row)-1]
	}
	var out []string
	start := 0
	for i := 0; i < len(row); i++ {
		switch row[i] {
		case '\\':
			i++
		case '|':
			out = append(out, row[start:i])
			start = i + 1
		}
	}
	out = append(out, row[start:])
	for i, c := range out {
		out[i] = strings.ReplaceAll(strings.Trim(c, " \t"), "\\|", "|")
	}
	return out
}

func delimiterRow(line string, columns int) ([]Align, bool) {
	row := cells(line)
	if len(row) != columns {
		return nil, false
	}
	align := make([]Align, columns)
	for i, c := range row {
		left, right := strings.HasPrefix(c, ":"), strings.HasSuffix(c, ":")
		if dashes := strings.Trim(c, ":"); dashes == "" || strings.Trim(dashes, "-") != "" || len(c)-len(dashes) > 2 {
			return nil, false
		}
		align[i] = map[[2]bool]Align{{true, false}: AlignLeft, {true, true}: AlignCenter, {false, true}: AlignRight}[[2]bool{left, right}]
	}
	return align, true
}

func blocks(nodes []*node) []Block {
	out := make([]Block, len(nodes))
	for i, n := range nodes {
		b := Block{Kind: n.kind, Line: n.line}
		switch n.kind {
		case Paragraph:
			b.Inlines = inlines(strings.TrimRight(strings.Join(n.lines, "\n"), " \t"))
		case Heading:
			b.Level, b.Inlines = n.level, inlines(n.lines[0])
		case Code:
			b.Language = n.info
			if len(n.lines) > 0 {
				b.Code = strings.Join(n.lines, "\n") + "\n"
			}
		case Quote:
			b.Children = blocks(n.children)
		case List:
			b.Ordered, b.Start, b.Tight = n.ordered, n.start, n.tight
			for _, it := range n.children {
				b.Items = append(b.Items, blocks(it.children))
			}
		case Table:
			b.Align = n.align
			for _, line := range n.lines {
				row, given := make([][]Inline, len(n.align)), cells(line)
				for j, c := range given[:min(len(given), len(row))] {
					row[j] = inlines(c)
				}
				b.Rows = append(b.Rows, row)
			}
		case Break:
		case Tag:
			b = n.tag
			b.Line = n.line
		default:
			panic("markdown: unknown block kind " + strconv.Itoa(int(n.kind)))
		}
		out[i] = b
	}
	return out
}

func anchors(blocks []Block, seen map[string]int, out []Anchor) []Anchor {
	for i := range blocks {
		b := &blocks[i]
		switch b.Kind {
		case Heading:
			title := plain(b.Inlines)
			var slug strings.Builder
			for _, r := range strings.ToLower(title) {
				switch {
				case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_':
					slug.WriteRune(r)
				case r == ' ':
					slug.WriteByte('-')
				}
			}
			b.ID = slug.String()
			if n := seen[b.ID]; n > 0 {
				seen[b.ID]++
				b.ID += "-" + strconv.Itoa(n)
			} else {
				seen[b.ID] = 1
			}
			out = append(out, Anchor{Level: b.Level, Text: title, ID: b.ID})
		case Quote:
			out = anchors(b.Children, seen, out)
		case List:
			for _, it := range b.Items {
				out = anchors(it, seen, out)
			}
		}
	}
	return out
}

func plain(inlines []Inline) string {
	var b strings.Builder
	for _, in := range inlines {
		switch in.Style {
		case Text, CodeSpan:
			b.WriteString(in.Text)
		case SoftBreak, HardBreak:
			b.WriteByte(' ')
		case Emphasis, Strong, Link:
			b.WriteString(plain(in.Children))
		}
	}
	return b.String()
}
