package tailwind

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/twind-dev/twind/twi/style"
)

func selector(sel string, when style.Condition) (style.Rule, string) {
	if !strings.HasPrefix(sel, ".") {
		return style.Rule{Class: sel}, "selector " + strconv.Quote(sel) + " is not a class"
	}
	class, rest := ident(sel[1:])
	r := style.Rule{Class: class, When: when}
	r.When.Attrs = append([]style.Attr(nil), when.Attrs...)
	for rest != "" {
		switch rest[0] {
		case ':':
			name, after := ident(rest[1:])
			state, ok := map[string]style.State{"hover": style.StateHover, "focus": style.StateFocus, "focus-visible": style.StateFocusVisible, "active": style.StateActive, "disabled": style.StateDisabled}[name]
			if !ok || strings.HasPrefix(after, "(") {
				return r, "pseudo-class " + strconv.Quote(rest) + " has no terminal state"
			}
			r.When.States |= state
			rest = after
		case '[':
			end := strings.IndexByte(rest, ']')
			if end < 0 {
				return r, "attribute selector not closed"
			}
			name, value, hasValue := strings.Cut(rest[1:end], "=")
			r.When.Attrs = append(r.When.Attrs, style.Attr{Name: name, Value: strings.Trim(value, `"'`), AnyValue: !hasValue})
			rest = rest[end+1:]
		default:
			return r, "combinator in " + strconv.Quote(sel)
		}
	}
	return r, ""
}

func ident(s string) (string, string) {
	var b strings.Builder
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s):
			hex := i + 1
			for hex < len(s) && hex < i+7 && strings.IndexByte("0123456789abcdefABCDEF", s[hex]) >= 0 {
				hex++
			}
			if hex == i+1 {
				r, size := utf8.DecodeRuneInString(s[i+1:])
				b.WriteRune(r)
				i += 1 + size
				continue
			}
			n, _ := strconv.ParseUint(s[i+1:hex], 16, 32)
			b.WriteRune(rune(n))
			i = hex
			if i < len(s) && s[i] == ' ' {
				i++
			}
		case c == '-' || c == '_' || c >= utf8.RuneSelf || '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
			b.WriteByte(c)
			i++
		default:
			return b.String(), s[i:]
		}
	}
	return b.String(), ""
}

func media(when style.Condition, prelude string) (style.Condition, string) {
	query := strings.ReplaceAll(prelude, " ", "")
	if query == "(hover:hover)" {
		return when, ""
	}
	var bound *int
	var value string
	for _, form := range []struct {
		prefix string
		bound  *int
	}{{"(width>=", &when.MinCols}, {"(min-width:", &when.MinCols}, {"(width<", &when.BelowCols}} {
		if v, ok := strings.CutPrefix(query, form.prefix); ok {
			bound, value = form.bound, strings.TrimSuffix(v, ")")
		}
	}
	if bound == nil {
		return when, "media query " + strconv.Quote(prelude) + " has no terminal meaning"
	}
	q, err := dimension(value)
	if err != nil || q.unit != "px" || q.value != math.Trunc(q.value) {
		return when, "breakpoint " + value + " is not a whole number of cells"
	}
	*bound = int(q.value)
	return when, ""
}
