package css

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/css"
)

func readFixture(t *testing.T, name string) []Node {
	t.Helper()
	src, err := os.ReadFile("testdata/tailwind-4.3.3/" + name + "/output.css")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(string(src))
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return nodes
}

func countRules(nodes []Node) int {
	n := 0
	for _, node := range nodes {
		switch v := node.(type) {
		case Rule:
			n += 1 + countRules(v.Block)
		case AtRule:
			n += countRules(v.Block)
		}
	}
	return n
}

func findRule(nodes []Node, selector string) (Rule, bool) {
	for _, node := range nodes {
		switch v := node.(type) {
		case Rule:
			if v.Selector == selector {
				return v, true
			}
			if r, ok := findRule(v.Block, selector); ok {
				return r, true
			}
		case AtRule:
			if r, ok := findRule(v.Block, selector); ok {
				return r, true
			}
		}
	}
	return Rule{}, false
}

func TestFixtureHello(t *testing.T) {
	nodes := readFixture(t, "hello")
	for _, node := range nodes {
		if at, ok := node.(AtRule); ok && at.Name == "layer" {
			t.Logf("@layer %s: %d rules, block %v", text(at.Prelude), countRules(at.Block), at.HasBlock)
		}
	}
	want := map[string]string{
		".p-4":         "padding: calc(var(--spacing) * 4);",
		".bg-zinc-950": "background-color: var(--color-zinc-950);",
		".rounded-lg":  "border-radius: var(--radius-lg);",
		".border":      "border-style: var(--tw-border-style); border-width: 1px;",
	}
	for _, selector := range []string{".p-4", ".bg-zinc-950", ".rounded-lg", ".border"} {
		rule, ok := findRule(nodes, selector)
		if !ok {
			t.Fatalf("%s not found", selector)
		}
		var decls []string
		for _, n := range rule.Block {
			d := n.(Declaration)
			decls = append(decls, d.Property+": "+text(d.Value)+";")
		}
		got := strings.Join(decls, " ")
		t.Logf("%s { %s }", selector, got)
		if got != want[selector] {
			t.Errorf("%s: got %q, want %q", selector, got, want[selector])
		}
	}
}

func TestFixtureMatrix(t *testing.T) {
	nodes := readFixture(t, "matrix")
	mix, ok := findRule(nodes, `.bg-zinc-950\/90`)
	if !ok || len(mix.Block) != 2 {
		t.Fatalf("color-mix rule: %#v", mix)
	}
	supports := mix.Block[1].(AtRule)
	if supports.Name != "supports" || text(supports.Prelude) != "(color: color-mix(in lab, red, red))" {
		t.Errorf("nested @supports: %#v", supports)
	}
	for _, selector := range []string{`.hover\:bg-zinc-800:hover`, `.data-\[selected\=true\]\:bg-zinc-800[data-selected="true"]`, `.w-\[37px\]`} {
		if _, ok := findRule(nodes, selector); !ok {
			t.Errorf("%s not found", selector)
		}
	}
	blur, _ := findRule(nodes, ".blur-sm")
	filter := blur.Block[1].(Declaration)
	if !strings.HasPrefix(text(filter.Value), "var(--tw-blur,) var(--tw-brightness,)") {
		t.Errorf("filter value: %q", text(filter.Value))
	}
}

func TestParseImportantAndCustomProperty(t *testing.T) {
	nodes, err := Parse(".a { color: red!important; --x: ; --font: a,\n    b; b:hover { top: 1px ! important } }")
	if err != nil {
		t.Fatal(err)
	}
	block := nodes[0].(Rule).Block
	red := block[0].(Declaration)
	if !red.Important || text(red.Value) != "red" {
		t.Errorf("red: %#v", red)
	}
	if empty := block[1].(Declaration); empty.Property != "--x" || len(empty.Value) != 0 {
		t.Errorf("--x: %#v", empty)
	}
	if font := block[2].(Declaration); text(font.Value) != "a,\n    b" {
		t.Errorf("--font: %q", text(font.Value))
	}
	hover := block[3].(Rule)
	if top := hover.Block[0].(Declaration); hover.Selector != "b:hover" || !top.Important || text(top.Value) != "1px" {
		t.Errorf("b:hover: %#v", hover)
	}
}

