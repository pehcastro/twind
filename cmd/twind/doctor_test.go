package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestDoctorParsesEachReplyKind(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want answers
	}{
		{"silent", "", answers{widths: []int{0, 0, 0}}},
		{"version ended by ST", "\x1bP>|WezTerm 20260929\x1b\\", answers{version: "WezTerm 20260929", widths: []int{0, 0, 0}}},
		{"version ended by BEL", "\x1bP>|xterm(390)\x07", answers{version: "xterm(390)", widths: []int{0, 0, 0}}},
		{"version holding escapes", "\x1bP>|a\x1b[31mb\rc\x1b\\", answers{version: "abc", widths: []int{0, 0, 0}}},
		{"a DECRQSS answer is not a name", "\x1bP1$r0;38:2::1:2:3m\x1b\\", answers{truecolor: true, widths: []int{0, 0, 0}}},
		{"truecolor with semicolons", "\x1bP1$r38;2;1;2;3m\x1b\\", answers{truecolor: true, widths: []int{0, 0, 0}}},
		{"truecolor refused", "\x1bP0$r\x1b\\", answers{widths: []int{0, 0, 0}}},
		{"truecolor dropped", "\x1bP1$r0m\x1b\\", answers{widths: []int{0, 0, 0}}},
		{"clipboard refused by xtgettcap", "\x1bP0+r\x1b\\", answers{widths: []int{0, 0, 0}}},
		{"clipboard from xtgettcap", "\x1bP1+r4D73=\x1b\\", answers{clipboard: true, widths: []int{0, 0, 0}}},
		{"clipboard from DA1", "\x1b[?61;4;52c", answers{primary: "61;4;52", clipboard: true, widths: []int{0, 0, 0}}},
		{"secondary attributes", "\x1b[>1;277;0c", answers{secondary: "1;277;0", widths: []int{0, 0, 0}}},
		{"secondary attributes too short", "\x1b[>0c", answers{widths: []int{0, 0, 0}}},
		{"modes", "\x1b[?1003;0$y\x1b[?1006;4$y\x1b[?2004;3$y\x1b[?9999;1$y", answers{paste: true, widths: []int{0, 0, 0}}},
		{"modes set and reset", "\x1b[?1003;1$y\x1b[?1006;2$y\x1b[?2004;2$y", answers{mouseAny: true, mouseSGR: true, paste: true, widths: []int{0, 0, 0}}},
		{"kitty keyboard flags", "\x1b[?1u", answers{keyboard: true, widths: []int{0, 0, 0}}},
		{"pointer ended by BEL", "\x1b]22;default\x07", answers{pointer: "default", widths: []int{0, 0, 0}}},
		{"pointer ended by ST", "\x1b]22;te\x1b[2Jxt\x1b\\", answers{pointer: "text", widths: []int{0, 0, 0}}},
		{"widths", "\x1b[5;1R\x1b[5;2R\x1b[5;3R\x1b[5;2R", answers{widths: []int{1, 2, 1}}},
		{"widths split and fenced", "\x1b[5;1R\x1b[5;2R\x1b[5;3R\x1b[5;2R\x1b[?62c", answers{primary: "62", widths: []int{1, 2, 1}}},
		{"fewer reports than glyphs", "\x1b[5;1R\x1b[5;2R\x1b[5;3R", answers{widths: []int{0, 0, 0}}},
		{"another row, no advance", "\x1b[5;1R\x1b[6;1R\x1b[5;1R\x1b[5;2R", answers{widths: []int{0, 0, 1}}},
		{"right margin clamps", "\x1b[5;78R\x1b[5;79R\x1b[5;80R\x1b[5;80R", answers{widths: []int{1, 0, 0}}},
		{"grid", "\x1b[8;36;120t", answers{grid: "120x36", widths: []int{0, 0, 0}}},
		{"zero grid", "\x1b[8;0;0t\x1b[4;612;840t", answers{widths: []int{0, 0, 0}}},
		{"kitty refuses zlib", "\x1b_Gi=31;ENOTSUP:compressed payloads are not supported\x1b\\\x1b_Gi=32;OK\x1b\\", answers{kittyZlib: "ENOTSUP:compressed payloads are not supported", kittyRaw: "OK", widths: []int{0, 0, 0}}},
		{"kitty status holding escapes", "\x1b_Gi=32;E\x1b[2JRR\x1b\\", answers{kittyRaw: "ERR", widths: []int{0, 0, 0}}},
		{"kitty answer for another id", "\x1b_Gi=7;OK\x1b\\", answers{widths: []int{0, 0, 0}}},
	}
	for _, tc := range cases {
		if got := parseAnswers([]byte(tc.raw), 3, 80); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestDoctorReportNamesUnknowns(t *testing.T) {
	out := answerLines(answers{widths: []int{0, 2}}, []string{"a", "b"})
	for _, line := range []string{"terminal   no answer\n", "secondary  no answer\n", "truecolor  false\n", "clipboard  false\n", "pointer    no answer\n", "grid       no answer\n", "kitty      raw no answer, zlib no answer\n", "glyphs     a ? b 2\n"} {
		if !strings.Contains(out, line) {
			t.Errorf("report %q lacks %q", out, line)
		}
	}
}
