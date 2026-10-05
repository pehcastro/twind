package highlight

import (
	"slices"
	"strings"
)

type dialect uint8

const (
	javaScript dialect = iota
	typeScript
	typeScriptJSX
)

func to(r Rule, dest string) Rule {
	if dest == "" {
		return r
	}
	return r.Goto(dest)
}

func scriptStates(d dialect) []State {
	words := strings.Fields
	keywordList := words("if else switch case default while do for break continue return var let const function class " +
		"extends static async await new typeof instanceof in of delete void try catch finally throw import export from as " +
		"this super null undefined debugger with yield get set")
	if d != javaScript {
		keywordList = append(keywordList, words("type interface enum namespace module declare abstract readonly public "+
			"private protected override accessor as satisfies keyof infer is using implements")...)
	}
	regexBefore := words("return throw typeof new void delete in of instanceof yield await case else")
	var division []string
	for _, k := range keywordList {
		if !slices.Contains(regexBefore, k) {
			division = append(division, k)
		}
	}
	opsAll := words(">>>= === !== >>> <<= >>= **= &&= ||= ??= ... ++ -- <= >= == != && || << >> ** ?? ?. => += -= *= /= " +
		"%= &= |= ^= - + < > = ! & | ? * ~ ^ %")
	opsOutsideMember := slices.DeleteFunc(slices.Clone(opsAll), func(op string) bool { return op == "?." })
	probeEnds := append(words(". ) [ ] { } ; , : ` === !== -- ++ <= >= == != && || - + < > = ! & | ? * / ~ ^ %"), " ", "\t", "\n", "\r")
	space := []string{" ", "\t", "\n", "\r"}
	done := Fallback(Text).Leave()
	comments := []Rule{Within(Comment, "//", "\n"), Within(Comment, "/*", "*/")}
	template := Match(Template, "`").Enter("template_literal")
	strs := []Rule{Match(String, `"`).Enter("string_double"), Match(String, "'").Enter("string_single"), template}
	numbers := func(push bool) []Rule {
		move := func(r Rule, state string) Rule {
			if push {
				return r.Enter(state + "_arg")
			}
			return r.Goto(state)
		}
		return []Rule{
			move(Match(Number, "0x", "0X"), "hex_number"),
			move(Match(Number, "0b", "0B"), "binary_number"),
			move(Match(Number, "0o", "0O"), "octal_number"),
			move(Match(Number).Digits(), "number"),
		}
	}
	whitespace := Match(Text, space...)
	common := slices.Concat(comments, strs, numbers(false), []Rule{whitespace})
	bodyCommon := slices.Concat(comments, strs, numbers(true))
	tmplCommon := append(slices.Clone(bodyCommon), whitespace)
	member := func(state string) []Rule {
		return []Rule{Match(Operator, "?.").Goto(state), Match(Punctuation, ".").Goto(state)}
	}
	probe := func(state string) Rule { return Match(Text, "_", "$").Letters().Goto(state) }
	keywords := func(regexDest, divDest string) []Rule {
		return []Rule{
			to(Word(Keyword, regexBefore...), regexDest),
			to(Word(Keyword, division...), divDest),
			to(Word(Keyword, "undefined", "null", "NaN", "Infinity"), divDest),
			to(Word(Boolean, "true", "false"), divDest),
		}
	}
	var decorator []Rule
	if d != javaScript {
		decorator = []Rule{Match(Decorator, "@").Enter("decorator")}
	}
	callOps := []Rule{Match(Operator, "===", "!=="), Match(Operator, "--", "++", "<=", ">=", "==", "!=", "&&", "||")}
	singleOps := words("- + < > = ! & | ? * ~ ^ %")
	ops := func(after string) Rule { return to(Match(Operator, opsOutsideMember...), after) }
	escapes := []Rule{
		Match(Escape, `\u{`).Enter("esc_u_braces"),
		Match(Escape, `\u`).Enter("esc_u4_d1"),
		Match(Escape, `\x`).Enter("esc_hex_d1"),
		Match(Escape, `\`).Enter("esc_simple"),
	}
	afterName := func(regexAllow, division, memberState string) []Rule {
		return slices.Concat(
			[]Rule{Match(Punctuation, ",", ";", ":").Goto(regexAllow)},
			member(memberState),
			[]Rule{Match(Operator, append(slices.Clone(opsOutsideMember), "/")...).Goto(regexAllow), Match(Text, space...).Goto(division)},
		)
	}
	numberStates := func(suffix string) []State {
		exit := func(r Rule) Rule {
			if suffix == "" {
				return r.Goto("division")
			}
			return r.Leave()
		}
		name := func(s string) string { return s + suffix }
		out := exit(Fallback(Text))
		return []State{
			{Name: name("number"), Rules: []Rule{
				Match(Number, "_").Digits(),
				Match(Number, ".").Goto(name("decimal_number")),
				Match(Number, "e", "E").Goto(name("exponent_sign")),
				exit(Match(Number, "n")),
				out,
			}},
			{Name: name("decimal_number"), Rules: []Rule{Match(Number, "_").Digits(), Match(Number, "e", "E").Goto(name("exponent_sign")), out}},
			{Name: name("exponent_sign"), Rules: []Rule{
				Match(Number, "+", "-").Goto(name("exponent_digits")),
				Match(Number).Digits().Goto(name("exponent_digits")),
				out,
			}},
			{Name: name("exponent_digits"), Rules: []Rule{Match(Number, "_").Digits(), out}},
			{Name: name("hex_number"), Rules: []Rule{Match(Number, "_").HexDigits(), exit(Match(Number, "n")), out}},
			{Name: name("binary_number"), Rules: []Rule{Match(Number, "_", "0", "1"), exit(Match(Number, "n")), out}},
			{Name: name("octal_number"), Rules: []Rule{Match(Number, "_").Range('0', '7'), exit(Match(Number, "n")), out}},
		}
	}
	hex := Match(Escape).HexDigits()
	states := slices.Concat([]State{
		{Name: "regex_allow", Rules: slices.Concat(common, []Rule{ops("")}, keywords("", "division"), decorator, []Rule{
			Match(Regex, "/").Enter("regex_pattern"),
			Match(Punctuation, "(", "{", "["),
			Match(Punctuation, ")", "}", "]").Goto("division"),
		}, member("member_access"), []Rule{Match(Punctuation, ";", ",", ":"), probe("identifier_probe")})},
		{Name: "member_access", Rules: slices.Concat(comments, []Rule{whitespace, probe("identifier_probe"), Fallback(Text).Goto("division")})},
		{Name: "member_access_tmpl", Rules: slices.Concat(comments, []Rule{
			whitespace, probe("identifier_probe_tmpl"), Fallback(Text).Goto("tmpl_division"),
		})},
		{Name: "identifier_probe", Probe: true, AtEnd: "identifier", Rules: []Rule{
			Match(Text, "(").Goto("function_name"),
			Match(Text, probeEnds...).Goto("identifier"),
		}},
		{Name: "function_name", Rules: []Rule{
			Match(Function, "_", "$").Letters().Digits(),
			Match(Punctuation, "(").Goto("function_body"),
			Match(Text, " ", "\t"),
		}},
		{Name: "function_body", Rules: slices.Concat(bodyCommon, keywords("regex_allow", "division"), []Rule{
			Match(Punctuation, ")").Goto("division"),
			Match(Punctuation, "(").Enter("paren_group"),
			Match(Punctuation, ","),
		}, callOps, []Rule{
			Match(Operator, singleOps...),
			Match(Regex, "/").Enter("regex_pattern"),
			Match(Punctuation, "[", "]", "{", "}"),
		}, member("member_access"), []Rule{Match(Punctuation, ";", ":"), probe("identifier_probe")})},
		{Name: "paren_group", Rules: slices.Concat(bodyCommon, keywords("", ""), []Rule{
			Match(Punctuation, ")").Leave(),
			Match(Punctuation, ","),
		}, member("member_access_paren"), []Rule{Match(Punctuation, "[", "]", "{", "}", ";", ":")}, callOps, []Rule{
			Match(Operator, words("- + < > = ! & | ? * / ~ ^ %")...),
			Match(Identifier, "_", "$").Letters().Digits(),
			Match(Punctuation, "(").Enter("paren_group"),
		})},
		{Name: "member_access_paren", Rules: slices.Concat(comments, []Rule{
			whitespace,
			Match(Identifier, "_", "$").Letters().Digits(),
			Fallback(Text).Goto("paren_group"),
		})},
		{Name: "identifier", Rules: slices.Concat([]Rule{
			Match(Identifier, "_", "$").Letters().Digits(),
			template,
			Match(Punctuation, "(", "[", "{").Goto("regex_allow"),
			Match(Punctuation, ")", "]", "}").Goto("division"),
		}, afterName("regex_allow", "division", "member_access"))},
		{Name: "division", Rules: slices.Concat(common, []Rule{ops("regex_allow")}, keywords("regex_allow", ""), decorator, []Rule{
			Match(Punctuation, "(", "{", "[").Goto("regex_allow"),
			Match(Operator, "/").Goto("regex_allow"),
			Match(Punctuation, ")", "}", "]"),
			Match(Punctuation, ";", ",", ":").Goto("regex_allow"),
		}, member("member_access"), []Rule{probe("identifier_probe")})},
		{Name: "string_double", Rules: append(slices.Clone(escapes), Match(String, `"`).Leave(), Fallback(String))},
		{Name: "string_single", Rules: append(slices.Clone(escapes), Match(String, "'").Leave(), Fallback(String))},
		{Name: "esc_simple", Rules: []Rule{Fallback(Escape).Leave()}},
		{Name: "esc_u_braces", Rules: []Rule{hex, Match(Escape, "}").Leave(), done}},
		{Name: "template_literal", Rules: slices.Concat([]Rule{Match(Punctuation, "${").Enter("tmpl_regex_allow")}, escapes,
			[]Rule{Match(Template, "`").Leave(), Fallback(Template)})},
		{Name: "tmpl_regex_allow", Rules: slices.Concat(tmplCommon, []Rule{ops("")}, keywords("", "tmpl_division"), decorator, []Rule{
			Match(Punctuation, "}").Leave(),
			Match(Regex, "/").Enter("regex_pattern"),
			Match(Punctuation, "{").Enter("tmpl_regex_allow"),
			Match(Punctuation, "(", "["),
			Match(Punctuation, ")", "]").Goto("tmpl_division"),
		}, member("member_access_tmpl"), []Rule{Match(Punctuation, ";", ",", ":"), probe("identifier_probe_tmpl")})},
		{Name: "tmpl_division", Rules: slices.Concat(tmplCommon, []Rule{ops("tmpl_regex_allow")}, keywords("tmpl_regex_allow", ""), decorator, []Rule{
			Match(Punctuation, "}").Leave(),
			Match(Punctuation, "{").Enter("tmpl_regex_allow"),
			Match(Punctuation, "(", "[").Goto("tmpl_regex_allow"),
			Match(Operator, "/").Goto("tmpl_regex_allow"),
			Match(Punctuation, ")", "]"),
			Match(Punctuation, ";", ",", ":").Goto("tmpl_regex_allow"),
		}, member("member_access_tmpl"), []Rule{probe("identifier_probe_tmpl")})},
		{Name: "identifier_probe_tmpl", Probe: true, AtEnd: "identifier_tmpl", Rules: []Rule{
			Match(Text, "(").Goto("function_name_tmpl"),
			Match(Text, probeEnds...).Goto("identifier_tmpl"),
		}},
		{Name: "function_name_tmpl", Rules: []Rule{
			Match(Function, "_", "$").Letters().Digits(),
			Match(Punctuation, "(").Goto("function_body_tmpl"),
			Match(Text, " ", "\t"),
		}},
		{Name: "function_body_tmpl", Rules: slices.Concat(bodyCommon, []Rule{
			Match(Punctuation, ")").Goto("tmpl_division"),
			Match(Punctuation, "(").Enter("paren_group"),
			Match(Punctuation, ","),
		}, callOps, []Rule{
			Match(Operator, singleOps...),
			Match(Regex, "/").Enter("regex_pattern"),
			Match(Punctuation, "{").Enter("tmpl_regex_allow"),
			Match(Punctuation, "}").Leave(),
			Match(Punctuation, "[", "]"),
			Match(Punctuation, ";", ".", ":"),
			probe("identifier_probe_tmpl"),
		})},
		{Name: "identifier_tmpl", Rules: slices.Concat([]Rule{
			Match(Identifier, "_", "$").Letters().Digits(),
			template,
			Match(Punctuation, "{").Enter("tmpl_regex_allow"),
			Match(Punctuation, "(", "[").Goto("tmpl_regex_allow"),
			Match(Punctuation, "}").Leave(),
			Match(Punctuation, ")", "]").Goto("tmpl_division"),
		}, afterName("tmpl_regex_allow", "tmpl_division", "member_access_tmpl"))},
		{Name: "regex_pattern", Rules: []Rule{
			Match(Regex, "/").Goto("regex_flags"),
			Match(Regex, `\`).Enter("regex_escape"),
			Match(Regex, "[").Enter("regex_class"),
			Match(Text, "\n", ";").Goto("division"),
			Fallback(Regex),
		}},
		{Name: "regex_escape", Rules: []Rule{Fallback(Regex).Leave()}},
		{Name: "regex_class", Rules: []Rule{Match(Regex, "]").Leave(), Match(Regex, `\`).Enter("regex_class_escape"), Fallback(Regex)}},
		{Name: "regex_class_escape", Rules: []Rule{Fallback(Regex).Leave()}},
		{Name: "regex_flags", Rules: []Rule{Match(Regex, "g", "i", "m", "s", "u", "y", "d"), done}},
	}, numberStates(""), numberStates("_arg"), digits("esc_hex_d", 2, hex), digits("esc_u4_d", 4, hex))
	if d != javaScript {
		states = append(states, State{Name: "decorator", Rules: []Rule{Match(Decorator, "_", "$", ".").Letters().Digits(), done}})
	}
	if d != typeScriptJSX {
		return states
	}
	lessThan := []Rule{Match(Operator, "<<=", "<<", "<="), Match(Text, "<").Enter("jsx_or_lt_probe")}
	notLess := slices.DeleteFunc(slices.Clone(opsAll), func(op string) bool { return op == "<" || op == "<<=" || op == "<<" || op == "<=" })
	jsxKeywords := func(divDest string) []Rule {
		kw := keywords("", divDest)
		return []Rule{kw[0], kw[1], kw[3], kw[2], to(Word(Type, words("number string boolean any never unknown object symbol bigint")...), divDest)}
	}
	expression := func(base []Rule, divDest string) []Rule {
		return slices.Concat(base, lessThan, []Rule{Match(Operator, notLess...)}, jsxKeywords(divDest), decorator)
	}
	for i, s := range states {
		switch s.Name {
		case "regex_allow":
			states[i].Rules = slices.Concat(expression(common, "division"), []Rule{
				Match(Regex, "/").Enter("regex_pattern"),
				Match(Punctuation, "(", "{", "["),
				Match(Punctuation, ")", "}", "]").Goto("division"),
				Match(Punctuation, ";", ",", ".", ":"),
				probe("identifier_probe"),
			})
		case "tmpl_regex_allow":
			states[i].Rules = slices.Concat(expression(tmplCommon, "tmpl_division"), []Rule{
				Match(Punctuation, "}").Leave(),
				Match(Regex, "/").Enter("regex_pattern"),
				Match(Punctuation, "{").Enter("tmpl_regex_allow"),
				Match(Punctuation, "(", "["),
				Match(Punctuation, ")", "]").Goto("tmpl_division"),
				Match(Punctuation, ";", ",", ".", ":"),
				probe("identifier_probe_tmpl"),
			})
		}
	}
	name := Match(TagName, "_", "$", "-").Letters().Digits()
	return append(states,
		State{Name: "jsx_or_lt_probe", Probe: true, AtEnd: "jsx_lt_emit", Rules: []Rule{
			Match(Text, ">").Enter("jsx_fragment_start"),
			Match(Text, "_", "$").Letters().Enter("jsx_name_probe"),
			Match(Text, append(words(`= < ! & | ? * + - . % ^ ~ / ( ) [ ] { } ; , : " ' `+"`"+` @ # \`), space...)...).Digits().Enter("jsx_lt_emit"),
		}},
		State{Name: "jsx_name_probe", Probe: true, AtEnd: "jsx_tag_start", Rules: []Rule{
			Match(Text, ",").Enter("jsx_lt_emit"),
			Word(Keyword, "extends").Enter("jsx_lt_emit"),
			Match(Text, words(`> / = { : . ( ) [ ] } ;`)...).Enter("jsx_tag_start"),
			Match(Text, words(`" ' `+"`"+` ! ? | & * + % ^ ~ @ # \`)...).Enter("jsx_tag_start"),
		}},
		State{Name: "jsx_lt_emit", Rules: []Rule{Match(Operator, "<").Leave()}},
		State{Name: "jsx_tag_start", Rules: []Rule{Match(Punctuation, "<").Goto("jsx_tag_name")}},
		State{Name: "jsx_fragment_start", Rules: []Rule{Match(Punctuation, "<>").Goto("jsx_children")}},
		State{Name: "jsx_tag_name", Rules: []Rule{
			name,
			Match(Punctuation, ".", ":"),
			Match(Punctuation, "/>").Leave(),
			Match(Punctuation, ">").Goto("jsx_children"),
			Match(Text, space...).Goto("jsx_tag_attrs"),
		}},
		State{Name: "jsx_tag_attrs", Rules: slices.Concat([]Rule{whitespace}, comments, []Rule{
			Match(Punctuation, "/>").Leave(),
			Match(Punctuation, ">").Goto("jsx_children"),
			Match(Operator, "=").Enter("jsx_attr_value"),
			Match(Punctuation, "{").Enter("tmpl_regex_allow"),
			Match(AttrName, "_", "$", "-").Letters().Digits(),
			Match(Punctuation, ":", "."),
		})},
		State{Name: "jsx_attr_value", Rules: []Rule{
			whitespace,
			Match(String, `"`).Goto("jsx_attr_string_double"),
			Match(String, "'").Goto("jsx_attr_string_single"),
			Match(Punctuation, "{").Goto("tmpl_regex_allow"),
			Match(Punctuation, "<").Goto("jsx_tag_name"),
			done,
		}},
		State{Name: "jsx_attr_string_double", Rules: []Rule{Match(String, `"`).Leave(), Match(Entity, "&").Enter("jsx_entity"), Fallback(String)}},
		State{Name: "jsx_attr_string_single", Rules: []Rule{Match(String, "'").Leave(), Match(Entity, "&").Enter("jsx_entity"), Fallback(String)}},
		State{Name: "jsx_entity", Rules: []Rule{Match(Entity, ";").Leave(), Match(Entity, "#").Letters().Digits(), done}},
		State{Name: "jsx_children", Rules: []Rule{
			Match(Entity, "&").Enter("jsx_entity"),
			Match(Punctuation, "{").Enter("tmpl_regex_allow"),
			Match(Punctuation, "</>").Leave(),
			Match(Punctuation, "</").Goto("jsx_close_name"),
			Match(Punctuation, "<>").Enter("jsx_children"),
			Match(Punctuation, "<").Enter("jsx_tag_name"),
			Fallback(Text),
		}},
		State{Name: "jsx_close_name", Rules: []Rule{name, Match(Punctuation, ".", ":"), whitespace, Match(Punctuation, ">").Leave()}},
	)
}
