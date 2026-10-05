package style_test

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/internal/tailwind"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/theme"
)

func rgba(r, g, b, a uint8) color.Color {
	return color.Color{Kind: color.Literal, RGBA: color.RGBA{R: r, G: g, B: b, A: a}}
}

func builtin(t testing.TB, name string, scheme theme.Scheme) *theme.Theme {
	t.Helper()
	for _, th := range theme.Builtin() {
		if th.Name == name && th.Scheme == scheme {
			return &th
		}
	}
	t.Fatalf("no built-in theme %s %d", name, scheme)
	return nil
}

func appSheet(t testing.TB) style.Sheet {
	t.Helper()
	src, err := os.ReadFile("../../internal/tailwind/testdata/tailwind-4.3.3/app/output.css")
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
		{"twind light", builtin(t, "twind", theme.Light), rgba(255, 255, 255, 255), rgba(107, 114, 128, 255), rgba(229, 231, 235, 255)},
		{"twind dark", builtin(t, "twind", theme.Dark), rgba(9, 8, 13, 255), rgba(176, 171, 186, 255), rgba(36, 31, 46, 255)},
		{"dream dark", builtin(t, "dream", theme.Dark), rgba(21, 26, 32, 255), rgba(153, 159, 166, 255), rgba(45, 50, 56, 255)},
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
	if got := sheet.WithTheme(builtin(t, "dream", theme.Dark)).Compute(style.ComputedStyle{}, []string{"bg-primary"}).Background; got != rgba(73, 160, 255, 255) {
		t.Errorf("dream dark bg-primary %v, want 73,160,255", got)
	}
}

func pinnedSheet(t *testing.T, classes ...string) style.Sheet {
	t.Helper()
	bin, err := filepath.Abs(filepath.Join("..", "..", ".twind", "bin", "tailwindcss-"+runtime.GOOS+"-"+map[string]string{"amd64": "x64", "arm64": "arm64"}[runtime.GOARCH]))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("no pinned Tailwind: %v", err)
	}
	dir := t.TempDir()
	manifest, input, output := filepath.Join(dir, "manifest.txt"), filepath.Join(dir, "input.css"), filepath.Join(dir, "output.css")
	if err := os.WriteFile(manifest, []byte(strings.Join(classes, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte(tailwind.Input([]string{manifest})), 0o600); err != nil {
		t.Fatal(err)
	}
	if msg, err := exec.Command(bin, "-i", input, "-o", output).CombinedOutput(); err != nil {
		t.Fatalf("tailwind: %v\n%s", err, msg)
	}
	css, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	rules, _, err := tailwind.Compile(string(css))
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := style.NewSheet(konst.IRVersion, rules)
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}

func TestGradientFollowsTheme(t *testing.T) {
	gradient := strings.Fields("bg-linear-to-r from-primary to-primary-300")
	sheet := pinnedSheet(t, append(gradient, "from-primary/40")...)
	primary300, _ := theme.ParseToken("primary-300")
	seen := map[[2]color.RGBA]bool{}
	for _, name := range []string{"twind", "dream", "cloud", "sukuna", "mono"} {
		for _, scheme := range []theme.Scheme{theme.Light, theme.Dark} {
			th := builtin(t, name, scheme)
			from, to := th.Tokens[theme.Primary], th.Tokens[primary300]
			g := sheet.WithTheme(th).Compute(style.ComputedStyle{}, gradient).Gradient
			t.Logf("%s %d: from %v to %v", name, scheme, g.From.Color.RGBA, g.To.Color.RGBA)
			if g.Kind != style.GradientLinear || g.Direction != style.ToRight || to.Kind != color.Literal || g.From.Color != from || g.To.Color != to {
				t.Errorf("%s %d: %+v, want linear to right from %v to %v", name, scheme, g, from, to)
			}
			faded := from
			faded.RGBA.A = uint8(math.Round(float64(from.RGBA.A) * 0.4))
			if got := sheet.WithTheme(th).Compute(style.ComputedStyle{}, []string{"from-primary/40"}).Gradient.From.Color; got != faded {
				t.Errorf("%s %d: from-primary/40 %v, want %v", name, scheme, got, faded)
			}
			seen[[2]color.RGBA{from.RGBA, to.RGBA}] = true
		}
	}
	if len(seen) < 8 {
		t.Errorf("%d distinct gradients over ten themes, want each theme its own", len(seen))
	}
	a, b := builtin(t, "twind", theme.Dark), builtin(t, "cloud", theme.Light)
	first := sheet.WithTheme(a).Compute(style.ComputedStyle{}, gradient).Gradient
	switched := sheet.WithTheme(b).Compute(style.ComputedStyle{}, gradient).Gradient
	back := sheet.WithTheme(a).Compute(style.ComputedStyle{}, gradient).Gradient
	if switched.From.Color != b.Tokens[theme.Primary] || switched.To.Color != b.Tokens[primary300] || back != first || switched == first {
		t.Errorf("twind dark %v, cloud light %v, twind dark again %v: want each its own theme's colours", first, switched, back)
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
