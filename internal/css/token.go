package css

import (
	"strings"
	"unicode/utf8"
)

type TokenKind uint8

const (
	TokenInvalid TokenKind = iota
	TokenIdent
	TokenFunction
	TokenAtKeyword
	TokenHash
	TokenString
	TokenURL
	TokenDelim
	TokenNumber
	TokenPercentage
	TokenDimension
	TokenWhitespace
	TokenColon
	TokenSemicolon
	TokenComma
	TokenOpenSquare
	TokenCloseSquare
	TokenOpenParen
	TokenCloseParen
	TokenOpenCurly
	TokenCloseCurly
)

type Token struct {
	Kind TokenKind
	Text string
}

type lexer struct {
	src string
	i   int
}

func tokenize(src string) ([]Token, []int, *Error) {
	lx := lexer{src: src}
	var toks []Token
	var offs []int
	for lx.i < len(src) {
		start := lx.i
		kind, err := lx.next()
		if err != nil {
			return nil, nil, err
		}
		if kind == TokenInvalid {
			continue
		}
		toks = append(toks, Token{Kind: kind, Text: src[start:lx.i]})
		offs = append(offs, start)
	}
	return toks, offs, nil
}

func (lx *lexer) at(k int) byte {
	if lx.i+k < len(lx.src) {
		return lx.src[lx.i+k]
	}
	return 0
}

func (lx *lexer) next() (TokenKind, *Error) {
	start := lx.i
	c := lx.src[lx.i]
	switch {
	case isSpace(c):
		for lx.i < len(lx.src) && isSpace(lx.src[lx.i]) {
			lx.i++
		}
		return TokenWhitespace, nil
	case c == '/' && lx.at(1) == '*':
		end := strings.Index(lx.src[lx.i+2:], "*/")
		if end < 0 {
			return 0, &Error{Kind: ErrUnclosedComment, off: start}
		}
		lx.i += end + 4
		return TokenInvalid, nil
	case c == '"' || c == '\'':
		return TokenString, lx.str(c)
	case c == '#' && (isName(lx.at(1)) || isEscape(lx.at(1), lx.at(2))):
		lx.i++
		lx.name()
		return TokenHash, nil
	case c == '@' && startsIdent(lx.at(1), lx.at(2), lx.at(3)):
		lx.i++
		lx.name()
		return TokenAtKeyword, nil
	case startsNumber(c, lx.at(1), lx.at(2)):
		return lx.numeric(), nil
	case startsIdent(c, lx.at(1), lx.at(2)):
		return lx.identLike()
	}
	lx.i++
	switch c {
	case ':':
		return TokenColon, nil
	case ';':
		return TokenSemicolon, nil
	case ',':
		return TokenComma, nil
	case '[':
		return TokenOpenSquare, nil
	case ']':
		return TokenCloseSquare, nil
	case '(':
		return TokenOpenParen, nil
	case ')':
		return TokenCloseParen, nil
	case '{':
		return TokenOpenCurly, nil
	case '}':
		return TokenCloseCurly, nil
	}
	_, size := utf8.DecodeRuneInString(lx.src[start:])
	lx.i = start + size
	return TokenDelim, nil
}

func (lx *lexer) str(quote byte) *Error {
	start := lx.i
	lx.i++
	for lx.i < len(lx.src) {
		switch lx.src[lx.i] {
		case quote:
			lx.i++
			return nil
		case '\n', '\r', '\f':
			return &Error{Kind: ErrUnclosedString, off: start}
		case '\\':
			lx.i += 2
		default:
			lx.i++
		}
	}
	return &Error{Kind: ErrUnclosedString, off: start}
}

func (lx *lexer) numeric() TokenKind {
	if lx.at(0) == '+' || lx.at(0) == '-' {
		lx.i++
	}
	lx.digits()
	if lx.at(0) == '.' && isDigit(lx.at(1)) {
		lx.i++
		lx.digits()
	}
	if e := lx.at(0); e == 'e' || e == 'E' {
		if isDigit(lx.at(1)) {
			lx.i++
			lx.digits()
		} else if (lx.at(1) == '+' || lx.at(1) == '-') && isDigit(lx.at(2)) {
			lx.i += 2
			lx.digits()
		}
	}
	switch {
	case startsIdent(lx.at(0), lx.at(1), lx.at(2)):
		lx.name()
		return TokenDimension
	case lx.at(0) == '%':
		lx.i++
		return TokenPercentage
	}
	return TokenNumber
}

func (lx *lexer) digits() {
	for isDigit(lx.at(0)) {
		lx.i++
	}
}

func (lx *lexer) identLike() (TokenKind, *Error) {
	start := lx.i
	lx.name()
	if lx.at(0) != '(' {
		return TokenIdent, nil
	}
	lx.i++
	if !strings.EqualFold(lx.src[start:lx.i], "url(") {
		return TokenFunction, nil
	}
	j := lx.i
	for j < len(lx.src) && isSpace(lx.src[j]) {
		j++
	}
	if j < len(lx.src) && (lx.src[j] == '"' || lx.src[j] == '\'') {
		return TokenFunction, nil
	}
	for lx.i < len(lx.src) {
		switch c := lx.src[lx.i]; {
		case c == ')':
			lx.i++
			return TokenURL, nil
		case c == '\\' && isEscape(c, lx.at(1)):
			lx.i += 2
		case c == '"' || c == '\'' || c == '(':
			return 0, &Error{Kind: ErrBadURL, off: start}
		default:
			lx.i++
		}
	}
	return 0, &Error{Kind: ErrBadURL, off: start}
}

func (lx *lexer) name() {
	for lx.i < len(lx.src) {
		c := lx.src[lx.i]
		switch {
		case isName(c):
			lx.i++
		case isEscape(c, lx.at(1)):
			lx.escape()
		default:
			return
		}
	}
}

func (lx *lexer) escape() {
	lx.i++
	if !isHex(lx.at(0)) {
		_, size := utf8.DecodeRuneInString(lx.src[lx.i:])
		lx.i += size
		return
	}
	for n := 0; n < 6 && isHex(lx.at(0)); n++ {
		lx.i++
	}
	if isSpace(lx.at(0)) {
		lx.i++
	}
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }

func isHex(c byte) bool {
	return isDigit(c) || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

func isNameStart(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c == '_' || c >= utf8.RuneSelf
}

func isName(c byte) bool { return isNameStart(c) || isDigit(c) || c == '-' }

func isEscape(c, next byte) bool {
	return c == '\\' && next != '\n' && next != '\r' && next != '\f' && next != 0
}

func startsIdent(a, b, c byte) bool {
	if a == '-' {
		return isNameStart(b) || b == '-' || isEscape(b, c)
	}
	return isNameStart(a) || isEscape(a, b)
}

func startsNumber(a, b, c byte) bool {
	if a == '+' || a == '-' {
		return isDigit(b) || b == '.' && isDigit(c)
	}
	return isDigit(a) || a == '.' && isDigit(b)
}
