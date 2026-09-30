package text

import (
	"iter"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestWidth(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		clusters int
		width    int
	}{
		{"ascii", "a", 1, 1},
		{"cjk", "中", 1, 2},
		{"combining", "e\U00000301", 1, 1},
		{"skin tone", "\U0001F44D\U0001F3FD", 1, 2},
		{"zwj family", "\U0001F468\U0000200D\U0001F469\U0000200D\U0001F467", 1, 2},
		{"heart vs16", "\U00002764\U0000FE0F", 1, 2},
		{"heart text", "\U00002764", 1, 1},
		{"zero width space", "\U0000200B", 1, 0},
		{"flags", "\U0001F1E7\U0001F1F7\U0001F1EF", 2, 4},
		{"hangul jamo", "\U00001100\U00001161\U000011A8", 1, 2},
		{"mixed", "a中e\U00000301", 3, 4},
	}
	for _, c := range cases {
		got := slices.Collect(Graphemes(c.s))
		if len(got) != c.clusters || Width(c.s) != c.width {
			t.Errorf("%s %+q: %d clusters %q, width %d; want %d clusters, width %d", c.name, c.s, len(got), got, Width(c.s), c.clusters, c.width)
			continue
		}
		t.Logf("%-16s %+-40q clusters %d width %d", c.name, c.s, len(got), Width(c.s))
	}
	t.Logf("Unicode %s", UnicodeVersion)
}

func TestGraphemeBreakTest(t *testing.T) {
	conformance(t, "GraphemeBreakTest", Graphemes)
}

func TestWordBreakTest(t *testing.T) {
	conformance(t, "WordBreakTest", Words)
}

func conformance(t *testing.T, name string, segments func(string) iter.Seq[string]) {
	data, err := os.ReadFile("testdata/" + name + ".txt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "# "+name+"-"+UnicodeVersion+".txt") {
		t.Fatalf("%s.txt is not Unicode %s", name, UnicodeVersion)
	}
	run, failed := 0, 0
	for line := range strings.Lines(string(data)) {
		rule, _, _ := strings.Cut(line, "#")
		fields := strings.Fields(rule)
		if len(fields) == 0 {
			continue
		}
		var s strings.Builder
		var want []string
		for _, field := range fields {
			switch field {
			case "÷":
				if s.Len() > 0 {
					want = append(want, s.String())
					s.Reset()
				}
			case "×":
			default:
				code, err := strconv.ParseUint(field, 16, 32)
				if err != nil {
					t.Fatal(err)
				}
				s.WriteRune(rune(code))
			}
		}
		run++
		got := slices.Collect(segments(strings.Join(want, "")))
		if !slices.Equal(got, want) {
			failed++
			t.Errorf("%s: got %+q", strings.TrimSpace(rule), got)
		}
	}
	t.Logf("%s-%s: %d lines run, %d failed", name, UnicodeVersion, run, failed)
	if run == 0 {
		t.Fatal("no lines run")
	}
}
