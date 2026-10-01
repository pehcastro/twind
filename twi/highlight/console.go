package highlight

import (
	"slices"
	"strings"
	"unicode/utf8"
)

type chars struct {
	literals []string
	ranges   []runeRange
}

func (c chars) rule(kind Kind) Rule {
	r := Match(kind, c.literals...)
	for _, rg := range c.ranges {
		r = r.Range(rg.lo, rg.hi)
	}
	return r
}

func (c chars) wide() []runeRange {
	var out []runeRange
	for _, rg := range c.ranges {
		if rg.hi >= utf8.RuneSelf {
			out = append(out, runeRange{max(rg.lo, utf8.RuneSelf), rg.hi})
		}
	}
	return out
}

func wideExcept(claimed []runeRange) chars {
	slices.SortFunc(claimed, func(a, b runeRange) int { return int(a.lo - b.lo) })
	var out chars
	next := rune(utf8.RuneSelf)
	for _, rg := range claimed {
		if rg.lo > next {
			out.ranges = append(out.ranges, runeRange{next, rg.lo - 1})
		}
		next = max(next, rg.hi+1)
	}
	if next <= 0xffff {
		out.ranges = append(out.ranges, runeRange{next, 0xffff})
	}
	return out
}

type promptTarget string

const (
	promptStay   promptTarget = ""
	promptReject promptTarget = "promptReject"
)

type promptEdge struct {
	on chars
	to promptTarget
}

type promptNode struct {
	name    string
	accepts *chars
	strict  bool
	words   [][]string
	wordTo  []string
	edges   []promptEdge
	other   promptTarget
}

