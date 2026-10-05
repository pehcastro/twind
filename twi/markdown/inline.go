package markdown

import (
	"strings"
	"unicode"
	"unicode/utf8"

	konst "github.com/pehcastro/twind/internal/konst/markdown"
)

type piece struct {
	Inline
	delim       byte
	n, orig     int
	open, close bool
	active      bool
	prev, next  *piece
}

type inliner struct {
	head     piece
	tail     *piece
	brackets []*piece
	text     strings.Builder
}

func inlines(src string) []Inline {
	in := &inliner{}
	in.tail = &in.head
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '\\' && i+1 < len(src) && src[i+1] == '\n':
			in.push(&piece{Inline: Inline{Style: HardBreak}})
			i = skip(src, i+2, " \t")
		case c == '\\' && i+1 < len(src) && asciiPunct(src[i+1]):
			in.text.WriteByte(src[i+1])
			i += 2
		case c == '`':
			n := run(src, i)
			code, end := codeSpan(src, i, n)
			if end < 0 {
				in.text.WriteString(src[i : i+n])
				i += n
				continue
			}
			in.push(&piece{Inline: Inline{Style: CodeSpan, Text: code}})
			i = end
		case c == '*' || c == '_':
			n := run(src, i)
			before, after := ' ', ' '
			if i > 0 {
				before, _ = utf8.DecodeLastRuneInString(src[:i])
			}
			if i+n < len(src) {
				after, _ = utf8.DecodeRuneInString(src[i+n:])
			}
			left := !unicode.IsSpace(after) && (!punct(after) || unicode.IsSpace(before) || punct(before))
			right := !unicode.IsSpace(before) && (!punct(before) || unicode.IsSpace(after) || punct(after))
			open, closes := left, right
			if c == '_' {
				open, closes = left && (!right || punct(before)), right && (!left || punct(after))
			}
			in.push(&piece{delim: c, n: n, orig: n, open: open, close: closes})
			i += n
		case c == '[':
			opener := &piece{Inline: Inline{Text: "["}, active: true}
			in.push(opener)
			in.brackets = append(in.brackets, opener)
			i++
		case c == ']':
			i = in.closeBracket(src, i)
		case c == '\n':
			line := in.text.String()
			trimmed := strings.TrimRight(line, " ")
			in.text.Reset()
			in.text.WriteString(trimmed)
			style := SoftBreak
			if len(line)-len(trimmed) >= konst.HardBreakSpaces {
				style = HardBreak
			}
			in.push(&piece{Inline: Inline{Style: style}})
			i = skip(src, i+1, " \t")
		default:
			in.text.WriteByte(c)
			i++
		}
	}
	in.flush()
	in.emphasis(&in.head)
	return collect(in.head.next, nil)
}

func punct(r rune) bool { return unicode.IsPunct(r) || unicode.IsSymbol(r) }

func run(src string, i int) int {
	return len(src) - i - len(strings.TrimLeft(src[i:], src[i:i+1]))
}

func skip(src string, i int, blanks string) int {
	return len(src) - len(strings.TrimLeft(src[i:], blanks))
}

func codeSpan(src string, i, n int) (string, int) {
	for j := i + n; j < len(src); {
		if src[j] != '`' {
			j++
			continue
		}
		m := run(src, j)
		if m != n {
			j += m
			continue
		}
		code := strings.ReplaceAll(src[i+n:j], "\n", " ")
		if len(code) >= 2 && code[0] == ' ' && code[len(code)-1] == ' ' && strings.Trim(code, " ") != "" {
			code = code[1 : len(code)-1]
		}
		return code, j + n
	}
	return "", -1
}

func (in *inliner) flush() {
	if in.text.Len() > 0 {
		p := &piece{Inline: Inline{Text: in.text.String()}, prev: in.tail}
		in.text.Reset()
		in.tail.next, in.tail = p, p
	}
}

func (in *inliner) push(p *piece) {
	in.flush()
	p.prev = in.tail
	in.tail.next, in.tail = p, p
}

