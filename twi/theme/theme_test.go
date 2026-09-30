package theme_test

import (
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/theme"
)

func TestThemeBuiltin(t *testing.T) {
	seen := map[string]int{}
	for _, th := range theme.Builtin() {
		seen[th.Name]++
		for tok := theme.Background; tok <= theme.Ring; tok++ {
			if th.Tokens[tok].Kind != color.Literal {
				t.Errorf("%s %d: %s unset", th.Name, th.Scheme, tok)
			}
		}
	}
	for _, name := range []string{"neutral", "zinc", "slate", "stone", "rose", "blue", "green", "orange", "violet"} {
		if seen[name] != 2 {
			t.Errorf("%s: %d schemes, want light and dark", name, seen[name])
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
