package text

import (
	"fmt"
	"strings"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/text"
)

type Bidi uint8

const (
	RemoveBidi Bidi = iota
	ShowBidi
)

const (
	bel             = 0x07
	esc             = 0x1b
	del             = 0x7f
	c1First         = 0x80
	c1Last          = 0x9f
	c1FromEscOffset = 0x40
	dcs             = 0x90
	sos             = 0x98
	csi             = 0x9b
	st              = 0x9c
	osc             = 0x9d
	pm              = 0x9e
	apc             = 0x9f
	utf8C1Lead      = 0xc2
)

func Sanitize(s string, bidi Bidi) string {
	var out strings.Builder
	for i := 0; i < len(s); {
		if n := controlLen(s[i:]); n > 0 {
			i += n
			continue
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		override := (r >= '\U0000202A' && r <= '\U0000202E') || (r >= '\U00002066' && r <= '\U00002069')
		switch {
		case r == utf8.RuneError && n == 1:
			out.WriteRune(utf8.RuneError)
		case override && bidi == ShowBidi:
			fmt.Fprintf(&out, konst.BidiVisible, r)
		case !override:
			out.WriteString(s[i : i+n])
		}
		i += n
	}
	return out.String()
}

func controlLen(s string) int {
	var c1 byte
	n := 0
	switch b := s[0]; {
	case b == esc && len(s) > 1 && s[1] >= '@' && s[1] <= '_':
		c1, n = s[1]+c1FromEscOffset, 2
	case b == esc:
		n = 1
		for n < len(s) && s[n] >= ' ' && s[n] <= '/' {
			n++
		}
		if n < len(s) && s[n] >= '0' && s[n] <= '~' {
			n++
		}
		return n
	case b >= c1First && b <= c1Last:
		c1, n = b, 1
	case b == utf8C1Lead && len(s) > 1 && s[1] >= c1First && s[1] <= c1Last:
		c1, n = s[1], 2
	case (b < ' ' && b != '\t' && b != '\n') || b == del:
		return 1
	default:
		return 0
	}
	switch c1 {
	case csi:
		for n < len(s) && s[n] >= ' ' && s[n] <= '?' {
			n++
		}
		if n < len(s) && s[n] >= '@' && s[n] <= '~' {
			n++
		}
	case osc, dcs, sos, pm, apc:
		for ; n < len(s); n++ {
			switch {
			case s[n] == bel || s[n] == st:
				return n + 1
			case strings.HasPrefix(s[n:], "\x1b\\") || strings.HasPrefix(s[n:], "\u009c"):
				return n + 2
			case s[n] == esc:
				return n
			}
		}
	}
	return n
}
