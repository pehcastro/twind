package theme_test

import (
	"math"
	"slices"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/highlight"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/theme"
)

func TestThemeBuiltin(t *testing.T) {
	seen := map[string]int{}
	all := theme.Builtin()
	if all[0].Name != "twind" {
		t.Errorf("first built-in theme %q, want the default twind", all[0].Name)
	}
	for _, th := range all {
		seen[th.Name]++
		for tok := theme.Background; tok <= theme.DestructiveForeground; tok++ {
			if th.Tokens[tok].Kind != color.Literal {
				t.Errorf("%s %d: %s unset", th.Name, th.Scheme, tok)
			}
		}
	}
	for _, name := range append([]string{"neutral", "zinc", "slate", "stone", "rose", "blue", "green", "orange", "violet"}, owners...) {
		if seen[name] != 2 {
			t.Errorf("%s: %d schemes, want light and dark", name, seen[name])
		}
	}
}

func linear(v uint8) float64 {
	s := float64(v) / 255
	if s <= 0.04045 {
		return s / 12.92
	}
	return math.Pow((s+0.055)/1.055, 2.4)
}

func contrast(a, b color.RGBA) float64 {
	la := 0.2126*linear(a.R) + 0.7152*linear(a.G) + 0.0722*linear(a.B)
	lb := 0.2126*linear(b.R) + 0.7152*linear(b.G) + 0.0722*linear(b.B)
	return (max(la, lb) + 0.05) / (min(la, lb) + 0.05)
}

func chromaHue(c color.RGBA) (float64, float64) {
	r, g, b := linear(c.R), linear(c.G), linear(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	a := 1.9779984951*l - 2.4285922050*m + 0.4505937099*s
	bb := 0.0259040371*l + 0.7827717662*m - 0.8086757660*s
	return math.Hypot(a, bb), math.Mod(math.Atan2(bb, a)*180/math.Pi+360, 360)
}

func TestThemeSyntaxReadable(t *testing.T) {
	names := []string{
		"syntax-keyword", "syntax-string", "syntax-number", "syntax-comment", "syntax-function",
		"syntax-constant", "syntax-namespace", "syntax-parameter", "syntax-punctuation",
	}
	for _, th := range theme.Builtin() {
		for _, name := range names {
			tok, ok := theme.ParseToken(name)
			if !ok {
				t.Fatalf("%s: not a token", name)
			}
			fg := th.Tokens[tok]
			if fg.Kind != color.Literal {
				t.Errorf("%s %d: %s unset", th.Name, th.Scheme, name)
				continue
			}
			for _, surface := range []theme.Token{theme.Muted, theme.Background} {
				if ratio := contrast(fg.RGBA, th.Tokens[surface].RGBA); ratio < konst.ReadableContrast {
					t.Errorf("%s %d: %s is %.2f:1 on %s", th.Name, th.Scheme, name, ratio, surface)
				}
			}
		}
	}
}

func TestThemeSyntaxHues(t *testing.T) {
	five := []theme.Token{theme.SyntaxKeyword, theme.SyntaxString, theme.SyntaxNumber, theme.SyntaxFunction, theme.SyntaxComment}
	for _, th := range theme.Builtin() {
		if slices.Contains([]string{"rose", "blue", "green", "orange", "violet"}, th.Name) {
			_, comment := chromaHue(th.Tokens[theme.SyntaxComment].RGBA)
			_, primary := chromaHue(th.Tokens[theme.Primary].RGBA)
			if d := math.Abs(comment - primary); min(d, 360-d) > 10 {
				t.Errorf("%s %d: comment hue %.0f is not tinted to primary hue %.0f", th.Name, th.Scheme, comment, primary)
			}
			continue
		}
		hues := map[theme.Token]float64{}
		for _, tok := range five {
			chroma, hue := chromaHue(th.Tokens[tok].RGBA)
			if chroma < 0.03 {
				t.Errorf("%s %d: %s has chroma %.3f, no hue", th.Name, th.Scheme, tok, chroma)
				continue
			}
			for other, seen := range hues {
				if d := math.Abs(hue - seen); min(d, 360-d) < 30 {
					t.Errorf("%s %d: %s hue %.0f is within 30 of %s hue %.0f", th.Name, th.Scheme, tok, hue, other, seen)
				}
			}
			hues[tok] = hue
		}
	}
}

func TestThemeSidebarChartSelection(t *testing.T) {
	names := []string{
		"sidebar", "sidebar-foreground", "sidebar-primary", "sidebar-primary-foreground",
		"sidebar-accent", "sidebar-accent-foreground", "sidebar-border", "sidebar-ring",
		"chart-1", "chart-2", "chart-3", "chart-4", "chart-5", "selection", "selection-foreground",
	}
	themes := map[string]theme.Theme{}
	for _, th := range theme.Builtin() {
		themes[th.Name+[]string{" light", " dark"}[th.Scheme]] = th
		for _, name := range names {
			tok, ok := theme.ParseToken(name)
			if !ok {
				t.Fatalf("%s: not a token", name)
			}
			if th.Tokens[tok].Kind != color.Literal {
				t.Errorf("%s %d: %s unset", th.Name, th.Scheme, name)
			}
		}
	}
	for _, c := range []struct{ theme, token, value string }{
		{"zinc light", "chart-1", "oklch(0.871 0.006 286.286)"},
		{"zinc dark", "sidebar", "oklch(0.21 0.006 285.885)"},
		{"zinc dark", "sidebar-primary", "oklch(0.488 0.243 264.376)"},
		{"slate light", "chart-4", "oklch(0.828 0.189 84.429)"},
		{"slate dark", "chart-4", "oklch(0.627 0.265 303.9)"},
		{"slate dark", "sidebar-ring", "oklch(0.551 0.027 264.364)"},
		{"violet light", "chart-3", "oklch(0.541 0.281 293.009)"},
		{"violet light", "sidebar-primary", "oklch(0.541 0.281 293.009)"},
		{"violet dark", "sidebar-primary", "oklch(0.606 0.25 292.717)"},
		{"violet light", "sidebar", "oklch(0.985 0 0)"},
		{"rose dark", "sidebar", "oklch(0.21 0.006 285.885)"},
		{"violet light", "selection", "oklch(0% 0 0)"},
		{"zinc dark", "selection", "oklch(0.922 0 0)"},
		{"slate dark", "selection-foreground", "oklch(0.205 0 0)"},
	} {
		tok, _ := theme.ParseToken(c.token)
		want, err := color.Parse(c.value)
		if err != nil {
			t.Fatal(err)
		}
		if got := themes[c.theme].Tokens[tok]; got != want {
			t.Errorf("%s %s = %v, want %v (%s)", c.theme, c.token, got, want, c.value)
		}
	}
}
