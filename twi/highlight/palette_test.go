package highlight_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/highlight"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/highlight"
	"github.com/pehcastro/twind/twi/theme"
)

func wcagLuminance(c color.RGBA) float64 {
	linear := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(c.R) + 0.7152*linear(c.G) + 0.0722*linear(c.B)
}

func TestPaletteSyntaxTokens(t *testing.T) {
	p := highlight.DefaultPalette()
	for k, want := range map[highlight.Kind]theme.Token{
		highlight.Text: theme.Foreground, highlight.Identifier: theme.Foreground,
		highlight.Keyword: theme.SyntaxKeyword, highlight.String: theme.SyntaxString, highlight.Number: theme.SyntaxNumber,
		highlight.Comment: theme.SyntaxComment, highlight.Function: theme.SyntaxFunction, highlight.Constant: theme.SyntaxConstant,
		highlight.Namespace: theme.SyntaxNamespace, highlight.Parameter: theme.SyntaxParameter, highlight.Punctuation: theme.SyntaxPunctuation,
	} {
		if p[k] != want {
			t.Errorf("%s: %s, want %s", k, p[k], want)
		}
	}
	for k := highlight.Text; k <= highlight.Doctype; k++ {
		if name := p[k].String(); name != "foreground" && !strings.HasPrefix(name, "syntax-") {
			t.Errorf("%s: %q is not a syntax token", k, name)
		}
	}
}

func TestPaletteReadableOnEveryTheme(t *testing.T) {
	var table strings.Builder
	p := highlight.DefaultPalette()
	for _, th := range theme.Builtin() {
		scheme := [...]string{theme.Light: "light", theme.Dark: "dark"}[th.Scheme]
		fmt.Fprintf(&table, "%s-%s", th.Name, scheme)
		for k := highlight.Text; k <= highlight.Doctype; k++ {
			fg := wcagLuminance(th.Tokens[p[k]].RGBA)
			lowest := math.Inf(1)
			for _, surface := range []theme.Token{theme.Muted, theme.Background} {
				bg := wcagLuminance(th.Tokens[surface].RGBA)
				ratio := (max(fg, bg) + 0.05) / (min(fg, bg) + 0.05)
				lowest = min(lowest, ratio)
				if ratio < konst.ReadableContrast {
					t.Errorf("%s %s: %s as %s is %.2f:1 on %s", th.Name, scheme, k, p[k], ratio, surface)
				}
			}
			fmt.Fprintf(&table, " %s=%.1f", k, lowest)
		}
		table.WriteString("\n")
	}
	t.Log("\n" + table.String())
}
