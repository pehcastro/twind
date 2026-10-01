package markdown

import (
	"regexp"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/tailwind"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen -o twir_gen_test.go -func styles

func TestRenderStylesFresh(t *testing.T) {
	stale, err := tailwind.Stale(".", "twir_gen_test.go")
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("twir_gen_test.go is stale against the classes in twi/markdown: run go generate")
	}
}

const sample = "# Twind Markdown\n\nPages are **Markdown** with *components* between the prose. Read the [docs](https://twind.dev/docs) first.\n\n- Headings return anchors\n- Lists nest\n  - like this\n\n| Block | Drawn as |\n| :---- | -------: |\n| Code | muted surface |\n| Table | bordered grid |\n\n```go\nfunc main() {\n\tfmt.Println(\"hi\")\n}\n```\n\n<Callout title=\"Heads up\" tone=\"info\" />\n"

func driven(t *testing.T, src string, o *Options) *drive.Driver {
	t.Helper()
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	page, err := Parse("sample.md", src, callout())
	if err != nil {
		t.Fatal(err)
	}
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		if copied := o.Copy; copied != nil {
			o.Copy = func(code string) { copied(code); rt.Invalidate() }
		}
		return func() twi.Node { return Render(page, *o) }
	}, drive.Styles(sheet), drive.Size(80, 30))
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	return d
}

func TestRenderSample(t *testing.T) {
	var followed []string
	d := driven(t, sample, &Options{Follow: func(target string) { followed = append(followed, target) }})
	frame := d.Frame().Text()
	t.Logf("80x30 frame:\n%s", frame)
	for _, want := range []string{"Twind Markdown", "Pages are Markdown with components", "• Headings return anchors", "◦ like this", "Block", "Drawn as", "muted surface", "bordered grid", "go", "func main() {", `fmt.Println("hi")`, "Heads up", "tone: info"} {
		if !strings.Contains(frame, want) {
			t.Errorf("frame has no %q", want)
		}
	}
	for _, gone := range []string{"**", "```", "<Callout", "https://twind.dev"} {
		if strings.Contains(frame, gone) {
			t.Errorf("frame shows markup %q", gone)
		}
	}
	for y, line := range strings.Split(frame, "\n") {
		if x := strings.Index(line, "docs"); x >= 0 {
			d.Click(len([]rune(line[:x])), y)
			break
		}
	}
	if len(followed) != 1 || followed[0] != "https://twind.dev/docs" {
		t.Errorf("clicking the link followed %q, want the docs target once", followed)
	}
	if err := d.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestRenderHostile(t *testing.T) {
	d := driven(t, hostilePage(), &Options{})
	frame := d.Frame()
	clean(t, "frame text", frame.Text())
	for _, want := range []string{"Head", "fake", "para", "func"} {
		if !strings.Contains(frame.Text(), want) {
			t.Errorf("frame has no %q:\n%s", want, frame.Text())
		}
	}
	if rest := regexp.MustCompile(`\x1b\[[0-9;:]*m`).ReplaceAllString(frame.ANSI(), ""); strings.ContainsAny(rest, "\x1b\u009b\u009d") {
		i := strings.IndexAny(rest, "\x1b\u009b\u009d")
		t.Errorf("an escape that is not SGR reached the output: %q", rest[i:min(len(rest), i+16)])
	}
}

func TestSpacing(t *testing.T) {
	d := driven(t, "Intro.\n\n## Section\n\nAfter.\n\n- one\n- two\n  - nested\n\n```go\nx\n```\n\n- loose a\n\n- loose b\n", &Options{})
	lines := strings.Split(d.Frame().Text(), "\n")
	row := func(s string) int {
		for y, line := range lines {
			if strings.Contains(line, s) {
				return y
			}
		}
		t.Fatalf("no %q in the frame:\n%s", s, d.Frame().Text())
		return 0
	}
	for _, c := range []struct {
		what        string
		above, next int
		want        int
	}{
		{"the first block from the top", -1, row("Intro."), 1},
		{"a section heading after a paragraph", row("Intro."), row("Section"), 3},
		{"a paragraph after a heading", row("Section"), row("After."), 2},
		{"a list after a paragraph", row("After."), row("• one"), 2},
		{"two tight list items", row("• one"), row("• two"), 2},
		{"a nested list after its item", row("• two"), row("◦ nested"), 2},
		{"two loose list items", row("• loose a"), row("• loose b"), 2},
	} {
		if got := c.next - c.above; got != c.want {
			t.Errorf("%s: %d rows apart, want %d:\n%s", c.what, got, c.want, d.Frame().Text())
		}
	}
	if y := row("◦ nested"); strings.TrimSpace(lines[y+1]) != "" || strings.TrimSpace(lines[y+2]) == "" {
		t.Errorf("a code block after a list: want one blank row, then the block:\n%s", d.Frame().Text())
	}
}

func longPage() string {
	var b strings.Builder
	for i := range 20 {
		b.WriteString("## Section " + strings.Repeat("I", i%4+1) + "\n\nSome **bold** text, *emphasis*, `code` and a [link](https://example.com/a) in a paragraph\nthat wraps over two source lines.\n\n- first item\n- second item with `code`\n  - nested item\n1. one\n2. two\n\n> A quote with *style*.\n\n```go\nfunc f() int {\n\treturn 1\n}\n```\n\n| a | b |\n| - | -: |\n| 1 | 2 |\n\n---\n\n")
	}
	return b.String()
}

func BenchmarkParseRender(b *testing.B) {
	src := longPage()
	if n := strings.Count(src, "\n"); n != 500 {
		b.Fatalf("page has %d lines, want 500", n)
	}
	b.Run("parse", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := Parse("long.md", src, nil); err != nil {
				b.Fatal(err)
			}
		}
	})
	page, err := Parse("long.md", src, nil)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("build", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			Render(page, Options{})
		}
	})
}
