package highlight_test

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/highlight"
)

func TestGo(t *testing.T)            { matchTwinkleplop(t, "go", highlight.Go()) }
func TestBash(t *testing.T)          { matchTwinkleplop(t, "bash", highlight.Bash()) }
func TestJSON(t *testing.T)          { matchTwinkleplop(t, "json", highlight.JSON()) }
func TestTOML(t *testing.T)          { matchTwinkleplop(t, "toml", highlight.TOML()) }
func TestCSS(t *testing.T)           { matchTwinkleplop(t, "css", highlight.CSS()) }
func TestYAML(t *testing.T)          { matchTwinkleplop(t, "yaml", highlight.YAML()) }
func TestMarkdown(t *testing.T)      { matchTwinkleplop(t, "markdown", highlight.Markdown()) }
func TestConsole(t *testing.T)       { matchTwinkleplop(t, "console", highlight.Console()) }
func TestTypeScript(t *testing.T)    { matchTwinkleplop(t, "typescript", highlight.TypeScript()) }
func TestTypeScriptJSX(t *testing.T) { matchTwinkleplop(t, "tsx", highlight.TSX()) }

type named struct{ kind, text string }

func classes(s highlight.Span) string {
	if s.Style == 0 {
		return s.Kind.String()
	}
	set := []string{s.Kind.String()}
	for i, name := range []string{"bold", "italic", "strike", "code", "link_text", "autolink"} {
		if s.Style&(1<<i) != 0 && !slices.Contains(set, name) {
			set = append(set, name)
		}
	}
	slices.Sort(set)
	return strings.Join(set, "+")
}

func matchTwinkleplop(t *testing.T, lang string, g *highlight.Grammar) {
	snippets, _ := filepath.Glob(filepath.Join("testdata", lang, "*.txt"))
	if len(snippets) == 0 {
		t.Fatalf("no snippets in testdata/%s", lang)
	}
	for _, snippet := range snippets {
		src, err := os.ReadFile(snippet)
		if err != nil {
			t.Fatal(err)
		}
		fixture, err := os.ReadFile(strings.TrimSuffix(snippet, ".txt") + ".spans")
		if err != nil {
			t.Fatal(err)
		}
		var want []named
		for line := range strings.Lines(string(fixture)) {
			kind, quoted, _ := strings.Cut(strings.TrimSuffix(line, "\n"), " ")
			text, err := strconv.Unquote(quoted)
			if err != nil {
				t.Fatalf("%s: %q: %v", snippet, line, err)
			}
			want = append(want, named{kind, text})
		}
		var got []named
		covered := 0
		for s := range highlight.Tokens(string(src), g) {
			if s.Start != covered || s.End <= s.Start {
				t.Fatalf("%s: span %s %d-%d after byte %d", snippet, s.Kind, s.Start, s.End, covered)
			}
			covered = s.End
			got = append(got, named{classes(s), string(src[s.Start:s.End])})
		}
		if covered != len(src) {
			t.Errorf("%s: spans end at byte %d of %d", snippet, covered, len(src))
		}
		mismatches := 0
		for i := range max(len(got), len(want)) {
			var g, w named
			if i < len(got) {
				g = got[i]
			}
			if i < len(want) {
				w = want[i]
			}
			if g != w {
				t.Errorf("%s span %d: got %s %q, twinkleplop %s %q", snippet, i, g.kind, g.text, w.kind, w.text)
				if mismatches++; mismatches == 5 {
					break
				}
			}
		}
	}
}