func serialize(b *strings.Builder, nodes []Node) {
	for _, node := range nodes {
		switch v := node.(type) {
		case AtRule:
			b.WriteString("@" + v.Name + " " + text(v.Prelude))
			if !v.HasBlock {
				b.WriteString(";\n")
				continue
			}
			b.WriteString("{\n")
			serialize(b, v.Block)
			b.WriteString("}\n")
		case Rule:
			b.WriteString(v.Selector + "{\n")
			serialize(b, v.Block)
			b.WriteString("}\n")
		case Declaration:
			b.WriteString(v.Property + ":" + text(v.Value))
			if v.Important {
				b.WriteString("!important")
			}
			b.WriteString(";\n")
		}
	}
}

func withoutPos(nodes []Node) []Node {
	out := make([]Node, len(nodes))
	for i, node := range nodes {
		switch v := node.(type) {
		case AtRule:
			v.Pos, v.Block = Pos{}, withoutPos(v.Block)
			out[i] = v
		case Rule:
			v.Pos, v.Block = Pos{}, withoutPos(v.Block)
			out[i] = v
		case Declaration:
			v.Pos = Pos{}
			out[i] = v
		}
	}
	return out
}

func TestRoundTrip(t *testing.T) {
	for _, name := range []string{"hello", "matrix"} {
		first := readFixture(t, name)
		var b strings.Builder
		serialize(&b, first)
		second, err := Parse(b.String())
		if err != nil {
			t.Fatalf("%s: reparse: %v", name, err)
		}
		if !reflect.DeepEqual(withoutPos(first), withoutPos(second)) {
			t.Errorf("%s: tree differs after a round trip", name)
		}
	}
}

func TestMalformed(t *testing.T) {
	deep := strings.Repeat(".a{", 10000) + strings.Repeat("}", 10000)
	tests := []struct {
		name string
		src  string
		kind ErrorKind
		pos  Pos
	}{
		{"unclosed block", ".a {\n  color: red;\n", ErrUnclosedBlock, Pos{1, 4}},
		{"unclosed nested block", "@layer x {\n  .a { color: red; }\n  .b {\n}", ErrUnclosedBlock, Pos{1, 10}},
		{"unclosed string", ".a { content: \"abc }", ErrUnclosedString, Pos{1, 15}},
		{"string broken by newline", ".a { content: 'abc\n' }", ErrUnclosedString, Pos{1, 15}},
		{"stray close brace", ".a { color: red; }\n}", ErrStrayCloseBrace, Pos{2, 1}},
		{"eof inside media prelude", ".a {}\n@media (hover: hover", ErrEOFInPrelude, Pos{2, 1}},
		{"eof inside media prelude without parens", "@media screen", ErrEOFInPrelude, Pos{1, 1}},
		{"nested blocks past the limit", deep, ErrTooDeep, Pos{1, 3*konst.MaxBlockDepth + 3}},
		{"unclosed comment", ".a {} /* x", ErrUnclosedComment, Pos{1, 7}},
		{"value runs to eof", ".a { width: calc(1px", ErrUnclosedBlock, Pos{1, 4}},
		{"rule without block", ".a { color: red }\n.b;", ErrMissingBlock, Pos{2, 1}},
		{"empty value", ".a { color: ; }", ErrEmptyValue, Pos{1, 6}},
		{"quote inside url", ".a { b: url(x\"y) }", ErrBadURL, Pos{1, 9}},
		{"column counts runes", ".é { }\n.ü { c: \"x", ErrUnclosedString, Pos{2, 9}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.src)
			var perr *Error
			if !errors.As(err, &perr) {
				t.Fatalf("got nodes %d and error %v, want %v at %v", len(nodes), err, tt.kind, tt.pos)
			}
			if perr.Kind != tt.kind || perr.Pos != tt.pos {
				t.Errorf("got %v (kind %d at %v), want kind %d at %v", perr, perr.Kind, perr.Pos, tt.kind, tt.pos)
			}
		})
	}
}