func Console() *Grammar {
	ws := chars{literals: []string{" ", "\t"}, ranges: []runeRange{{0xa0, 0xa0}}}
	eol := chars{literals: []string{"\n", "\r"}}
	wsEOL := chars{literals: slices.Concat(ws.literals, eol.literals), ranges: ws.ranges}
	symbols := chars{literals: []string{"$", "#", "%", ">"}, ranges: []runeRange{{0x276f, 0x276f}, {0x279c, 0x279c}}}
	for _, g := range [][2]rune{
		{0x0370, 0x0373}, {0x0375, 0x0377}, {0x037a, 0x037d}, {0x037f, 0x037f}, {0x0384, 0x0384}, {0x0386, 0x0386},
		{0x0388, 0x038a}, {0x038c, 0x038c}, {0x038e, 0x03a1}, {0x03a3, 0x03e1}, {0x03f0, 0x03ff}, {0x1d26, 0x1d2a},
		{0x1d5d, 0x1d61}, {0x1d66, 0x1d6a}, {0x1dbf, 0x1dbf}, {0x1f00, 0x1f15}, {0x1f18, 0x1f1d}, {0x1f20, 0x1f45},
		{0x1f48, 0x1f4d}, {0x1f50, 0x1f57}, {0x1f59, 0x1f59}, {0x1f5b, 0x1f5b}, {0x1f5d, 0x1f5d}, {0x1f5f, 0x1f7d},
		{0x1f80, 0x1fb4}, {0x1fb6, 0x1fc4}, {0x1fc6, 0x1fd3}, {0x1fd6, 0x1fdb}, {0x1fdd, 0x1fef}, {0x1ff2, 0x1ff4},
		{0x1ff6, 0x1ffe}, {0x2126, 0x2126}, {0xab65, 0xab65},
	} {
		symbols.ranges = append(symbols.ranges, runeRange{g[0], g[1]})
	}
	sep := chars{literals: []string{"@", ":"}}
	wordStart := chars{literals: []string{"_"}, ranges: []runeRange{{'a', 'z'}, {'A', 'Z'}, {'0', '9'}}}
	one := func(s string) chars { return chars{literals: []string{s}} }
	shells := []string{"sh", "bash", "zsh", "ksh", "mksh", "dash", "ash", "fish", "csh", "tcsh"}
	zsh := strings.Fields("for while repeat select until if then else elif math cond cmdor cmdand pipe errpipe foreach case " +
		"function subsh cursh array quote dquote bquote cmdsubst mathsubst elif-then heredoc heredocd brace braceparam always")
	nodes := []promptNode{
		{name: "start", accepts: &symbols, strict: true, words: [][]string{shells, zsh}, wordTo: []string{"shell_name", "zsh_word"},
			edges: []promptEdge{{one("("), "venv_open"}, {one("["), "bracket_open"}, {wordStart, "user_first"}}, other: promptReject},
		{name: "venv_open", edges: []promptEdge{{ws, promptReject}}, other: "venv_body"},
		{name: "venv_body", edges: []promptEdge{{one(")"), "venv_closed"}, {ws, promptReject}}, other: promptStay},
		{name: "venv_closed", accepts: &symbols, edges: []promptEdge{{one(")"), promptStay}, {ws, "venv_ws"}}, other: "venv_body"},
		{name: "venv_ws", accepts: &symbols, strict: true, words: [][]string{shells}, wordTo: []string{"shell_name"},
			edges: []promptEdge{{ws, promptStay}, {one("["), "bracket_open"}, {wordStart, "user_first"}}, other: promptReject},
		{name: "shell_name", accepts: &symbols,
			edges: []promptEdge{{one("-"), "shell_version"}, {ws, "word1_ws_solo"}, {sep, "user_sep"}}, other: "user_body"},
		{name: "shell_version", accepts: &symbols, edges: []promptEdge{{ws, "word1_ws_solo"}}, other: promptStay},
		{name: "zsh_word", accepts: &chars{literals: []string{">"}}, edges: []promptEdge{{ws, "zsh_ws"}, {sep, "user_sep"}}, other: "user_body"},
		{name: "zsh_ws", words: [][]string{zsh}, wordTo: []string{"zsh_word"}, edges: []promptEdge{{ws, promptStay}}, other: promptReject},
		{name: "user_first", edges: []promptEdge{{ws, promptReject}}, other: "user_body"},
		{name: "user_body", edges: []promptEdge{{sep, "user_sep"}, {ws, promptReject}}, other: promptStay},
		{name: "user_sep", edges: []promptEdge{{ws, promptReject}}, other: "user_host"},
		{name: "user_host", accepts: &symbols, edges: []promptEdge{{ws, "word1_ws"}}, other: promptStay},
		{name: "word1_ws", accepts: &symbols, edges: []promptEdge{{ws, promptStay}}, other: "word2"},
		{name: "word2", accepts: &symbols, edges: []promptEdge{{ws, "word2_ws"}}, other: promptStay},
		{name: "word2_ws", accepts: &symbols, strict: true, edges: []promptEdge{{ws, promptStay}}, other: promptReject},
		{name: "word1_ws_solo", accepts: &symbols, strict: true, edges: []promptEdge{{ws, promptStay}}, other: promptReject},
		{name: "bracket_open", edges: []promptEdge{{ws, promptReject}}, other: "bracket_head"},
		{name: "bracket_head", edges: []promptEdge{{sep, "bracket_sep"}, {ws, promptReject}}, other: promptStay},
		{name: "bracket_sep", other: "bracket_body"},
		{name: "bracket_body", edges: []promptEdge{{one("]"), "bracket_closed"}}, other: promptStay},
		{name: "bracket_closed", accepts: &symbols, other: promptStay},
	}
	notWSEOL := wideExcept(wsEOL.wide())
	states := []State{
		{Name: "line_start", Rules: []Rule{
			eol.rule(Text),
			wideExcept(nil).rule(Text).Goto("probe_start"),
			Fallback(Text).Goto("probe_start"),
		}},
		{Name: "output_line", Rules: []Rule{Match(Text, "\n").Goto("line_start"), Match(Text, "\r"), Fallback(Output)}},
		{Name: "probe_symbol_check", Probe: true, AtEnd: "prefix_start", Rules: []Rule{
			wsEOL.rule(Text).Goto("prefix_start"),
			notWSEOL.rule(Text).Leave(),
			Fallback(Text).Leave(),
		}},
		{Name: "probe_symbol_check_strict", Probe: true, AtEnd: "prefix_start", Rules: []Rule{
			wsEOL.rule(Text).Goto("prefix_start"),
			notWSEOL.rule(Text).Goto("output_line"),
			Fallback(Text).Goto("output_line"),
		}},
		{Name: "prefix_symbol_check", Probe: true, AtEnd: "prompt_symbol", Rules: []Rule{
			wsEOL.rule(Text).Goto("prompt_symbol"),
			notWSEOL.rule(Text).Enter("prompt_text_symbol"),
			Fallback(Text).Enter("prompt_text_symbol"),
		}},
		{Name: "prompt_symbol", Rules: []Rule{symbols.rule(Prompt).Goto("gap"), Fallback(Text).Goto("gap")}},
		{Name: "prompt_text_symbol", Rules: []Rule{symbols.rule(PromptPrefix).Leave(), Fallback(Text).Leave()}},
		{Name: "gap", Rules: []Rule{
			chars{literals: []string{" ", "\t", "\r"}, ranges: ws.ranges}.rule(Text),
			Match(Text, "\n").Goto("line_start"),
			notWSEOL.rule(Text).Goto("command"),
			Fallback(Text).Goto("command"),
		}},
		{Name: "command", Rules: []Rule{Match(Text, "\n").Goto("line_start"), Match(Text, "\r"), Fallback(rawShell)}},
	}
	for _, n := range nodes {
		states = append(states, probeNode(n, eol), prefixNode(n, eol))
	}
	g, err := Compile(Definition{States: states})
	if err != nil {
		panic(err)
	}
	bash := Bash()
	g.reclassify = func(w *work) { w.embedCommands(bash, zsh) }
	return g
}

