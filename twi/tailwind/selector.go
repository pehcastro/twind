package tailwind

import (
	"cmp"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/twind-dev/twind/twi/style"
)

func selector(sel string, when style.Condition) (style.Rule, string) {
	r := style.Rule{Class: sel, When: when}
	r.When.Attrs = slices.Clone(when.Attrs)
	rest := sel
	if inner, after, ok := call(rest, ":where("); ok && strings.HasPrefix(after, " ") {
		m, left, reason := compound(inner)
		if reason = cmp.Or(reason, leftover(left)); reason != "" {
			return r, reason
		}
		m.Relation = style.RelationAncestor
		r.Near, rest = m, after[1:]
	}
	if inner, after, ok := call(rest, ":is("); ok {
		rest = inner + after
	}
	if !strings.HasPrefix(rest, ".") {
		return r, "selector " + strconv.Quote(sel) + " is not a class"
	}
	anchor, rest, reason := compound(rest)
	r.Class = anchor.Class
	for reason == "" && strings.HasPrefix(rest, ":") {
		var near, more style.Match
		if inner, after, ok := call(rest, ":is("); ok {
			near, reason = relative(inner)
			rest = after
		} else if inner, after, ok := call(rest, ":has("); ok {
			near, reason = has(inner)
			rest = after
		} else {
			return r, "pseudo-class " + strconv.Quote(rest) + " has no terminal state"
		}
		if reason != "" || r.Near.Relation != style.RelationSelf {
			return r, cmp.Or(reason, "more than one relative in "+strconv.Quote(sel))
		}
		r.Near = near
		more, rest, reason = compound(rest)
		anchor.States, anchor.Attrs = anchor.States|more.States, append(anchor.Attrs, more.Attrs...)
	}
	if reason != "" {
		return r, reason
	}
	r.When.States |= anchor.States
	r.When.Attrs = append(r.When.Attrs, anchor.Attrs...)
	if rest == "" {
		return r, ""
	}
	relation := style.RelationDescendant
	if after, ok := strings.CutPrefix(rest, " > "); ok {
		relation, rest = style.RelationChild, after
	} else if after, ok := strings.CutPrefix(rest, " "); ok {
		rest = after
	} else {
		return r, "combinator in " + strconv.Quote(sel)
	}
	if r.Near.Relation != style.RelationSelf {
		return r, "a relative and a target on one rule"
	}
	r.Target, rest, reason = compound(rest)
	r.Target.Relation = relation
	return r, cmp.Or(reason, leftover(rest))
}

func leftover(rest string) string {
	if rest == "" {
		return ""
	}
	return "selector part " + strconv.Quote(rest) + " has no terminal meaning"
}

func relative(inner string) (style.Match, string) {
	marker, after, ok := call(inner, ":where(")
	var m style.Match
	switch {
	case !ok:
		return m, "relative " + strconv.Quote(inner) + " is not a group or a peer"
	case strings.HasSuffix(after, " ~ *"):
		m.Relation, after = style.RelationPrevious, strings.TrimSuffix(after, " ~ *")
	case strings.HasSuffix(after, " *"):
		m.Relation, after = style.RelationAncestor, strings.TrimSuffix(after, " *")
	default:
		return m, "relative " + strconv.Quote(inner) + " is not an ancestor or an earlier sibling"
	}
	group, left, reason := compound(marker)
	if reason = cmp.Or(reason, leftover(left)); reason != "" || group.Class == "" {
		return m, cmp.Or(reason, "relative "+strconv.Quote(marker)+" has no class")
	}
	test, left, reason := compound(after)
	if reason = cmp.Or(reason, leftover(left)); reason != "" || test.Class != "" || test.Element != style.ElementAny {
		return m, cmp.Or(reason, "relative "+strconv.Quote(after)+" tests more than a state or an attribute")
	}
	m.Class, m.States, m.Attrs = group.Class, test.States, test.Attrs
	return m, ""
}

func has(inner string) (style.Match, string) {
	inner = strings.TrimSpace(inner)
	relation := style.RelationDescendant
	if after, ok := strings.CutPrefix(inner, ">"); ok {
		relation, inner = style.RelationChild, strings.TrimSpace(after)
	}
	if in, after, ok := call(inner, ":is("); ok && after == "" {
		inner = in
	}
	m, left, reason := compound(inner)
	m.Relation = relation
	return m, cmp.Or(reason, leftover(left))
}

func call(s, name string) (string, string, bool) {
	if !strings.HasPrefix(s, name) {
		return "", s, false
	}
	depth := 1
	for i := len(name); i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '(':
			depth++
		case ')':
			if depth--; depth == 0 {
				return s[len(name):i], s[i+1:], true
			}
		}
	}
	return "", s, false
}

func compound(s string) (style.Match, string, string) {
	var m style.Match
	switch {
	case strings.HasPrefix(s, "*"):
		s = s[1:]
	case strings.HasPrefix(s, "."):
		m.Class, s = ident(s[1:])
	case s != "" && 'a' <= s[0] && s[0] <= 'z':
		var name string
		name, s = ident(s)
		element, ok := map[string]style.Element{"a": style.ElementA, "button": style.ElementButton, "img": style.ElementImg, "input": style.ElementInput, "kbd": style.ElementKbd, "label": style.ElementLabel, "p": style.ElementP, "select": style.ElementSelect, "span": style.ElementSpan, "svg": style.ElementSVG, "textarea": style.ElementTextarea}[name]
		if !ok {
			return m, s, "element " + strconv.Quote(name) + " has no terminal node"
		}
		m.Element = element
	}
	for s != "" {
		switch s[0] {
		case ':':
			name, after := ident(s[1:])
			if strings.HasPrefix(after, "(") {
				return m, s, ""
			}
			state, ok := map[string]style.State{"hover": style.StateHover, "focus": style.StateFocus, "focus-visible": style.StateFocusVisible, "active": style.StateActive, "disabled": style.StateDisabled, "focus-within": style.StateFocusWithin, "checked": style.StateChecked}[name]
			if !ok {
				return m, s, "pseudo-class " + strconv.Quote(s) + " has no terminal state"
			}
			m.States |= state
			s = after
		case '[':
			end := strings.IndexByte(s, ']')
			if end < 0 {
				return m, s, "attribute selector not closed"
			}
			name, value, hasValue := strings.Cut(s[1:end], "=")
			if strings.ContainsAny(name, "~|^$*") {
				return m, s, "attribute operator in " + strconv.Quote(s[:end+1])
			}
			m.Attrs = append(m.Attrs, style.Attr{Name: name, Value: strings.Trim(value, `"'`), AnyValue: !hasValue})
			s = s[end+1:]
		default:
			return m, s, ""
		}
	}
	return m, "", ""
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
	switch query {
	case "(hover:hover)":
		return when, ""
	case "(prefers-color-scheme:dark)":
		when.Scheme = style.SchemeDark
		return when, ""
	case "(prefers-color-scheme:light)":
		when.Scheme = style.SchemeLight
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
