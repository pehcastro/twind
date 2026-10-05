package highlight

import (
	"slices"
	"strings"
)

func YAML() *Grammar {
	ws := []string{" ", "\t"}
	eol := []string{"\n", "\r"}
	wsEOL := slices.Concat(ws, eol)
	nameChars := []string{"_", "-", ".", "/", ":", "@", "$", "#", "!", "%", "?", "+", "=", "~", "'", "(", ")"}
	tagChars := []string{"_", "-", ".", "/", ":", "@", "$", "#", "+", "=", "~", "'", "(", ")", "?", "&", ";", "*", "%", "!"}
	comment := Within(Comment, "#", "\n").OneLine()
	space := Match(Text, ws...)
	done := Fallback(Text).Leave()
	shared := []Rule{
		space,
		Match(Text, eol...),
		comment,
		Match(String, `"`).Enter("double_string_body"),
		Within(String, "'", "'"),
		Match(Punctuation, "---", "..."),
		Match(Variable, "&").Enter("anchor_body"),
		Match(Variable, "*").Enter("alias_body"),
		Match(Operator, "!!", "!").Enter("tag_body"),
		Match(Punctuation, "[").Enter("flow_seq"),
		Match(Punctuation, "{").Enter("flow_map"),
	}
	flow := func(closer, other string) []Rule {
		return append(slices.Clone(shared),
			Match(Punctuation, closer).Leave(),
			Match(Punctuation, other).Leave(),
			Match(Punctuation, ":", ","),
			Match(Text, "-").Enter("dash_probe_flow"),
			Match(Text, "?").Enter("question_probe_flow"),
			Match(Punctuation, "@", "`"),
			Fallback(Identifier).Enter("plain_scalar_flow"),
		)
	}
	indicator := func(name, sign, scalar string) []State {
		return []State{
			{Name: name, Probe: true, AtEnd: sign + "_as_punct", Rules: []Rule{
				Match(Text, wsEOL...).Enter(sign + "_as_punct"),
				Fallback(Text).Enter(scalar),
			}},
		}
	}
	header := func(name, body string) State {
		return State{Name: name, Rules: []Rule{
			Match(Operator, "-", "+"),
			Match(Number).Digits(),
			space,
			comment,
			Match(Text, "\n").Goto(body),
			Fallback(String),
		}}
	}
	block := func(name string) []State {
		return []State{
			{Name: name + "_body", Rules: []Rule{Match(Text, "\n").Enter(name + "_nl_probe"), Fallback(String)}},
			{Name: name + "_nl_probe", Probe: true, AtEnd: name + "_nl_stay", Rules: []Rule{
				Match(Text, wsEOL...).Goto(name + "_nl_stay"),
				Fallback(Text).Goto(name + "_nl_exit"),
			}},
			{Name: name + "_nl_stay", Rules: []Rule{Match(String, "\n").Goto(name + "_body")}},
			{Name: name + "_nl_exit", Rules: []Rule{Match(String, "\n").Leave()}},
		}
	}
	scalar := func(name, sign string) State {
		return State{Name: name, Rules: []Rule{Match(Identifier, sign).Goto("plain_scalar")}}
	}
	hex := Match(Escape).HexDigits()
	states := slices.Concat([]State{
		{Name: "main", Rules: append(slices.Clone(shared),
			Match(Operator, "|").Enter("literal_header"),
			Match(Operator, ">").Enter("folded_header"),
			Match(Operator, "%").Enter("directive_name"),
			Match(Punctuation, ":", ","),
			Match(Text, "-").Enter("dash_probe"),
			Match(Text, "?").Enter("question_probe"),
			Match(Punctuation, "@", "`"),
			Fallback(Identifier).Enter("plain_scalar"),
		)},
		{Name: "flow_seq", Rules: flow("]", "}")},
		{Name: "flow_map", Rules: flow("}", "]")},
		{Name: "plain_scalar", Rules: []Rule{
			Match(Text, wsEOL...).Leave(),
			Match(Text, ":").Enter("colon_probe"),
			Fallback(Identifier),
		}},
		{Name: "plain_scalar_flow", Rules: []Rule{
			Match(Text, wsEOL...).Leave(),
			Match(Text, ":").Enter("colon_probe"),
			Match(Text, ",", "[", "]", "{", "}").Leave(),
			Fallback(Identifier),
		}},
		{Name: "colon_probe", Probe: true, AtEnd: "colon_sep", Rules: []Rule{
			Match(Text, wsEOL...).Goto("colon_sep"),
			Fallback(Text).Goto("colon_scalar"),
		}},
		{Name: "colon_sep", Rules: []Rule{Match(Punctuation, ":").Leave()}},
		{Name: "colon_scalar", Rules: []Rule{Match(Identifier, ":").Goto("plain_scalar")}},
		{Name: "dash_as_punct", Rules: []Rule{Match(Punctuation, "-").Leave()}},
		{Name: "question_as_punct", Rules: []Rule{Match(Punctuation, "?").Leave()}},
		scalar("dash_as_scalar", "-"),
		scalar("question_as_scalar", "?"),
		{Name: "dash_as_scalar_flow", Rules: []Rule{Match(Identifier, "-").Goto("plain_scalar_flow")}},
		{Name: "question_as_scalar_flow", Rules: []Rule{Match(Identifier, "?").Goto("plain_scalar_flow")}},
		{Name: "anchor_body", Rules: []Rule{Match(Variable, nameChars...).Letters().Digits(), done}},
		{Name: "alias_body", Rules: []Rule{Match(Variable, nameChars...).Letters().Digits(), done}},
		{Name: "tag_body", Rules: []Rule{
			Match(Operator, "<").Goto("tag_verbatim"),
			Match(Keyword, tagChars...).Letters().Digits(),
			done,
		}},
		{Name: "tag_verbatim", Rules: []Rule{Match(Operator, ">").Leave(), Fallback(Keyword)}},
		{Name: "directive_name", Rules: []Rule{
			Match(Text, ws...).Goto("directive_body"),
			Match(Text, eol...).Leave(),
			Match(Keyword, "_").Letters().Digits(),
			Fallback(Text).Goto("directive_body"),
		}},
		{Name: "directive_body", Rules: []Rule{space, Match(Text, eol...).Leave(), Fallback(Identifier)}},
		header("literal_header", "literal_body"),
		header("folded_header", "folded_body"),
		{Name: "double_string_body", Rules: []Rule{
			Match(Escape, `\x`).Enter("esc_hex_d1"),
			Match(Escape, `\u`).Enter("esc_u4_d1"),
			Match(Escape, `\U`).Enter("esc_u8_d1"),
			Match(Escape, `\`).Enter("esc_simple"),
			Match(String, `"`).Leave(),
			Fallback(String),
		}},
		{Name: "esc_simple", Rules: []Rule{Fallback(Escape).Leave()}},
	},
		indicator("dash_probe", "dash", "dash_as_scalar"),
		indicator("question_probe", "question", "question_as_scalar"),
		indicator("dash_probe_flow", "dash", "dash_as_scalar_flow"),
		indicator("question_probe_flow", "question", "question_as_scalar_flow"),
		block("literal"),
		block("folded"),
		digits("esc_hex_d", 2, hex),
		digits("esc_u4_d", 4, hex),
		digits("esc_u8_d", 8, hex),
	)
	return mustCompile(states, yamlPasses)
}

func yamlPasses(w *work) {
	for i, s := range w.spans {
		if s.Kind != Identifier {
			continue
		}
		switch t := w.text(i); t {
		case "true", "True", "TRUE", "false", "False", "FALSE", "yes", "Yes", "YES", "no", "No", "NO",
			"on", "On", "ON", "off", "Off", "OFF", "y", "Y", "n", "N":
			w.spans[i].Kind = Boolean
		case "null", "Null", "NULL", "~":
			w.spans[i].Kind = Null
		default:
			if yamlNumber(t) {
				w.spans[i].Kind = Number
			}
		}
	}
	for i, s := range w.spans {
		if s.Kind == Identifier && w.is(w.next(i+1), Punctuation, ":") {
			w.spans[i].Kind = Property
		}
	}
}

func yamlNumber(t string) bool {
	signed := strings.HasPrefix(t, "+") || strings.HasPrefix(t, "-")
	body := t
	if signed {
		body = t[1:]
	}
	switch {
	case body == ".inf" || body == ".Inf" || body == ".INF":
		return true
	case !signed && (body == ".nan" || body == ".NaN" || body == ".NAN"):
		return true
	case !signed && strings.HasPrefix(body, "0x"):
		return len(body) > 2 && strings.Trim(body[2:], "0123456789abcdefABCDEF") == ""
	case !signed && strings.HasPrefix(body, "0o"):
		return len(body) > 2 && strings.Trim(body[2:], "01234567") == ""
	}
	pos := 0
	run := func() int {
		start := pos
		for pos < len(body) && body[pos] >= '0' && body[pos] <= '9' {
			pos++
		}
		return pos - start
	}
	whole, fraction := run(), 0
	if strings.HasPrefix(body[pos:], ".") {
		pos++
		fraction = run()
	}
	if whole+fraction == 0 {
		return false
	}
	if pos < len(body) && (body[pos] == 'e' || body[pos] == 'E') {
		pos++
		if pos < len(body) && (body[pos] == '+' || body[pos] == '-') {
			pos++
		}
		if run() == 0 {
			return false
		}
	}
	return pos == len(body)
}