func (in *inliner) closeBracket(src string, i int) int {
	in.flush()
	if len(in.brackets) == 0 {
		in.text.WriteByte(']')
		return i + 1
	}
	opener := in.brackets[len(in.brackets)-1]
	in.brackets = in.brackets[:len(in.brackets)-1]
	target, end := destination(src, i+1)
	if !opener.active || end < 0 {
		in.text.WriteByte(']')
		return i + 1
	}
	in.emphasis(opener)
	link := &piece{Inline: Inline{Style: Link, Target: target, Children: collect(opener.next, nil)}, prev: opener.prev}
	opener.prev.next, in.tail = link, link
	for _, b := range in.brackets {
		b.active = false
	}
	return end
}

func destination(src string, i int) (string, int) {
	if i >= len(src) || src[i] != '(' {
		return "", -1
	}
	j := skip(src, i+1, " \t\n")
	var dest strings.Builder
	if j < len(src) && src[j] == '<' {
		for j++; j < len(src) && src[j] != '>'; j++ {
			if src[j] == '\n' || src[j] == '<' {
				return "", -1
			}
			if src[j] == '\\' && j+1 < len(src) && asciiPunct(src[j+1]) {
				j++
			}
			dest.WriteByte(src[j])
		}
		if j >= len(src) {
			return "", -1
		}
		j++
	} else {
		depth := 0
		for ; j < len(src) && src[j] > ' ' && (src[j] != ')' || depth > 0); j++ {
			switch {
			case src[j] == '\\' && j+1 < len(src) && asciiPunct(src[j+1]):
				j++
			case src[j] == '(':
				depth++
			case src[j] == ')':
				depth--
			}
			dest.WriteByte(src[j])
		}
		if depth != 0 {
			return "", -1
		}
	}
	k := skip(src, j, " \t\n")
	if k > j && k < len(src) && strings.IndexByte("\"'(", src[k]) >= 0 {
		closer := map[byte]byte{'"': '"', '\'': '\'', '(': ')'}[src[k]]
		for k++; k < len(src) && src[k] != closer; k++ {
			if src[k] == '\\' {
				k++
			}
		}
		if k >= len(src) {
			return "", -1
		}
		k = skip(src, k+1, " \t\n")
	}
	if k >= len(src) || src[k] != ')' {
		return "", -1
	}
	return dest.String(), k + 1
}

func (in *inliner) emphasis(bottom *piece) {
	type closer struct {
		delim byte
		rest  int
		open  bool
	}
	floor := map[closer]*piece{}
	for c := bottom.next; c != nil; {
		if c.delim == 0 || !c.close {
			c = c.next
			continue
		}
		key := closer{c.delim, c.orig % 3, c.open}
		stop := floor[key]
		if stop == nil {
			stop = bottom
		}
		var o *piece
		for x := c.prev; x != stop && x != bottom; x = x.prev {
			ruleOfThree := (x.close || c.open) && (x.orig+c.orig)%3 == 0 && (x.orig%3 != 0 || c.orig%3 != 0)
			if x.delim == c.delim && x.open && !ruleOfThree {
				o = x
				break
			}
		}
		if o == nil {
			floor[key] = c.prev
			if !c.open {
				c.close = false
			}
			c = c.next
			continue
		}
		use := 1
		if o.n >= 2 && c.n >= 2 {
			use = 2
		}
		o.n -= use
		c.n -= use
		e := &piece{Inline: Inline{Style: [...]Style{1: Emphasis, 2: Strong}[use], Children: collect(o.next, c)}, prev: o, next: c}
		o.next, c.prev = e, e
		if o.n == 0 {
			o.prev.next, e.prev = e, o.prev
		}
		if c.n == 0 {
			e.next = c.next
			if c.next != nil {
				c.next.prev = e
			} else {
				in.tail = e
			}
			c = c.next
		}
	}
}

func collect(from, to *piece) []Inline {
	var out []Inline
	for p := from; p != to; p = p.next {
		in := p.Inline
		if p.delim != 0 {
			in.Text = strings.Repeat(string(p.delim), p.n)
		}
		switch {
		case in.Style == Text && in.Text == "":
		case in.Style == Text && len(out) > 0 && out[len(out)-1].Style == Text:
			out[len(out)-1].Text += in.Text
		default:
			out = append(out, in)
		}
	}
	return out
}
