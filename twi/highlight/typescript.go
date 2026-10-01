package highlight

func TypeScript() *Grammar { return script(typeScript) }

func TSX() *Grammar { return script(typeScriptJSX) }

func script(d dialect) *Grammar {
	g := mustCompile(scriptStates(d), nil)
	js := mustCompile(scriptStates(javaScript), nil)
	page := html()
	sub := embeds{css: CSS(), html: page, jsdoc: jsdoc()}
	rules := newScriptRules()
	page.reclassify = func(w *work) { w.embedRaw(js, sub.css) }
	js.reclassify = scriptPipeline(false, rules, sub)
	g.reclassify = scriptPipeline(true, rules, sub)
	return g
}

func jsdoc() *Grammar {
	gutter := Match(Comment, " ", "\t", "\r", "\n", "*")
	nameChars := func(kind Kind) Rule { return Match(kind, "_", "$", ".").Letters().Digits() }
	afterTag := func(extra ...Rule) []Rule {
		return append([]Rule{
			Match(Comment, " ", "\t"),
			Match(Punctuation, "{").Enter("type_expr"),
		}, append(extra, Fallback(Text).Goto("prose"))...)
	}
	newline := Match(Comment, "\n").Goto("line_start")
	return mustCompile([]State{
		{Name: "line_start", Rules: []Rule{
			gutter,
			Word(Keyword, "@param", "@arg", "@argument", "@property", "@prop").Goto("after_name_tag"),
			Word(Keyword, "@typedef", "@callback", "@template").Goto("after_type_tag"),
			Match(Keyword, "@").Goto("tag"),
			Fallback(Text).Goto("prose"),
		}},
		{Name: "tag", Rules: []Rule{Match(Keyword, "-", "_").Letters().Digits(), Fallback(Text).Goto("after_tag")}},
		{Name: "after_tag", Rules: afterTag(newline)},
		{Name: "after_name_tag", Rules: afterTag(Match(Punctuation, "["), newline, Match(Text, "_", "$").Letters().Goto("name"))},
		{Name: "after_type_tag", Rules: afterTag(newline, Match(Text, "_", "$").Letters().Goto("type_name"))},
		{Name: "name", Rules: []Rule{nameChars(Parameter), Fallback(Text).Goto("prose")}},
		{Name: "type_name", Rules: []Rule{nameChars(Type), Fallback(Text).Goto("prose")}},
		{Name: "type_expr", Rules: []Rule{
			Match(Punctuation, "}").Leave(),
			Match(Punctuation, "{").Enter("type_expr"),
			Match(Keyword, "@").Goto("inline_tag"),
			nameChars(Type),
			Match(Punctuation, "<", ">", "[", "]", "(", ")", ",", "|", "&", "=", "?", "!", "*", ":", ";", "'", `"`),
			gutter,
			Fallback(Comment),
		}},
		{Name: "inline_tag", Rules: []Rule{Match(Keyword, "-").Letters().Digits(), Fallback(Text).Goto("type_expr")}},
		{Name: "prose", Rules: []Rule{newline, Fallback(Comment)}},
	}, nil)
}

func html() *Grammar {
	space := Match(Text, " ", "\t", "\n", "\r")
	name := func(kind Kind) Rule { return Match(kind, "-", "_", ":").Letters().Digits() }
	inside := []Rule{space, Match(Operator, "="), Within(String, `"`, `"`), Within(String, "'", "'"), name(AttrName)}
	closers := func(more ...Rule) []Rule {
		return append([]Rule{Match(Punctuation, "/>").Leave()}, more...)
	}
	raw := func(tag string, kind Kind) []State {
		return []State{
			{Name: tag + "_attrs", Rules: append(closers(Match(Punctuation, ">").Goto(tag+"_content")), inside...)},
			{Name: tag + "_content", Rules: []Rule{Match(Text, "</").Enter(tag + "_close_probe"), Fallback(kind)}},
			{Name: tag + "_close_probe", Probe: true, AtEnd: tag + "_close_fail", Rules: []Rule{Match(Text, tag+">").Goto(tag + "_close_emit")}},
			{Name: tag + "_close_fail", Rules: []Rule{Match(kind, "<").Leave()}},
			{Name: tag + "_close_emit", Rules: []Rule{Match(Punctuation, "</").Goto(tag + "_close_name")}},
			{Name: tag + "_close_name", Rules: []Rule{Match(TagName, tag).Goto(tag + "_close_gt")}},
			{Name: tag + "_close_gt", Rules: []Rule{Match(Punctuation, ">").Leave()}},
		}
	}
	states := []State{
		{Name: "content", Rules: []Rule{
			Within(Comment, "<!--", "-->"),
			Match(Doctype, "<!DOCTYPE", "<!doctype").Enter("doctype"),
			Match(Punctuation, "</").Enter("close_tag"),
			Match(Punctuation, "<").Enter("tag_start"),
			Fallback(Text),
		}},
		{Name: "tag_start", Rules: []Rule{
			Word(TagName, "script").Goto("script_attrs"),
			Word(TagName, "style").Goto("style_attrs"),
			Match(Punctuation, "/>").Leave(),
			Match(Punctuation, ">").Leave(),
			space.Goto("tag_attrs"),
			name(TagName).Goto("tag_open"),
		}},
		{Name: "tag_open", Rules: closers(Match(Punctuation, ">").Leave(), space.Goto("tag_attrs"), name(TagName))},
		{Name: "tag_attrs", Rules: append(closers(Match(Punctuation, ">").Leave()), inside...)},
		{Name: "close_tag", Rules: []Rule{Match(Punctuation, ">").Leave(), space, name(TagName)}},
		{Name: "doctype", Rules: []Rule{Match(Punctuation, ">").Leave(), Fallback(Doctype)}},
	}
	states = append(append(states, raw("script", rawScript)...), raw("style", rawStyle)...)
	return mustCompile(states, nil)
}
