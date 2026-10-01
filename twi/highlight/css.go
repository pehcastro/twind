package highlight

import "strconv"

func CSS() *Grammar {
	comment := Within(Comment, "/*", "*/")
	double := Match(String, `"`).Enter("string_double")
	single := Match(String, "'").Enter("string_single")
	id := Match(SelectorID, "#").Enter("id_selector")
	class := Match(SelectorClass, ".").Enter("class_selector")
	pseudoElement := Match(SelectorPseudo, "::").Enter("pseudo_class")
	pseudoClass := Match(SelectorPseudo, ":").Enter("pseudo")
	attributeOperators := Match(Operator, "~=", "|=", "^=", "$=", "*=", "=")
	combinators := Match(Operator, ">", "+", "~")
	done := Fallback(Text).Leave()
	word := func(kind Kind, extra ...Rule) []Rule {
		return append([]Rule{Match(kind).Letters().Digits(), Match(kind, "-", "_")}, append(extra, done)...)
	}
	whitespace := []string{" ", "\t", "\n", "\r", "\f"}
	var hexEscapes []State
	for n := 2; n <= 6; n++ {
		next := "esc_hex_d" + strconv.Itoa(n+1)
		if n == 6 {
			next = "esc_hex_done"
		}
		hexEscapes = append(hexEscapes, State{Name: "esc_hex_d" + strconv.Itoa(n), Rules: []Rule{
			Match(Escape).HexDigits().Goto(next),
			Match(Escape, whitespace...).Leave(),
			done,
		}})
	}
	body := func(quote string) []Rule {
		return []Rule{Match(Escape, `\`).Enter("esc_start"), Match(String, quote).Leave(), Fallback(String)}
	}
	unitStart := []Rule{Match(Unit).Letters().Enter("unit"), Match(Unit, "%").Leave(), done}
	states := []State{
		{Name: "main", Rules: []Rule{
			comment, double, single,
			Match(Keyword, "@").Enter("at_rule"),
			Match(Punctuation, "{").Enter("declaration"),
			Match(Punctuation, "}"),
			Match(Punctuation, ";", ")", "]", ","),
			Match(Punctuation, "(").Enter("parentheses"),
			Match(Punctuation, "[").Enter("brackets"),
			id, class, pseudoElement, pseudoClass, attributeOperators, combinators,
			Match(Selector, "*"),
			Match(Text, "-").Enter("negative_number_or_identifier"),
			Match(Number).Digits().Enter("number"),
			Match(Selector).Letters().Enter("identifier"),
		}},
		{Name: "declaration", Rules: []Rule{
			comment,
			Match(Punctuation, "}").Leave(),
			Match(Punctuation, "{").Enter("declaration"),
			Match(Punctuation, ";"),
			Match(Keyword, "@").Enter("at_rule"),
			Match(Selector, "&").Enter("nested_selector"),
			class, id, pseudoElement,
			Match(CSSVariable, "--").Enter("css_custom_property_declaration"),
			Match(Punctuation, ":").Enter("value"),
			Match(Text, "-").Enter("probe_identifier"),
			Match(Text).Letters().Enter("probe_identifier"),
		}},
		{Name: "value", Rules: []Rule{
			comment, double, single,
			Match(Punctuation, ";").Leave(),
			Match(Punctuation, "}").Leave(),
			Match(Keyword, "!").Enter("important"),
			Match(Number, "#").Enter("hex_color"),
			Match(Punctuation, "(").Enter("function_args"),
			Match(CSSVariable, "--").Enter("css_custom_property"),
			Match(Operator, "-").Enter("after_minus"),
			Match(Number).Digits().Enter("number"),
			Match(Number, ".").Enter("decimal"),
			Match(Identifier).Letters().Enter("value_identifier"),
			Match(Operator, ",", "/", "+", "*"),
		}},
		{Name: "probe_identifier", Probe: true, AtEnd: "property", Rules: []Rule{
			Match(Text, "{").Enter("selector"),
			Match(Text, ";", "}").Enter("property"),
		}},
		{Name: "nested_selector", Rules: []Rule{
			pseudoClass, pseudoElement, class, id,
			Match(Punctuation, "[").Enter("brackets"),
			Match(Punctuation, "{").Enter("declaration"),
			combinators,
			done,
		}},
		{Name: "at_rule", Rules: []Rule{
			double, single,
			Match(Keyword).Letters(),
			Match(Keyword, "-"),
			Match(Punctuation, "(").Enter("media_params"),
			Match(Punctuation, "{").Leave(),
			Match(Punctuation, ";").Leave(),
		}},
		{Name: "media_params", Rules: []Rule{
			comment, double, single,
			Match(Punctuation, ")").Leave(),
			Match(Punctuation, "(").Enter("media_params"),
			Match(Punctuation, ":"),
			Match(Punctuation, ","),
			Match(Punctuation, ";"),
			Match(Operator, "="),
			Match(Operator, ">", "<"),
			Match(Number).Digits().Enter("number"),
			Match(Number, ".").Enter("decimal"),
			Match(Keyword).Letters().Enter("media_keyword"),
			Match(Keyword, "-").Enter("media_keyword"),
		}},
		{Name: "media_keyword", Rules: []Rule{
			Match(Text, ":", ",", ";", ")", " ", "\t", "\n", "\r").Leave(),
			Match(Keyword).Letters().Digits(),
			Match(Keyword, "-", "_"),
			done,
		}},
		{Name: "property", Rules: word(Property, Match(Punctuation, ":").Enter("value"))},
		{Name: "selector", Rules: word(Selector,
			Match(SelectorPseudo, ":").Enter("pseudo"),
			Match(Punctuation, "{").Enter("declaration"),
		)},
		{Name: "id_selector", Rules: word(SelectorID)},
		{Name: "class_selector", Rules: word(SelectorClass)},
		{Name: "pseudo", Rules: []Rule{
			Match(SelectorPseudo).Letters(),
			Match(SelectorPseudo, "-"),
			Match(Punctuation, "(").Enter("parentheses"),
			done,
		}},
		{Name: "pseudo_class", Rules: []Rule{Match(SelectorPseudo).Letters(), Match(SelectorPseudo, "-"), done}},
		{Name: "parentheses", Rules: []Rule{
			Match(Punctuation, ")").Leave(),
			Match(Punctuation, "(").Enter("parentheses"),
			Match(Identifier).Range(0, 127),
		}},
		{Name: "brackets", Rules: []Rule{
			Match(Punctuation, "]").Leave(),
			double, single, attributeOperators,
			Match(Attribute).Letters().Digits(),
			Match(Attribute, "-", "_"),
		}},
		{Name: "number", Rules: append([]Rule{Match(Number).Digits(), Match(Number, ".").Enter("decimal")}, unitStart...)},
		{Name: "decimal", Rules: append([]Rule{Match(Number).Digits()}, unitStart...)},
		{Name: "after_minus", Rules: []Rule{
			Match(Number).Digits().Enter("number"),
			Match(Number, ".").Enter("decimal"),
			Match(CSSVariable, "-").Enter("css_custom_property"),
			Match(Identifier).Letters().Enter("value_identifier"),
			done,
		}},
		{Name: "unit", Rules: []Rule{Match(Unit).Letters(), done}},
		{Name: "hex_color", Rules: []Rule{Match(Number).HexDigits(), done}},
		{Name: "css_custom_property", Rules: word(CSSVariable)},
		{Name: "css_custom_property_declaration", Rules: word(CSSVariable, Match(Punctuation, ":").Enter("value"))},
		{Name: "value_identifier", Rules: word(Identifier, Match(Punctuation, "(").Enter("function_args"))},
		{Name: "function_args", Rules: []Rule{
			comment, double, single,
			Match(Punctuation, ")").Leave(),
			Match(Punctuation, "(").Enter("function_args"),
			Match(Number, "#").Enter("hex_color"),
			Match(CSSVariable, "--").Enter("css_custom_property"),
			Match(Number).Digits().Enter("number"),
			Match(Identifier, ".").Enter("url_filename"),
			Match(Identifier).Letters().Enter("identifier_in_function"),
			Match(Operator, ",", "/", "+", "*", "-"),
		}},
		{Name: "url_filename", Rules: []Rule{Match(Identifier, ".", "-", "_", "/", ":").Letters().Digits(), done}},
		{Name: "identifier_in_function", Rules: word(Identifier)},
		{Name: "important", Rules: []Rule{Match(Keyword, "important").Leave()}},
		{Name: "identifier", Rules: word(Selector)},
		{Name: "negative_number_or_identifier", Rules: []Rule{
			Match(Number).Digits().Enter("number"),
			Match(Selector).Letters().Enter("identifier"),
			Match(Selector, "-"),
			done,
		}},
		{Name: "string_double", Rules: body(`"`)},
		{Name: "string_single", Rules: body("'")},
		{Name: "esc_start", Rules: []Rule{Match(Escape).HexDigits().Goto("esc_hex_d2"), Fallback(Escape).Leave()}},
		{Name: "esc_hex_done", Rules: []Rule{Match(Escape, whitespace...).Leave(), done}},
	}
	return mustCompile(append(states, hexEscapes...), cssPasses)
}

func cssPasses(w *work) {
	for i, s := range w.spans {
		if s.Kind == Identifier && w.is(w.next(i+1), Punctuation, "(") {
			w.spans[i].Kind = Function
		}
	}
}
