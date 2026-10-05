package css

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	konst "github.com/pehcastro/twind/internal/konst/css"
)

type Pos struct {
	Line int
	Col  int
}

type Node interface{ node() }

type AtRule struct {
	Pos      Pos
	Name     string
	Prelude  []Token
	HasBlock bool
	Block    []Node
}

type Rule struct {
	Pos      Pos
	Selector string
	Block    []Node
}

type Declaration struct {
	Pos       Pos
	Property  string
	Value     []Token
	Important bool
}

func (AtRule) node()      {}
func (Rule) node()        {}
func (Declaration) node() {}

type ErrorKind uint8

const (
	ErrInvalid ErrorKind = iota
	ErrUnclosedComment
	ErrUnclosedString
	ErrBadURL
	ErrUnclosedBlock
	ErrStrayCloseBrace
	ErrEOFInPrelude
	ErrMissingBlock
	ErrEmptyValue
	ErrTooDeep
)

type Error struct {
	Kind ErrorKind
	Pos  Pos
	off  int
}

func (e *Error) Error() string {
	var msg string
	switch e.Kind {
	case ErrUnclosedComment:
		msg = "comment not closed"
	case ErrUnclosedString:
		msg = "string not closed"
	case ErrBadURL:
		msg = "malformed url()"
	case ErrUnclosedBlock:
		msg = "block not closed"
	case ErrStrayCloseBrace:
		msg = "} with no open block"
	case ErrEOFInPrelude:
		msg = "end of input inside at-rule prelude"
	case ErrMissingBlock:
		msg = "rule has no block"
	case ErrEmptyValue:
		msg = "declaration has no value"
	case ErrTooDeep:
		msg = "blocks nested too deep"
	default:
		panic(fmt.Sprintf("css: unknown error kind %d", e.Kind))
	}
	return fmt.Sprintf("css:%d:%d: %s", e.Pos.Line, e.Pos.Col, msg)
}

type parser struct {
	src   string
	toks  []Token
	offs  []int
	lines []int
	i     int
}

func Parse(src string) ([]Node, error) {
	p := parser{src: src, lines: []int{0}}
	for i := range len(src) {
		if src[i] == '\n' {
			p.lines = append(p.lines, i+1)
		}
	}
	toks, offs, lexErr := tokenize(src)
	if lexErr != nil {
		return nil, p.fail(lexErr.Kind, lexErr.off)
	}
	p.toks, p.offs = toks, offs
	return p.list(0, -1)
}

func (p *parser) pos(off int) Pos {
	line := sort.Search(len(p.lines), func(i int) bool { return p.lines[i] > off })
	return Pos{Line: line, Col: utf8.RuneCountInString(p.src[p.lines[line-1]:off]) + 1}
}

func (p *parser) fail(kind ErrorKind, off int) *Error {
	return &Error{Kind: kind, Pos: p.pos(off), off: off}
}

func (p *parser) kind() TokenKind {
	if p.i < len(p.toks) {
		return p.toks[p.i].Kind
	}
	return TokenInvalid
}

func (p *parser) off() int {
	if p.i < len(p.offs) {
		return p.offs[p.i]
	}
	return len(p.src)
}

func (p *parser) list(depth, open int) ([]Node, error) {
	var nodes []Node
	for {
		switch p.kind() {
		case TokenWhitespace, TokenSemicolon:
			p.i++
			continue
		case TokenInvalid:
			if open >= 0 {
				return nil, p.fail(ErrUnclosedBlock, open)
			}
			return nodes, nil
		case TokenCloseCurly:
			if open < 0 {
				return nil, p.fail(ErrStrayCloseBrace, p.off())
			}
			p.i++
			return nodes, nil
		}
		var n Node
		var err error
		switch {
		case p.kind() == TokenAtKeyword:
			n, err = p.atRule(depth)
		case open >= 0 && p.declarationAhead():
			n, err = p.declaration()
		default:
			n, err = p.rule(depth)
		}
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
}

func (p *parser) block(depth int) ([]Node, error) {
	open := p.off()
	if depth == konst.MaxBlockDepth {
		return nil, p.fail(ErrTooDeep, open)
	}
	p.i++
	return p.list(depth+1, open)
}

func (p *parser) atRule(depth int) (Node, error) {
	start := p.off()
	at := AtRule{Pos: p.pos(start), Name: p.toks[p.i].Text[1:]}
	p.i++
	at.Prelude = trim(p.values())
	if p.kind() == TokenInvalid {
		return nil, p.fail(ErrEOFInPrelude, start)
	}
	if p.kind() != TokenOpenCurly {
		return at, nil
	}
	at.HasBlock = true
	block, err := p.block(depth)
	at.Block = block
	return at, err
}

func (p *parser) rule(depth int) (Node, error) {
	start := p.off()
	selector := trim(p.values())
	if p.kind() != TokenOpenCurly {
		return nil, p.fail(ErrMissingBlock, start)
	}
	block, err := p.block(depth)
	return Rule{Pos: p.pos(start), Selector: text(selector), Block: block}, err
}

func (p *parser) declarationAhead() bool {
	j := p.i + 1
	for j < len(p.toks) && p.toks[j].Kind == TokenWhitespace {
		j++
	}
	if p.kind() != TokenIdent || j == len(p.toks) || p.toks[j].Kind != TokenColon {
		return false
	}
	saved := p.i
	p.values()
	curly := p.kind() == TokenOpenCurly
	p.i = saved
	return !curly
}

func (p *parser) declaration() (Node, error) {
	start := p.off()
	decl := Declaration{Pos: p.pos(start), Property: p.toks[p.i].Text}
	for p.kind() != TokenColon {
		p.i++
	}
	p.i++
	value := trim(p.values())
	if n := len(value); n >= 2 && value[n-1].Kind == TokenIdent && strings.EqualFold(value[n-1].Text, "important") {
		rest := trim(value[:n-1])
		if len(rest) > 0 && rest[len(rest)-1] == (Token{Kind: TokenDelim, Text: "!"}) {
			value, decl.Important = trim(rest[:len(rest)-1]), true
		}
	}
	if len(value) == 0 && !strings.HasPrefix(decl.Property, "--") {
		return nil, p.fail(ErrEmptyValue, start)
	}
	decl.Value = value
	return decl, nil
}

func (p *parser) values() []Token {
	start := p.i
	var closers []TokenKind
	for ; p.i < len(p.toks); p.i++ {
		k := p.toks[p.i].Kind
		if len(closers) == 0 && (k == TokenSemicolon || k == TokenOpenCurly || k == TokenCloseCurly) {
			break
		}
		switch k {
		case TokenOpenParen, TokenFunction:
			closers = append(closers, TokenCloseParen)
		case TokenOpenSquare:
			closers = append(closers, TokenCloseSquare)
		case TokenOpenCurly:
			closers = append(closers, TokenCloseCurly)
		default:
			if len(closers) > 0 && k == closers[len(closers)-1] {
				closers = closers[:len(closers)-1]
			}
		}
	}
	return slices.Clip(p.toks[start:p.i])
}

func text(toks []Token) string {
	var b strings.Builder
	for _, t := range toks {
		b.WriteString(t.Text)
	}
	return b.String()
}

func trim(toks []Token) []Token {
	for len(toks) > 0 && toks[0].Kind == TokenWhitespace {
		toks = toks[1:]
	}
	for len(toks) > 0 && toks[len(toks)-1].Kind == TokenWhitespace {
		toks = toks[:len(toks)-1]
	}
	return toks
}
