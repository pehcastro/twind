package style_test

import (
	"os"
	"strings"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/tailwind"
	"github.com/twind-dev/twind/twi/theme"
)

func rgba(r, g, b, a uint8) color.Color {
	return color.Color{Kind: color.Literal, RGBA: color.RGBA{R: r, G: g, B: b, A: a}}
}

func builtin(t *testing.T, name string, scheme theme.Scheme) *theme.Theme {
	t.Helper()
	for _, th := range theme.Builtin() {
		if th.Name == name && th.Scheme == scheme {
			return &th
		}
	}
	t.Fatalf("no built-in theme %s %d", name, scheme)
	return nil
}

func appSheet(t *testing.T) style.Sheet {
	t.Helper()
	src, err := os.ReadFile("../tailwind/testdata/tailwind-4.3.3/app/output.css")
	if err != nil {
		t.Fatal(err)
	}
	rules, _, err := tailwind.Compile(string(src))
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := style.NewSheet(konst.IRVersion, rules)
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}

func TestThemeTokens(t *testing.T) {
	sheet := appSheet(t)
	cases := []struct {
		name                string
		theme               *theme.Theme
		card, muted, border color.Color
	}{
		{"zinc light", builtin(t, "zinc", theme.Light), rgba(255, 255, 255, 255), rgba(113, 113, 123, 255), rgba(228, 228, 231, 255)},
		{"zinc dark", builtin(t, "zinc", theme.Dark), rgba(24, 24, 27, 255), rgba(159, 159, 169, 255), rgba(255, 255, 255, 26)},
		{"rose dark", builtin(t, "rose", theme.Dark), rgba(23, 23, 23, 255), rgba(161, 161, 161, 255), rgba(255, 255, 255, 26)},
	}
	plain := sheet.Compute(style.ComputedStyle{}, []string{"bg-red-500"}).Background
	for _, c := range cases {
		themed := sheet.WithTheme(c.theme)
		got := themed.Compute(style.ComputedStyle{}, strings.Fields("bg-card text-muted-foreground border"))
		t.Logf("%s: background %v color %v border %v", c.name, got.Background.RGBA, got.Color.RGBA, got.BorderColor.RGBA)
		if got.Background != c.card || got.Color != c.muted || got.BorderColor != c.border {
			t.Errorf("%s: background %v color %v border %v, want %v %v %v", c.name, got.Background, got.Color, got.BorderColor, c.card, c.muted, c.border)
		}
		if red := themed.Compute(style.ComputedStyle{}, []string{"bg-red-500"}).Background; red != plain {
			t.Errorf("%s: bg-red-500 %v, want %v as with no theme", c.name, red, plain)
		}
		half := c.card
		half.RGBA.A = 128
		if got := themed.Compute(style.ComputedStyle{}, []string{"bg-card/50"}).Background; got != half {
			t.Errorf("%s: bg-card/50 %v, want the card at alpha 0.5 %v", c.name, got, half)
		}
	}
	if got := sheet.WithTheme(builtin(t, "rose", theme.Dark)).Compute(style.ComputedStyle{}, []string{"bg-primary"}).Background; got != rgba(165, 0, 54, 255) {
		t.Errorf("rose dark bg-primary %v, want 165,0,54", got)
	}
}

func TestThemeFallback(t *testing.T) {
	sheet := appSheet(t)
	partial := &theme.Theme{Name: "partial", Scheme: theme.Dark, Tokens: theme.Tokens{theme.Primary: rgba(1, 2, 3, 255)}}
	got := sheet.WithTheme(partial).Compute(style.ComputedStyle{}, strings.Fields("bg-card border"))
	if got.Background != rgba(24, 24, 27, 255) || got.BorderColor != rgba(255, 255, 255, 26) {
		t.Errorf("dark theme without card or border: background %v border %v, want the dark build values zinc-900 and white 10%%", got.Background, got.BorderColor)
	}
	if got := sheet.WithTheme(partial).Compute(style.ComputedStyle{}, []string{"bg-primary"}).Background; got != rgba(1, 2, 3, 255) {
		t.Errorf("bg-primary %v, want the theme's 1,2,3", got)
	}
	plain := sheet.Compute(style.ComputedStyle{}, strings.Fields("bg-card border dark:bg-zinc-900"))
	if plain.Background != rgba(255, 255, 255, 255) || plain.BorderColor != rgba(228, 228, 231, 255) {
		t.Errorf("no theme: background %v border %v, want the light build values", plain.Background, plain.BorderColor)
	}
	light := sheet.WithTheme(&theme.Theme{Scheme: theme.Light}).Compute(style.ComputedStyle{}, []string{"dark:bg-zinc-900"})
	if light.Background.Kind != color.Unset {
		t.Errorf("light theme applied dark:bg-zinc-900: %v", light.Background)
	}
	dark := sheet.WithTheme(&theme.Theme{Scheme: theme.Dark}).Compute(style.ComputedStyle{}, []string{"dark:bg-zinc-900"})
	if dark.Background != rgba(24, 24, 27, 255) {
		t.Errorf("dark theme: dark:bg-zinc-900 %v, want zinc-900", dark.Background)
	}
}
