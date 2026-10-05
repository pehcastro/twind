package markdown

import (
	"testing"

	"github.com/pehcastro/twind/internal/highlight"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/theme"
)

func TestFenceLanguages(t *testing.T) {
	var h Highlighter
	for _, c := range []struct {
		language, src string
		want          highlight.Kind
	}{
		{"go", "func main() {}", highlight.Keyword},
		{"bash", "echo $HOME", highlight.Variable},
		{"sh", "echo $HOME", highlight.Variable},
		{"shell", "echo $HOME", highlight.Variable},
		{"console", "$ ls\nfile", highlight.Prompt},
		{"json", `{"a": true}`, highlight.Boolean},
		{"toml", "[[bin]]\nname = \"x\"", highlight.TableHeader},
		{"css", ".card:hover { color: red; }", highlight.SelectorClass},
		{"yaml", "a: ~", highlight.Null},
		{"md", "# Title", highlight.HeadingMarker},
		{"markdown", "# Title", highlight.HeadingMarker},
		{"ts", "let x: Point = p", highlight.Type},
		{"js", "const f = () => 1", highlight.Function},
		{"tsx", `const a = <div className="x" />`, highlight.TagName},
	} {
		g := h.grammar(c.language)
		if g == nil {
			t.Errorf("%s: no grammar", c.language)
			continue
		}
		var kinds []string
		found := false
		for span := range highlight.Tokens(c.src, g) {
			kinds = append(kinds, span.Kind.String())
			found = found || span.Kind == c.want
		}
		if !found {
			t.Errorf("%s %q: kinds %v, want a %s span", c.language, c.src, kinds, c.want)
		}
	}
	for _, pair := range [][2]string{{"sh", "bash"}, {"shell", "bash"}, {"js", "ts"}, {"markdown", "md"}} {
		if h.grammar(pair[0]) != h.grammar(pair[1]) {
			t.Errorf("%s and %s compiled their grammar twice", pair[0], pair[1])
		}
	}
	for _, plain := range []string{"text", ""} {
		if h.grammar(plain) != nil {
			t.Errorf("%q: got a grammar, want plain", plain)
		}
	}
}

func themed(t *testing.T, view func() twi.Node, width, height int) (drive.Frame, theme.Tokens) {
	t.Helper()
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	dark := theme.Default().WithScheme(theme.Dark)
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(dark)
		return view
	}, drive.With(twi.Styles(sheet)), drive.Size(width, height))
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	return d.Frame(), dark.Tokens
}

func TestHighlightKinds(t *testing.T) {
	kinds := int(highlight.Doctype) + 1
	cells, tokens := themed(t, func() twi.Node {
		rows := make([]twi.NodeOption, kinds)
		for k := range rows {
			rows[k] = twi.Element(twi.Class(kindClass(highlight.Kind(k))), twi.Text("x"))
		}
		return element("flex flex-col", rows)
	}, 2, kinds)
	palette := highlight.DefaultPalette()
	for k := range kinds {
		if got, want := cells.At(0, k).Fg, tokens[palette[k]]; got != want {
			t.Errorf("%s: fg %v, want the %s token %v", highlight.Kind(k), got, palette[k], want)
		}
	}
}

func TestFenceMarkdownStyles(t *testing.T) {
	var h Highlighter
	src := "Some **bold**, *italic*, _**both**_,\n~~struck~~ and `code`."
	cells, tokens := themed(t, func() twi.Node { return element("whitespace-pre", []twi.NodeOption{h.Code("md", src)}) }, 60, 2)
	for _, c := range []struct {
		x, y, n int
		attr    drive.Attr
	}{
		{5, 0, 8, drive.Bold},
		{15, 0, 8, drive.Italic},
		{25, 0, 10, drive.Italic},
		{26, 0, 8, drive.Bold | drive.Italic},
		{0, 1, 10, drive.Strikethrough},
	} {
		for i := range c.n {
			if got := cells.At(c.x+i, c.y); got.Attr&c.attr != c.attr {
				t.Errorf("cell %q at %d,%d: attributes %b, want %b", got.Grapheme, c.x+i, c.y, got.Attr, c.attr)
			}
		}
		if got := cells.At(c.x+c.n, c.y); got.Attr&c.attr == c.attr {
			t.Errorf("cell %q after the span at %d,%d keeps attributes %b", got.Grapheme, c.x+c.n, c.y, got.Attr&c.attr)
		}
	}
	if got, want := cells.At(16, 1).Fg, tokens[theme.SyntaxString]; got != want {
		t.Errorf("inline code: fg %v, want the syntax-string token %v", got, want)
	}
	var g Highlighter
	cells, tokens = themed(t, func() twi.Node { return element("whitespace-pre", []twi.NodeOption{g.Code("go", "/* a\nb */ x")}) }, 20, 2)
	for _, y := range []int{0, 1} {
		if got, want := cells.At(0, y).Fg, tokens[theme.SyntaxComment]; got != want {
			t.Errorf("block comment line %d: fg %v, want the syntax-comment token %v", y, got, want)
		}
	}
}
