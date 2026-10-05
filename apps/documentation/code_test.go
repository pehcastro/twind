package docsapp

import (
	"testing"

	"github.com/pehcastro/twind/internal/buffer"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/theme"
)

func codeBlocks(t *testing.T) *drive.Driver {
	t.Helper()
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	d := drive.New(App, drive.Size(120, 160), drive.With(twi.Styles(sheet)))
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	jump(t, d, "Code blocks")
	return d
}

func TestFenceLanguages(t *testing.T) {
	d := codeBlocks(t)
	cells, tokens := d.Frame().Cells(), theme.Default().WithScheme(theme.Dark).Tokens
	syntax := map[color.Color]bool{}
	for token := theme.SyntaxKeyword; token <= theme.SyntaxPunctuation; token++ {
		syntax[tokens[token]] = true
	}
	for _, first := range []string{"func main() {", "for f in *.md; do", "$ twind build ./app", `{ "name": "twind"`, "[server]", "name: twind", "@theme {", "# Title", "const count: number = 3", "export const Card"} {
		x, y := spot(t, d, first)
		seen := map[color.Color]bool{}
		for i := range len([]rune(first)) {
			if cell := cells.At(x+i, y); cell.Grapheme != " " && syntax[cell.Fg] {
				seen[cell.Fg] = true
			}
		}
		if len(seen) < 2 {
			t.Errorf("%q: %d syntax colours, want at least 2:\n%s", first, len(seen), d.Frame().Text())
		}
	}
}

func TestFenceMarkdownStyles(t *testing.T) {
	d := codeBlocks(t)
	t.Logf("Code blocks, 120x160:\n%s", d.Frame().Text())
	cells, tokens := d.Frame().Cells(), theme.Default().WithScheme(theme.Dark).Tokens
	for _, c := range []struct {
		text string
		attr buffer.Attr
	}{
		{"**bold**", buffer.Bold},
		{"*italic*", buffer.Italic},
		{"~~struck~~", buffer.Strikethrough},
	} {
		x, y := spot(t, d, c.text)
		for i := range len(c.text) {
			if got := cells.At(x+i, y); got.Attr&c.attr == 0 {
				t.Errorf("%s: cell %q has attributes %b, want %b", c.text, got.Grapheme, got.Attr, c.attr)
			}
		}
		if got := cells.At(x+len(c.text), y); got.Attr&c.attr != 0 {
			t.Errorf("%s: the cell after it, %q, keeps attribute %b", c.text, got.Grapheme, c.attr)
		}
	}
	x, y := spot(t, d, "# Title")
	if got, want := cells.At(x+2, y).Fg, tokens[theme.SyntaxFunction]; got != want {
		t.Errorf("the Title heading: fg %v, want the syntax-function token %v", got, want)
	}
	x, y = spot(t, d, "`code`")
	if got, want := cells.At(x+1, y).Fg, tokens[theme.SyntaxString]; got != want {
		t.Errorf("inline code: fg %v, want the syntax-string token %v", got, want)
	}
}
