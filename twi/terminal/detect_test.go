package terminal

import (
	"bytes"
	"testing"

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