func probeNode(n promptNode, eol chars) State {
	var rules []Rule
	var claimed []runeRange
	add := func(c chars, r Rule) {
		rules = append(rules, r)
		claimed = append(claimed, c.wide()...)
	}
	move := func(t promptTarget, r Rule) Rule {
		switch t {
		case promptReject:
			return r.Goto("output_line")
		case promptStay:
			return r
		}
		return r.Enter("probe_" + string(t))
	}
	if n.accepts != nil {
		check := "probe_symbol_check"
		if n.strict {
			check += "_strict"
		}
		add(*n.accepts, n.accepts.rule(Text).Enter(check))
	}
	add(eol, eol.rule(Text).Goto("output_line"))
	for i, words := range n.words {
		rules = append(rules, Match(Text, words...).Goto("probe_"+n.wordTo[i]))
	}
	for _, e := range n.edges {
		add(e.on, move(e.to, e.on.rule(Text)))
	}
	if n.other != promptStay {
		rules = append(rules, move(n.other, wideExcept(claimed).rule(Text)), move(n.other, Fallback(Text)))
	}
	return State{Name: "probe_" + n.name, Probe: true, AtEnd: "output_line", Rules: rules}
}

func prefixNode(n promptNode, eol chars) State {
	var rules []Rule
	var claimed []runeRange
	add := func(c chars, r Rule) {
		rules = append(rules, r)
		claimed = append(claimed, c.wide()...)
	}
	if n.accepts != nil {
		add(*n.accepts, n.accepts.rule(Text).Enter("prefix_symbol_check"))
	}
	add(eol, eol.rule(Text).Goto("output_line"))
	for i, words := range n.words {
		rules = append(rules, Match(PromptPrefix, words...).Goto("prefix_"+n.wordTo[i]))
	}
	for _, e := range n.edges {
		switch e.to {
		case promptReject:
			add(e.on, e.on.rule(Text).Goto("output_line"))
		case promptStay:
			add(e.on, e.on.rule(PromptPrefix))
		default:
			add(e.on, e.on.rule(PromptPrefix).Goto("prefix_"+string(e.to)))
		}
	}
	rest := wideExcept(claimed)
	switch n.other {
	case promptStay:
		rules = append(rules, rest.rule(PromptPrefix), Fallback(PromptPrefix))
	case promptReject:
		rules = append(rules, rest.rule(Text).Goto("output_line"), Fallback(Text).Goto("output_line"))
	default:
		next := "prefix_" + string(n.other)
		rules = append(rules, rest.rule(PromptPrefix).Goto(next), Fallback(PromptPrefix).Goto(next))
	}
	return State{Name: "prefix_" + n.name, Rules: rules}
}

func (w *work) embedCommands(bash *Grammar, zsh []string) {
	w.out = w.out[:0]
	for i := 0; i < len(w.spans); {
		if w.spans[i].Kind != rawShell {
			w.out = append(w.out, w.spans[i])
			i++
			continue
		}
		i = w.embedCommand(bash, zsh, i)
	}
	w.spans, w.out = w.out, w.spans
}

func (w *work) embedCommand(bash *Grammar, zsh []string, i int) int {
	sp, src := w.spans, w.src
	lineEnd := func(from int) int {
		if at := strings.IndexByte(src[from:], '\n'); at >= 0 {
			return from + at
		}
		return len(src)
	}
	zshPrefix := func(text string) bool {
		for word := range strings.SplitSeq(text, " ") {
			if !slices.Contains(zsh, word) {
				return false
			}
		}
		return true
	}
	k, eol := i, lineEnd(sp[i].Start)
	w.regions = append(w.regions[:0], region{start: sp[i].Start, end: min(eol+1, len(src))})
	for k < len(sp) && sp[k].Kind == rawShell && sp[k].Start < eol {
		k++
	}
	for eol < len(src) && k < len(sp) && sp[k].Start == eol+1 {
		prompt := k
		for prompt < len(sp) && sp[prompt].Kind == PromptPrefix {
			prompt++
		}
		if prompt > k && !zshPrefix(src[sp[k].Start:sp[prompt-1].End]) || !w.is(prompt, Prompt, ">") {
			break
		}
		next, content := eol+1, lineEnd(eol+1)
		eol = content
		if prompt+1 < len(sp) && sp[prompt+1].Kind == rawShell && sp[prompt+1].Start < eol {
			content = sp[prompt+1].Start
		}
		w.regions = append(w.regions, region{start: next, end: content, hole: true, from: k, to: prompt + 1})
		if content < len(src) {
			w.regions = append(w.regions, region{start: content, end: min(eol+1, len(src))})
		}
		k = prompt + 1
		for k < len(sp) && sp[k].Kind == rawShell && sp[k].Start < eol {
			k++
		}
	}
	w.splice(bash)
	return k
}
