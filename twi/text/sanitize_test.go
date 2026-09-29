package text

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestHostile(t *testing.T) {
	corpus, err := os.ReadFile("testdata/hostile.txt")
	if err != nil {
		t.Fatal(err)
	}
	cases := 0
	for line := range strings.Lines(string(corpus)) {
		fields := strings.Split(strings.TrimSuffix(line, "\n"), "\t")
		name := fields[0]
		var parts []string
		for _, field := range fields[1:] {
			s, err := strconv.Unquote(field)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			parts = append(parts, s)
		}
		want := parts[len(parts)-1]
		parts = parts[:len(parts)-1]
		var joined strings.Builder
		for _, part := range parts {
			joined.WriteString(Sanitize(part, RemoveBidi))
		}
		got := Sanitize(strings.Join(parts, ""), RemoveBidi)
		if got != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
		for _, out := range []string{got, joined.String()} {
			if bad := unsafeText(out); bad != "" {
				t.Errorf("%s: %q keeps %s", name, out, bad)
			}
		}
		cases++
	}
	t.Logf("%d hostile cases", cases)
}

func TestHostileBidiShown(t *testing.T) {
	got := Sanitize("file\U0000202Egnp.exe", ShowBidi)
	if got != "file<U+202E>gnp.exe" {
		t.Errorf("got %q", got)
	}
}

func unsafeText(s string) string {
	switch {
	case !utf8.ValidString(s):
		return "invalid UTF-8"
	case strings.IndexByte(s, 0x1b) >= 0:
		return "byte 0x1B"
	case strings.IndexByte(s, 0x9b) >= 0:
		return "byte 0x9B"
	}
	for _, r := range s {
		if (r < 0x20 && r != '\t' && r != '\n') || (r >= 0x7f && r <= 0x9f) || (r >= 0x202a && r <= 0x202e) || (r >= 0x2066 && r <= 0x2069) {
			return strconv.QuoteRune(r)
		}
	}
	return ""
}
