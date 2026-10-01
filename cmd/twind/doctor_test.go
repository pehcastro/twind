package main

import (
	"reflect"
	"strings"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
)

func TestDoctorAsksConhostNothingItPrints(t *testing.T) {
	queries := []string{konst.VersionQuery, konst.SecondaryQuery, konst.KeyboardQuery, konst.DoctorModes, konst.TruecolorQuery, konst.ClipboardQuery, konst.PointerQuery, konst.KittyQuery, konst.KittyRawQuery, konst.GridQuery}
	cases := []struct {
		name  string
		da1   string
		asked bool
	}{
		{"conhost", "\x1b[?1;0c", false},
		{"silent", "", false},
		{"wezterm", "\x1b[?65;4;6;18;22;52c", true},
		{"kitty", "\x1b[?62;c", true},
	}
	for _, tc := range cases {
		var writes []string
		raw, skipped, err := doctorAsk(func(q string) ([]byte, error) {
			writes = append(writes, q)
			answer := tc.da1
			if strings.Contains(q, konst.KittyRawQuery) {
				answer = "\x1b_Gi=32;OK\x1b\\" + answer
			}
			if strings.HasPrefix(q, konst.ProbeBegin) {
				answer = "\x1b[5;1R\x1b[5;2R\x1b[5;4R" + answer
			}
			return []byte(answer), nil
		}, []string{"a", "b"})
		if err != nil {
			t.Fatal(err)
		}
		a := parseAnswers(raw, 2, 80)
		if !reflect.DeepEqual(a.widths, []int{1, 3}) {
			t.Errorf("%s: widths %v, want [1 3]", tc.name, a.widths)
		}
		if tc.asked != (a.kittyRaw == "OK") {
			t.Errorf("%s: kitty raw %q, asked %v", tc.name, a.kittyRaw, tc.asked)
		}
		all := strings.Join(writes, "")
		if leaked := strings.Contains(all, konst.DCS) || strings.Contains(all, "\x1b_"); leaked != tc.asked {
			t.Errorf("%s: APC or DCS written %v, want %v in %q", tc.name, leaked, tc.asked, all)
		}
		if skipped == tc.asked {
			t.Errorf("%s: skipped %v, asked %v", tc.name, skipped, tc.asked)
		}
		if !tc.asked {
			continue
		}
		sent := 0
		for _, q := range queries {
			if !strings.Contains(all, q) {
				t.Errorf("%s: %q never sent", tc.name, q)
			}
			sent += len(q)
		}
		if sent != len(konst.DoctorQueries) {
			t.Errorf("%s: the queries add up to %d bytes, DoctorQueries is %d", tc.name, sent, len(konst.DoctorQueries))
		}
	}
}

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
