package highlight_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/pehcastro/twind/twi/highlight"
)

func spansOf(src string, g *highlight.Grammar) string {
	var b strings.Builder
	for s := range highlight.Tokens(src, g) {
		b.WriteString(s.Kind.String() + " " + strconv.Quote(src[s.Start:s.End]) + "\n")
	}
	return b.String()
}

func TestProbeWithoutEndFailsOnce(t *testing.T) {
	g, err := highlight.Compile(highlight.Definition{States: []highlight.State{
		{Name: "main", Rules: []highlight.Rule{highlight.Match(highlight.Text, "<").Enter("look"), highlight.Match(highlight.Keyword).Letters()}},
		{Name: "look", Probe: true, Rules: []highlight.Rule{highlight.Match(highlight.Text, ">").Enter("tag")}},
		{Name: "tag", Rules: []highlight.Rule{
			highlight.Match(highlight.Punctuation, "<"),
			highlight.Match(highlight.Property).Letters(),
			highlight.Match(highlight.Punctuation, ">").Leave(),
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	got := spansOf("<ab> <cd", g)
	want := "punctuation \"<\"\nproperty \"ab\"\npunctuation \">\"\ntext \" <\"\nkeyword \"cd\"\n"
	if got != want {
		t.Errorf("got\n%swant\n%s", got, want)
	}
}

func TestNestingPastTheStackStaysCovered(t *testing.T) {
	src := strings.Repeat("$(", 1000) + "echo" + strings.Repeat(")", 1000)
	end := 0
	for s := range highlight.Tokens(src, highlight.Bash()) {
		if s.Start != end {
			t.Fatalf("span %d-%d after byte %d", s.Start, s.End, end)
		}
		end = s.End
	}
	if end != len(src) {
		t.Errorf("spans end at %d of %d", end, len(src))
	}
}

func TestBreakStopsTheTokenizer(t *testing.T) {
	n := 0
	for range highlight.Tokens("package main\n\nfunc main() {}\n", highlight.Go()) {
		if n++; n == 3 {
			break
		}
	}
	if n != 3 {
		t.Errorf("%d spans before break, want 3", n)
	}
}

func TestGoroutinesShareAGrammar(t *testing.T) {
	g := highlight.Go()
	var sources []string
	for _, name := range []string{"program", "passes", "edges"} {
		src, err := os.ReadFile(filepath.Join("testdata", "go", name+".txt"))
		if err != nil {
			t.Fatal(err)
		}
		sources = append(sources, string(src))
	}
	want := map[string]string{}
	for _, src := range sources {
		want[src] = spansOf(src, g)
	}
	var wg sync.WaitGroup
	var wrong atomic.Int32
	for i := range 300 {
		src := sources[i%len(sources)]
		wg.Go(func() {
			if spansOf(src, g) != want[src] {
				wrong.Add(1)
			}
		})
	}
	wg.Wait()
	if n := wrong.Load(); n > 0 {
		t.Errorf("%d of 300 concurrent runs differ from the serial spans", n)
	}
}

func TestCompileRejects(t *testing.T) {
	m := highlight.Match
	states := func(s ...highlight.State) highlight.Definition { return highlight.Definition{States: s} }
	main := func(rules ...highlight.Rule) highlight.State { return highlight.State{Name: "main", Rules: rules} }
	many := func(n int) []highlight.Rule {
		rules := make([]highlight.Rule, n)
		for i := range rules {
			rules[i] = m(highlight.Text, "x"+strconv.Itoa(i)).Enter("probe")
		}
		return rules
	}
	cases := []struct {
		name string
		def  highlight.Definition
		want string
	}{
		{"no state", states(), "needs a state"},
		{"unknown target", states(main(m(highlight.String, `"`).Enter("nowhere"))), `unknown state "nowhere"`},
		{"duplicate state", states(main(), main()), "declared twice"},
		{"empty literal", states(main(m(highlight.String, ""))), "empty or not ASCII"},
		{"non-ASCII literal", states(main(m(highlight.String, "é"))), "empty or not ASCII"},
		{"overlapping wide ranges", states(main(m(highlight.String).Range(200, 300), m(highlight.Number).Range(250, 260))), "both match U+00FA"},
		{"two fallbacks", states(main(highlight.Fallback(highlight.String), highlight.Fallback(highlight.Number))), "both fallbacks"},
		{"probe first", states(highlight.State{Name: "p", Probe: true}), "first state cannot probe"},
		{"emitting probe opener", states(main(m(highlight.String, "<").Enter("p")), highlight.State{Name: "p", Probe: true}), "cannot emit"},
		{"too many probe openers", states(main(many(17)...), highlight.State{Name: "probe", Probe: true}), "17 rules open a probe"},
		{"too many rules", states(main(many(255)...), highlight.State{Name: "probe"}), "255 rules"},
	}
	for _, tc := range cases {
		_, err := highlight.Compile(tc.def)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want an error containing %q", tc.name, err, tc.want)
		}
	}
}
