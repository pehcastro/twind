package highlight

import (
	"slices"
	"strconv"
)

func mustCompile(states []State, reclassify func(*work)) *Grammar {
	g, err := Compile(Definition{States: append(states, escapeDigits()...)})
	if err != nil {
		panic(err)
	}
	g.reclassify = reclassify
	return g
}

func digits(name string, count int, digit Rule) []State {
	chain := make([]State, count)
	for i := range chain {
		next := digit.Leave()
		if i+1 < count {
			next = digit.Goto(name + strconv.Itoa(i+2))
		}
		chain[i] = State{Name: name + strconv.Itoa(i+1), Rules: []Rule{next, Fallback(Text).Leave()}}
	}
	return chain
}

func escapeDigits() []State {
	hex, octal := Match(Escape).HexDigits(), Match(Escape).Range('0', '7')
	return slices.Concat(digits("x", 2, hex), digits("u", 4, hex), digits("U", 8, hex), digits("octal", 2, octal))
}

func Go() *Grammar {
	done := Fallback(Text).Leave()
	escape := Match(Escape, `\`).Enter("escape")
	states := []State{
		{Name: "main", Rules: []Rule{
			Match(Text, " ", "\t", "\n", "\r"),
			Match(Comment, "//").Enter("line_comment"),
			Match(Comment, "/*").Enter("block_comment"),
			Match(String, `"`).Enter("string_body"),
			Match(String, "`").Enter("raw_string_body"),
			Match(String, "'").Enter("rune_body"),
			Match(Number, ".0", ".1", ".2", ".3", ".4", ".5", ".6", ".7", ".8", ".9").Enter("decimal_fraction"),
			Match(Operator, "<<=", ">>=", "&^=", "...", "&^", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<", ">>",
				"&&", "||", "==", "!=", "<=", ">=", "<-", "++", "--", ":=", "+", "-", "*", "/", "%", "&", "|", "^", "<", ">", "=", "!", "~"),
			Match(Punctuation, "(", ")", "[", "]", "{", "}", ",", ";", ":", "."),
			Word(Keyword, "break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for", "func",
				"go", "goto", "if", "import", "interface", "map", "package", "range", "return", "select", "struct", "switch", "type", "var"),
			Word(Boolean, "true", "false"),
			Word(Keyword, "nil", "iota", "any", "bool", "byte", "comparable", "complex64", "complex128", "error", "float32", "float64",
				"int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr"),
			Word(Function, "append", "cap", "clear", "close", "complex", "copy", "delete", "imag", "len", "make", "max", "min",
				"new", "panic", "print", "println", "real", "recover"),
			Match(Number, "0x", "0X").Enter("hex_number"),
			Match(Number, "0b", "0B").Enter("binary_number"),
			Match(Number, "0o", "0O").Enter("octal_number"),
			Match(Number).Digits().Enter("decimal_number"),
			Match(Identifier, "_").Letters().Enter("identifier"),
		}},
		{Name: "identifier", Rules: []Rule{Match(Identifier, "_").Letters().Digits(), done}},
		{Name: "line_comment", Rules: []Rule{Match(Comment, "\n").Leave(), Fallback(Comment)}},
		{Name: "block_comment", Rules: []Rule{Match(Comment, "*/").Leave(), Fallback(Comment)}},
		{Name: "string_body", Rules: []Rule{escape, Match(String, `"`).Leave(), Fallback(String)}},
		{Name: "raw_string_body", Rules: []Rule{Match(String, "`").Leave(), Fallback(String)}},
		{Name: "rune_body", Rules: []Rule{escape, Match(String, "'").Leave(), Fallback(String)}},
		{Name: "escape", Rules: []Rule{
			Match(Escape, "x").Goto("x1"),
			Match(Escape, "u").Goto("u1"),
			Match(Escape, "U").Goto("U1"),
			Match(Escape).Range('0', '7').Goto("octal1"),
			Fallback(Escape).Leave(),
		}},
		{Name: "decimal_number", Rules: []Rule{
			Match(Number, "_").Digits(),
			Match(Number, ".").Goto("decimal_fraction"),
			Match(Number, "e", "E").Goto("exponent_sign"),
			Match(Number, "i").Leave(),
			done,
		}},
		{Name: "decimal_fraction", Rules: []Rule{
			Match(Number, "_").Digits(),
			Match(Number, "e", "E").Goto("exponent_sign"),
			Match(Number, "i").Leave(),
			done,
		}},
		{Name: "exponent_sign", Rules: []Rule{
			Match(Number, "+", "-").Goto("exponent_digits"),
			Match(Number).Digits().Goto("exponent_digits"),
			done,
		}},
		{Name: "exponent_digits", Rules: []Rule{Match(Number, "_").Digits(), Match(Number, "i").Leave(), done}},
		{Name: "hex_number", Rules: []Rule{
			Match(Number, "_").HexDigits(),
			Match(Number, ".").Goto("hex_fraction"),
			Match(Number, "p", "P").Goto("exponent_sign"),
			Match(Number, "i").Leave(),
			done,
		}},
		{Name: "hex_fraction", Rules: []Rule{
			Match(Number, "_").HexDigits(),
			Match(Number, "p", "P").Goto("exponent_sign"),
			Match(Number, "i").Leave(),
			done,
		}},
		{Name: "binary_number", Rules: []Rule{Match(Number, "_", "0", "1"), Match(Number, "i").Leave(), done}},
		{Name: "octal_number", Rules: []Rule{Match(Number, "_").Range('0', '7'), Match(Number, "i").Leave(), done}},
	}
	return mustCompile(states, goPasses)
}

