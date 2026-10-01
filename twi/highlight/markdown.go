package highlight

import (
	"slices"
	"strings"
)

func Markdown() *Grammar {
	var escaped []string
	for _, c := range "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~" {
		escaped = append(escaped, `\`+string(c))
	}
	escapes := Match(CharEscape, escaped...)
	links := []Rule{
		Match(linkTextOpen, "![").Enter("link_text"),
		Match(linkTextOpen, "[").Enter("link_text"),
		Match(autolinkOpen, "<").Enter("autolink_body"),
		Match(Entity, "&").Enter("entity_body"),
		Match(codeOpen, "`").Enter("code_body"),
	}
	openers := map[string]Rule{
		"**": Match(boldOpen, "**").Enter("bold_star_body"),
		"__": Match(boldOpen, "__").Enter("bold_under_body"),
		"~~": Match(strikeOpen, "~~").Enter("strike_body"),
		"*":  Match(italicOpen, "*").Enter("italic_star_body"),
		"_":  Match(italicOpen, "_").Enter("italic_under_body"),
	}
	emphasis := func(except string) []Rule {
		var rules []Rule
		for _, d := range []string{"**", "__", "~~", "*", "_"} {
			if d != except {
				rules = append(rules, openers[d])
			}
		}
		return rules
	}
	line := []Rule{Match(HardBreak, "\\\n").Goto("block_start"), escapes}
	nested := []Rule{Match(Text, "\\\n").Leave(), escapes}
	body := func(name string, close Kind, delimiter string, kind Kind) State {
		return State{Name: name, Rules: slices.Concat(
			[]Rule{Match(close, delimiter).Leave(), Match(Text, "\n").Leave()}, nested, links, emphasis(delimiter), []Rule{Fallback(kind)},
		)}
	}
	fence := func(name, mark string) []State {
		var marks []string
		for n := 6; n >= 3; n-- {
			marks = append(marks, strings.Repeat(mark, n))
		}
		return []State{
			{Name: "fence_info_" + name, Rules: []Rule{Match(Text, "\n").Goto("fence_body_" + name), Fallback(CodeLanguage)}},
			{Name: "fence_body_" + name, Rules: []Rule{
				Match(RawCodeBlock, "\n").Goto("fence_maybe_close_" + name),
				Fallback(RawCodeBlock),
			}},
			{Name: "fence_maybe_close_" + name, Rules: []Rule{
				Match(CodeFence, marks...).Goto("fence_close_tail"),
				Fallback(Text).Goto("fence_body_" + name),
			}},
		}
	}
	inline := func(name string, fallback Rule) State {
		return State{Name: name, Rules: slices.Concat(
			[]Rule{Match(Text, "\n").Goto("block_start")}, line, links, emphasis(""), []Rule{fallback},
		)}
	}
	states := slices.Concat([]State{
		{Name: "main", Rules: []Rule{
			Match(FrontMatterMarker, "---\n").Goto("front_matter_body"),
			Fallback(Text).Goto("block_start"),
		}},
		{Name: "front_matter_body", Rules: []Rule{
			Match(FrontMatterMarker, "\n---\n", "\n...\n").Goto("block_start"),
			Match(FrontMatterMarker, "\n---", "\n...").Goto("block_start"),
			Fallback(RawFrontMatter),
		}},
		{Name: "block_start", Rules: []Rule{
			Match(Text, "\n", "\r"),
			Match(CodeBlock, "    ", "\t").Enter("indented_code"),
			Match(CodeFence, "``````", "`````", "````", "```").Goto("fence_info_btick"),
			Match(CodeFence, "~~~~~~", "~~~~~", "~~~~", "~~~").Goto("fence_info_tilde"),
			Match(HeadingMarker, "######", "#####", "####", "###", "##", "#").Goto("heading_space"),
			Match(ThematicBreak, "---", "***", "___").Goto("thematic_break_tail"),
			Match(BlockquoteMarker, ">"),
			Match(ListMarker, "- ", "* ", "+ ").Goto("list_body_probe"),
			Match(Text, " ", "\t"),
			Fallback(Text).Goto("inline_content"),
		}},
		{Name: "heading_space", Rules: []Rule{
			Match(Text, "\n").Goto("block_start"),
			Match(Text, " ", "\t").Goto("heading_body"),
			Fallback(Text).Goto("inline_content"),
		}},
		inline("heading_body", Fallback(Heading)),
		inline("inline_content", Fallback(Text)),
		body("bold_star_body", boldClose, "**", Bold),
		body("bold_under_body", boldClose, "__", Bold),
		body("italic_star_body", italicClose, "*", Italic),
		body("italic_under_body", italicClose, "_", Italic),
		body("strike_body", strikeClose, "~~", Strike),
		{Name: "code_body", Rules: []Rule{Match(codeClose, "`").Leave(), Match(Text, "\n").Leave(), Fallback(Code)}},
		{Name: "link_text", Rules: slices.Concat([]Rule{
			Match(linkTextClose, "]").Goto("link_after_close"),
			Match(Text, "\n").Leave(),
		}, nested, links[2:], emphasis(""), []Rule{Fallback(LinkText)})},
		{Name: "link_after_close", Rules: []Rule{
			Match(URLLink, "(").Goto("link_destination"),
			Match(URLLink, "[").Goto("link_reference_label"),
			Fallback(Text).Leave(),
		}},
		{Name: "link_destination", Rules: []Rule{
			Match(URLLink, ")").Leave(),
			Match(Text, "\n").Leave(),
			Within(URLTitle, `"`, `"`).Escaped(`\`).OneLine(),
			Within(URLTitle, "'", "'").Escaped(`\`).OneLine(),
			Fallback(URL),
		}},
		{Name: "link_reference_label", Rules: []Rule{Match(URLLink, "]").Leave(), Match(Text, "\n").Leave(), Fallback(Property)}},
		{Name: "autolink_body", Rules: []Rule{Match(autolinkClose, ">").Leave(), Match(Text, "\n").Leave(), Fallback(Autolink)}},
		{Name: "entity_body", Rules: []Rule{
			Match(Entity, ";").Leave(),
			Match(Entity, "#", "x").Letters().Digits(),
			Fallback(Text).Leave(),
		}},
		{Name: "list_body_probe", Rules: []Rule{
			Match(TaskMarker, "[ ] ", "[x] ", "[X] ").Goto("inline_content"),
			Fallback(Text).Goto("inline_content"),
		}},
		{Name: "indented_code", Rules: []Rule{Match(CodeBlock, "\n").Leave(), Fallback(CodeBlock)}},
		{Name: "thematic_break_tail", Rules: []Rule{
			Match(ThematicBreak, "\n").Goto("block_start"),
			Match(ThematicBreak, "-", "*", "_", " ", "\t"),
			Fallback(Text).Goto("inline_content"),
		}},
		{Name: "fence_close_tail", Rules: []Rule{Match(CodeFence, "\n").Goto("block_start"), Fallback(CodeFence)}},
	}, fence("btick", "`"), fence("tilde", "~"))
	return mustCompile(states, composeStyles)
}

func composeStyles(w *work) {
	styles := [...]Kind{Bold, Italic, Strike, Code, LinkText, Autolink}
	var stack [len(styles)]Kind
	depth, lastEnd := 0, 0
	set := func(n int) (Style, Kind) {
		var s Style
		for _, k := range stack[:n] {
			s |= 1 << slices.Index(styles[:], k)
		}
		if n == 0 {
			return 0, Text
		}
		return s, stack[n-1]
	}
	for i, sp := range w.spans {
		if depth > 0 && strings.IndexByte(w.src[lastEnd:sp.Start], '\n') >= 0 {
			depth = 0
		}
		lastEnd = sp.End
		if sp.Kind < boldOpen || sp.Kind > autolinkClose {
			w.spans[i].Style, _ = set(depth)
			continue
		}
		style, open := styles[(sp.Kind-boldOpen)/2], (sp.Kind-boldOpen)%2 == 0
		at := slices.Index(stack[:depth], style)
		if open && at < 0 {
			stack[depth] = style
			depth++
		}
		w.spans[i].Style, w.spans[i].Kind = set(depth)
		if at >= 0 {
			depth = at
		}
	}
}
