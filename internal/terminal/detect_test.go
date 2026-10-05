package terminal

import (
	"bytes"
	"strings"
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/terminal"
	"github.com/pehcastro/twind/twi/color"
)

func TestProfile(t *testing.T) {
	cases := []struct {
		name     string
		terminal bool
		env      map[string]string
		want     color.Profile
	}{
		{"no color on a terminal keeps attributes", true, map[string]string{"NO_COLOR": "1", "COLORTERM": "truecolor"}, color.Attributes},
		{"no color beats force color", true, map[string]string{"NO_COLOR": "1", "FORCE_COLOR": "3", "COLORTERM": "truecolor"}, color.Attributes},
		{"no color on a buffer gets none", false, map[string]string{"NO_COLOR": "1"}, color.None},
		{"no color with force color on a buffer", false, map[string]string{"NO_COLOR": "1", "FORCE_COLOR": "3"}, color.Attributes},
		{"no color on a dumb term", true, map[string]string{"NO_COLOR": "1", "TERM": "dumb"}, color.None},
		{"empty no color is unset", true, map[string]string{"NO_COLOR": "", "COLORTERM": "truecolor"}, color.TrueColor},
		{"buffer gets none", false, map[string]string{"COLORTERM": "truecolor"}, color.None},
		{"force color on a buffer", false, map[string]string{"FORCE_COLOR": "1"}, color.ANSI16},
		{"force color level three on a buffer", false, map[string]string{"FORCE_COLOR": "3"}, color.TrueColor},
		{"force color keeps a better detection", false, map[string]string{"FORCE_COLOR": "1", "COLORTERM": "24bit"}, color.TrueColor},
		{"force color zero turns it off", true, map[string]string{"FORCE_COLOR": "0", "COLORTERM": "truecolor"}, color.None},
		{"colorterm truecolor on a terminal", true, map[string]string{"COLORTERM": "truecolor"}, color.TrueColor},
		{"windows terminal session", true, map[string]string{"WT_SESSION": "3c1c9a4e"}, color.TrueColor},
		{"term 256color", true, map[string]string{"TERM": "xterm-256color"}, color.ANSI256},
		{"plain term", true, map[string]string{"TERM": "xterm"}, color.ANSI16},
		{"dumb term", true, map[string]string{"TERM": "dumb", "COLORTERM": "truecolor"}, color.None},
	}
	for _, tc := range cases {
		env := func(key string) string { return tc.env[key] }
		if got := profileFor(tc.terminal, color.None, env); got != tc.want {
			t.Errorf("%s: profile %d, want %d", tc.name, got, tc.want)
		}
		if tc.terminal {
			continue
		}
		if got := Profile(&bytes.Buffer{}, env); got != tc.want {
			t.Errorf("%s: Profile(bytes.Buffer) %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestProfileOfAConsoleThatPromisesTruecolor(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want color.Profile
	}{
		{"no env", nil, color.TrueColor},
		{"mintty through the inbox conpty", map[string]string{"TERM": "xterm", "TERM_PROGRAM": "mintty"}, color.TrueColor},
		{"term 256color", map[string]string{"TERM": "xterm-256color"}, color.TrueColor},
		{"no color", map[string]string{"NO_COLOR": "1"}, color.Attributes},
		{"force color zero", map[string]string{"FORCE_COLOR": "0"}, color.None},
		{"force color one is a floor", map[string]string{"FORCE_COLOR": "1"}, color.TrueColor},
		{"dumb term", map[string]string{"TERM": "dumb"}, color.None},
	}
	for _, tc := range cases {
		if got := profileFor(true, color.TrueColor, func(key string) string { return tc.env[key] }); got != tc.want {
			t.Errorf("%s: profile %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestProfileRaisedByTheTerminalsAnswer(t *testing.T) {
	const da1 = "\x1b[?64;1;2;6;9;15;16;17;18;21;22;28c"
	cases := []struct {
		name    string
		env     map[string]string
		answers []string
		asked   bool
		want    color.Profile
	}{
		{"xterm echoes the colour, colons", map[string]string{"TERM": "xterm"}, []string{"\x1bP1$r0;38:2::1:2:3m\x1b\\" + da1}, true, color.TrueColor},
		{"echoes the colour, semicolons", map[string]string{"TERM": "xterm"}, []string{"\x1bP1$r0;38;2;1;2;3m\x1b\\" + da1}, true, color.TrueColor},
		{"answer split over two reads", map[string]string{"TERM": "xterm-256color"}, []string{"\x1bP1$r0;38:2::1", ":2:3m\x1b\\" + da1}, true, color.TrueColor},
		{"quantised to the palette", map[string]string{"TERM": "xterm-256color"}, []string{"\x1bP1$r0;38;5;16m\x1b\\" + da1}, true, color.ANSI256},
		{"invalid request", map[string]string{"TERM": "xterm"}, []string{"\x1bP0$r\x1b\\" + da1}, true, color.ANSI16},
		{"a valid reply without the colour", map[string]string{"TERM": "xterm"}, []string{"\x1bP1$r0m\x1b\\" + da1}, true, color.ANSI16},
		{"no DECRQSS, fence only", map[string]string{"TERM": "xterm"}, []string{da1}, true, color.ANSI16},
		{"colorterm already says truecolor", map[string]string{"TERM": "xterm", "COLORTERM": "truecolor"}, nil, false, color.TrueColor},
		{"no color is not asked", map[string]string{"TERM": "xterm", "NO_COLOR": "1"}, nil, false, color.Attributes},
		{"dumb is not asked", map[string]string{"TERM": "dumb"}, nil, false, color.None},
	}
	for _, tc := range cases {
		term := newFake(tc.answers...)
		opened := false
		open := func() (tty, error) {
			opened = true
			return term.tty, nil
		}
		got := raise(profileFor(true, color.None, func(key string) string { return tc.env[key] }), term, open)
		if got != tc.want || opened != tc.asked {
			t.Errorf("%s: profile %d asked %t, want %d asked %t", tc.name, got, opened, tc.want, tc.asked)
		}
		if tc.asked && !strings.HasSuffix(term.out(), konst.TruecolorFenced) {
			t.Errorf("%s: wrote %q, want the fenced truecolor probe", tc.name, term.out())
		}
		if tc.asked && term.tty.restored != 1 {
			t.Errorf("%s: tty restored %d times, want 1", tc.name, term.tty.restored)
		}
	}
}