func JSON() *Grammar {
	done := Fallback(Text).Leave()
	states := []State{
		{Name: "main", Rules: []Rule{
			Match(Text, " ", "\t", "\n", "\r"),
			Match(String, `"`).Enter("string_body"),
			Word(Boolean, "true", "false"),
			Word(Keyword, "null"),
			Match(Punctuation, "{", "}", "[", "]", ",", ":"),
			Match(Number, "-").Enter("negative_number"),
			Match(Number).Digits().Enter("number"),
		}},
		{Name: "negative_number", Rules: []Rule{Match(Number).Digits().Goto("number"), done}},
		{Name: "number", Rules: []Rule{
			Match(Number).Digits(),
			Match(Number, ".").Goto("decimal"),
			Match(Number, "e", "E").Goto("exponent_sign"),
			done,
		}},
		{Name: "decimal", Rules: []Rule{Match(Number).Digits(), Match(Number, "e", "E").Goto("exponent_sign"), done}},
		{Name: "exponent_sign", Rules: []Rule{
			Match(Number, "+", "-").Goto("exponent_digits"),
			Match(Number).Digits().Goto("exponent_digits"),
			done,
		}},
		{Name: "exponent_digits", Rules: []Rule{Match(Number).Digits(), done}},
		{Name: "string_body", Rules: []Rule{
			Match(Escape, `\`).Enter("escape"),
			Match(String, `"`).Leave(),
			Fallback(String),
		}},
		{Name: "escape", Rules: []Rule{Match(Escape, "u").Goto("u1"), Fallback(Escape).Leave()}},
	}
	return mustCompile(states, nil)
}

func TOML() *Grammar {
	done := Fallback(Text).Leave()
	space := Match(Text, " ", "\t")
	newline := Match(Text, "\n", "\r")
	comment := Within(Comment, "#", "\n").OneLine()
	basicKey := Match(Property, `"`).Enter("basic_key_body")
	literalKey := Within(Property, "'", "'")
	bareKey := Match(Property, "-", "_").Letters().Digits()
	quotedKeys := []Rule{Match(Property, `"""`).Enter("ml_basic_key_body"), Within(Property, "'''", "'''"), basicKey, literalKey, bareKey}
	values := func(lead ...Rule) []Rule {
		return append(lead,
			Match(String, `"""`).Enter("ml_basic_string_body"),
			Within(String, "'''", "'''"),
			Match(String, `"`).Enter("basic_string_body"),
			Within(String, "'", "'"),
			Word(Boolean, "true", "false"),
			Word(Keyword, "inf", "nan"),
			Match(Operator, "+", "-").Enter("value_signed"),
			Match(Text).Digits().Enter("digit_probe1"),
			Match(Punctuation, "[").Enter("value_array"),
			Match(Punctuation, "{").Enter("value_inline_table"),
		)
	}
	body := func(kind Kind, close string) []Rule {
		return []Rule{
			Match(Escape, `\u`).Enter("u1"),
			Match(Escape, `\U`).Enter("U1"),
			Match(Escape, `\`).Enter("escape"),
			Match(kind, close).Leave(),
			Fallback(kind),
		}
	}
	toNumber := Fallback(Text).Enter("number_from_probe")
	states := []State{
		{Name: "root", Rules: slices.Concat([]Rule{space, newline, comment, Match(Text, "[").Enter("bracket_probe")},
			quotedKeys, []Rule{Match(Operator, "=").Goto("value"), Match(Punctuation, ".")})},
		{Name: "bracket_probe", Probe: true, AtEnd: "table_header", Rules: []Rule{
			Match(Text, "[").Enter("array_table_header"),
			Fallback(Text).Enter("table_header"),
		}},
		{Name: "table_header", Rules: []Rule{Match(Punctuation, "[").Goto("table_header_key")}},
		{Name: "table_header_key", Rules: []Rule{
			space, comment, basicKey, literalKey, bareKey,
			Match(Punctuation, "."),
			Match(Punctuation, "]").Leave(),
		}},
		{Name: "array_table_header", Rules: []Rule{Match(TableHeader, "[[").Goto("array_table_header_key")}},
		{Name: "array_table_header_key", Rules: []Rule{
			space, comment, basicKey, literalKey, bareKey,
			Match(Punctuation, "."),
			Match(TableHeader, "]]").Leave(),
		}},
		{Name: "value", Rules: append(values(Match(Text, "\n", "\r").Goto("root"), space, comment), Fallback(Text).Goto("root"))},
		{Name: "value_array", Rules: append(values(space, newline, comment), Match(Punctuation, ","), Match(Punctuation, "]").Leave())},
		{Name: "value_inline_table", Rules: slices.Concat([]Rule{space}, quotedKeys, []Rule{
			Match(Operator, "=").Goto("value_inline_table_after_val"),
			Match(Punctuation, ",", "."),
			Match(Punctuation, "}").Leave(),
		})},
		{Name: "value_inline_table_after_val", Rules: append(values(space), Match(Text, ",", "}").Goto("value_inline_table"))},
		{Name: "basic_string_body", Rules: body(String, `"`)},
		{Name: "ml_basic_string_body", Rules: body(String, `"""`)},
		{Name: "basic_key_body", Rules: body(Property, `"`)},
		{Name: "ml_basic_key_body", Rules: body(Property, `"""`)},
		{Name: "escape", Rules: []Rule{Fallback(Escape).Leave()}},
		{Name: "value_number", Rules: []Rule{
			Match(Number, "_").Digits(),
			Match(Number, "x", "X").Goto("value_hex"),
			Match(Number, "o", "O").Goto("value_octal"),
			Match(Number, "b", "B").Goto("value_binary"),
			Match(Number, ".").Goto("value_decimal"),
			Match(Number, "e", "E").Goto("value_exponent_sign"),
			done,
		}},
		{Name: "value_hex", Rules: []Rule{Match(Number, "_").HexDigits(), done}},
		{Name: "value_octal", Rules: []Rule{Match(Number, "_").Range('0', '7'), done}},
		{Name: "value_binary", Rules: []Rule{Match(Number, "_", "0", "1"), done}},
		{Name: "value_decimal", Rules: []Rule{Match(Number, "_").Digits(), Match(Number, "e", "E").Goto("value_exponent_sign"), done}},
		{Name: "value_exponent_sign", Rules: []Rule{
			Match(Number, "+", "-").Goto("value_exponent_digits"),
			Match(Number).Digits().Goto("value_exponent_digits"),
			done,
		}},
		{Name: "value_exponent_digits", Rules: []Rule{Match(Number, "_").Digits(), done}},
		{Name: "value_signed", Rules: []Rule{Word(Keyword, "inf", "nan").Leave(), Match(Number).Digits().Goto("value_number"), done}},
		{Name: "digit_probe1", Probe: true, AtEnd: "number_from_probe", Rules: []Rule{Match(Text).Digits().Enter("digit_probe2"), toNumber}},
		{Name: "digit_probe2", Probe: true, AtEnd: "number_from_probe", Rules: []Rule{
			Match(Text).Digits().Enter("digit_probe3"),
			Match(Text, ":").Enter("local_time_from_probe"),
			toNumber,
		}},
		{Name: "digit_probe3", Probe: true, AtEnd: "number_from_probe", Rules: []Rule{Match(Text).Digits().Enter("digit_probe4"), toNumber}},
		{Name: "digit_probe4", Probe: true, AtEnd: "number_from_probe", Rules: []Rule{
			Match(Text, "-").Enter("datetime_from_probe"),
			Match(Text, ".").Enter("number_from_probe"),
			toNumber,
		}},
		{Name: "number_from_probe", Rules: []Rule{Match(Number).Digits().Goto("value_number")}},
		{Name: "local_time_from_probe", Rules: []Rule{Match(Datetime).Digits().Goto("local_time_body")}},
		{Name: "local_time_body", Rules: []Rule{Match(Datetime).Digits(), Match(Punctuation, ":", "."), done}},
		{Name: "datetime_from_probe", Rules: []Rule{Match(Datetime).Digits().Goto("year")}},
		{Name: "year", Rules: []Rule{Match(Datetime).Digits(), Match(Punctuation, "-").Goto("month"), done}},
		{Name: "month", Rules: []Rule{Match(Datetime).Digits(), Match(Punctuation, "-").Goto("day"), done}},
		{Name: "day", Rules: []Rule{Match(Datetime).Digits(), Match(Datetime, "T", "t", " ").Goto("time"), done}},
		{Name: "time", Rules: []Rule{
			Match(Datetime).Digits(),
			Match(Punctuation, ":", "."),
			Match(Datetime, "Z", "z").Goto("zone"),
			Match(Punctuation, "+", "-").Goto("offset"),
			done,
		}},
		{Name: "zone", Rules: []Rule{done}},
		{Name: "offset", Rules: []Rule{Match(Datetime).Digits(), Match(Punctuation, ":"), done}},
	}
	return mustCompile(states, nil)
}

func Bash() *Grammar {
	space := Match(Text, " ", "\t", "\n", "\r")
	continuation := Match(Operator, "\\\n")
	single := Within(String, "'", "'")
	backtick := Within(String, "`", "`").Escaped(`\`)
	variables := []string{"$@", "$*", "$#", "$?", "$-", "$$", "$!", "$_", "$0", "$1", "$2", "$3", "$4", "$5", "$6", "$7", "$8", "$9"}
	for c := 'a'; c <= 'z'; c++ {
		variables = append(variables, "$"+string(c), "$"+string(c-'a'+'A'))
	}
	expansions := []Rule{
		Match(String, "$'").Enter("ansi_string"),
		Match(String, `$"`).Enter("double_string"),
		Match(Punctuation, "$((").Enter("arith"),
		Match(Punctuation, "$(").Enter("cmd_sub"),
		Match(Punctuation, "${").Enter("param_exp"),
		Match(Variable, variables...),
		Match(Operator, "$"),
	}
	quoted := slices.Concat(expansions, []Rule{Match(String, `"`).Enter("double_string"), single, backtick})
	command := slices.Concat([]Rule{space, continuation, Within(Comment, "#", "\n")}, quoted, []Rule{
		Match(Punctuation, "((").Enter("arith"),
		Match(Punctuation, "<(", ">(", "(").Enter("cmd_sub"),
		Word(Keyword, "[[").Enter("conditional"),
		Match(Operator, "<<<", "<<-", "&>>", "<<", ">>", ">&", "<&", "<>", ">|", "&>", "<", ">"),
		Match(Punctuation, ";;&", ";;", ";&", "&&", "||", "|&", ";", "&", "|"),
		Match(Operator, "+=", "="),
		Word(Operator, "!"),
		Word(Builtin, ":"),
		Match(Punctuation, "{", "}", "[", "]", ",", ".", "~", "*", "?"),
		Match(Identifier, "_").Letters().Digits(),
		Match(Operator, "-", "/"),
	})
	arith := slices.Concat([]Rule{space, continuation}, quoted, []Rule{
		Match(Punctuation, "(").Enter("arith_group"),
		Match(Operator, "**=", "<<=", ">>=", "**", "<<", ">>", "<=", ">=", "==", "!=", "&&", "||", "++", "--", "+=", "-=", "*=",
			"/=", "%=", "&=", "|=", "^=", "+", "-", "*", "/", "%", "&", "|", "^", "~", "!", "<", ">", "=", "?", ":", ","),
		Match(Number, "0x", "0X").Digits(),
		Match(Punctuation, "#", "."),
		Match(Identifier, "_").Letters(),
		Fallback(Identifier),
	})
	states := []State{
		{Name: "main", Rules: command},
		{Name: "cmd_sub", Rules: slices.Concat([]Rule{Match(Punctuation, ")").Leave()}, command)},
		{Name: "arith", Rules: slices.Concat([]Rule{Match(Punctuation, "))").Leave()}, arith)},
		{Name: "arith_group", Rules: slices.Concat([]Rule{Match(Punctuation, ")").Leave()}, arith)},
		{Name: "conditional", Rules: slices.Concat([]Rule{Word(Keyword, "]]").Leave(), space, continuation}, quoted, []Rule{
			Match(Operator, "=~").Enter("regex_start"),
			Match(Operator, "==", "!=", "<=", ">=", "=", "<", ">", "-ef", "-nt", "-ot", "-eq", "-ne", "-lt", "-le", "-gt", "-ge",
				"-a", "-b", "-c", "-d", "-e", "-f", "-g", "-h", "-k", "-n", "-o", "-p", "-r", "-s", "-t", "-u", "-v", "-w", "-x", "-z",
				"-G", "-L", "-N", "-O", "-R", "-S", "&&", "||", "!"),
			Match(Punctuation, "(", ")"),
			Match(Identifier, "_").Letters().Digits(),
			Match(Operator, "-", "/", "*", "?", ":", ".", ",", "~", "+"),
			Fallback(Identifier),
		})},
		{Name: "regex_start", Rules: []Rule{Match(Text, " ", "\t"), Fallback(Text).Goto("regex")}},
		{Name: "regex", Rules: slices.Concat([]Rule{
			Match(Text, " ", "\t", "\n", "\r").Leave(),
			Match(String, `"`).Enter("double_string"),
			single,
		}, expansions, []Rule{Fallback(Regex)})},
		{Name: "double_string", Rules: slices.Concat([]Rule{
			Match(String, `"`).Leave(),
			Match(Escape, `\$`, "\\`", `\"`, `\\`, "\\\n"),
		}, expansions, []Rule{backtick, Fallback(String)})},
		{Name: "ansi_string", Rules: []Rule{Match(String, "'").Leave(), Match(Escape, `\`).Enter("ansi_escape"), Fallback(String)}},
		{Name: "ansi_escape", Rules: []Rule{
			Match(Escape, "x").Goto("x1"),
			Match(Escape, "u").Goto("u1"),
			Match(Escape, "U").Goto("U1"),
			Match(Escape, "c").Goto("control"),
			Match(Escape).Range('0', '7').Goto("octal1"),
			Fallback(Escape).Leave(),
		}},
		{Name: "control", Rules: []Rule{Fallback(Escape).Leave()}},
		{Name: "param_exp", Rules: slices.Concat([]Rule{Match(Punctuation, "}").Leave(), space, continuation}, quoted, []Rule{
			Match(Operator, "@U", "@u", "@L", "@Q", "@E", "@P", "@A", "@a", "@K", "@k", ":-", ":=", ":?", ":+", "##", "%%", "//",
				"/#", "/%", "^^", ",,", "#", "%", "/", "^", ",", "!", "-", "=", "?", "+", ":", "@"),
			Match(Punctuation, "[", "]", "*", "?"),
			Match(Identifier, "_").Letters().Digits(),
			Fallback(Identifier),
		})},
	}
	words := map[string]Kind{"true": Boolean, "false": Boolean}
	for _, w := range []string{"cd", "echo", "exec", "exit", "export", "eval", "getopts", "hash", "printf", "pwd", "read", "readonly",
		"return", "set", "shift", "test", "times", "trap", "unset", "break", "continue", "umask", "wait", "alias", "bind", "builtin",
		"caller", "command", "declare", "disown", "enable", "help", "history", "jobs", "kill", "let", "local", "logout", "mapfile",
		"popd", "pushd", "readarray", "shopt", "source", "suspend", "type", "typeset", "ulimit", "unalias"} {
		words[w] = Builtin
	}
	for _, w := range []string{"if", "then", "elif", "else", "fi", "time", "for", "in", "until", "while", "do", "done", "case",
		"esac", "coproc", "select", "function"} {
		words[w] = Keyword
	}
	return mustCompile(states, bashPasses(words))
}
