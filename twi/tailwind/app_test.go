package tailwind

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/style"
)

const appFixture = "testdata/tailwind-4.3.3/app"

func rgba(r, g, b, a uint8) color.Color {
	return color.Color{Kind: color.Literal, RGBA: color.RGBA{R: r, G: g, B: b, A: a}}
}

func appWarnings(t *testing.T, prefix string) map[string]Category {
	t.Helper()
	_, warnings := compileFixture(t, appFixture)
	got := map[string]Category{}
	for _, w := range warnings {
		if strings.HasPrefix(w.Class, prefix) {
			t.Log(w)
			got[w.Class] = w.Category
		}
	}
	return got
}

func TestAppFixtureMatchesPreset(t *testing.T) {
	src, err := os.ReadFile(appFixture + "/input.css")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.ReplaceAll(string(src), "\r\n", "\n"), Input([]string{"./classes.txt"}); got != want {
		t.Errorf("%s/input.css is not Input() with the current preset: recapture the fixture\n got %q\nwant %q", appFixture, got, want)
	}
}

func TestShadow(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	black := func(a uint8) color.Color { return rgba(0, 0, 0, a) }
	cases := map[string]struct{ outer, inset []style.Shadow }{
		"shadow-sm":                  {outer: []style.Shadow{{X: 1, Y: 1, Tintable: true, Color: black(13)}}},
		"shadow-md":                  {outer: []style.Shadow{{X: 1, Y: 1, Tintable: true, Color: black(18)}}},
		"shadow-lg":                  {outer: []style.Shadow{{X: 1, Y: 1, Blur: 1, Tintable: true, Color: black(26)}}},
		"shadow-2xl":                 {outer: []style.Shadow{{X: 2, Y: 1, Blur: 2, Tintable: true, Color: black(64)}}},
		"shadow-none":                {},
		"inset-shadow-sm":            {inset: []style.Shadow{{Y: 1, Tintable: true, Color: black(18), Inset: true}}},
		"shadow-inner":               {outer: []style.Shadow{{Y: 1, Tintable: true, Color: black(18), Inset: true}}},
		"shadow-[0_2px_0_-1px_#000]": {outer: []style.Shadow{{Y: 2, Spread: -1, Tintable: true, Color: black(255)}}},
		"shadow-md inset-shadow-sm":  {outer: []style.Shadow{{X: 1, Y: 1, Tintable: true, Color: black(18)}}, inset: []style.Shadow{{Y: 1, Tintable: true, Color: black(18), Inset: true}}},
		"shadow-md shadow-none":      {},
	}
	for classes, want := range cases {
		got := sheet.Compute(style.ComputedStyle{}, strings.Fields(classes))
		t.Logf("%s: outer %+v inset %+v", classes, got.Shadows, got.InsetShadows)
		if len(got.Shadows)+len(want.outer) > 0 && !reflect.DeepEqual(got.Shadows, want.outer) {
			t.Errorf("%s: outer %+v, want %+v", classes, got.Shadows, want.outer)
		}
		if len(got.InsetShadows)+len(want.inset) > 0 && !reflect.DeepEqual(got.InsetShadows, want.inset) {
			t.Errorf("%s: inset %+v, want %+v", classes, got.InsetShadows, want.inset)
		}
	}
	warned := appWarnings(t, "")
	for _, class := range []string{"shadow-2xs", "shadow-xs", "shadow-sm", "shadow", "shadow-md", "shadow-lg", "shadow-xl", "shadow-2xl", "shadow-none", "inset-shadow-2xs", "inset-shadow-xs", "inset-shadow-sm", "shadow-inner"} {
		if c, ok := warned[class]; ok {
			t.Errorf("%s warned %s", class, c)
		}
	}
	if warned["ring-2"] != Unsupported {
		t.Errorf("ring-2: %v, want an unsupported warning", warned)
	}
}

func TestShadowColor(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	red, err := color.Parse("oklch(63.7% 0.237 25.331)")
	if err != nil {
		t.Fatal(err)
	}
	halfRed := red
	halfRed.RGBA.A = 128
	md := func(c color.Color) []style.Shadow { return []style.Shadow{{X: 1, Y: 1, Color: c, Tintable: true}} }
	insetSm := func(c color.Color) []style.Shadow {
		return []style.Shadow{{Y: 1, Color: c, Inset: true, Tintable: true}}
	}
	cases := []struct {
		classes      string
		outer, inset []style.Shadow
	}{
		{"shadow-md shadow-red-500/50", md(halfRed), nil},
		{"shadow-md", md(rgba(0, 0, 0, 18)), nil},
		{"shadow-red-500", nil, nil},
		{"shadow-md shadow-red-500", md(red), nil},
		{"shadow-md shadow-transparent", nil, nil},
		{"shadow-md shadow-[rgba(255,0,0,0.5)]", md(rgba(255, 0, 0, 128)), nil},
		{"shadow-[0_2px_0_-1px_#000] shadow-red-500/50", []style.Shadow{{Y: 2, Spread: -1, Color: halfRed, Tintable: true}}, nil},
		{"shadow-md inset-shadow-sm inset-shadow-red-500", md(rgba(0, 0, 0, 18)), insetSm(red)},
		{"shadow-md inset-shadow-sm shadow-red-500", md(red), insetSm(rgba(0, 0, 0, 18))},
		{"shadow-md", md(rgba(0, 0, 0, 18)), nil},
	}
	for _, tc := range cases {
		got := sheet.Compute(style.ComputedStyle{}, strings.Fields(tc.classes))
		t.Logf("%s: outer %+v inset %+v", tc.classes, got.Shadows, got.InsetShadows)
		if len(got.Shadows)+len(tc.outer) > 0 && !reflect.DeepEqual(got.Shadows, tc.outer) {
			t.Errorf("%s: outer %+v, want %+v", tc.classes, got.Shadows, tc.outer)
		}
		if len(got.InsetShadows)+len(tc.inset) > 0 && !reflect.DeepEqual(got.InsetShadows, tc.inset) {
			t.Errorf("%s: inset %+v, want %+v", tc.classes, got.InsetShadows, tc.inset)
		}
	}
	for class, c := range appWarnings(t, "") {
		if strings.Contains(class, "shadow") || strings.Contains(class, "rgba") {
			t.Errorf("%s warned %s", class, c)
		}
	}
	if got := sheet.Compute(style.ComputedStyle{}, []string{"bg-[rgba(255,0,0,0.5)]/50"}).Background; got != rgba(255, 0, 0, 64) {
		t.Errorf("bg-[rgba(255,0,0,0.5)]/50: %+v, want red at alpha 0.25", got)
	}
	rules, warnings, err := Compile("@layer utilities { .s { box-shadow: 1px 1px #000; } .c { --tw-shadow-color: #f00; } }")
	if err != nil || len(warnings) != 0 {
		t.Fatal(err, warnings)
	}
	literal, err := style.NewSheet(1, rules)
	if err != nil {
		t.Fatal(err)
	}
	if got := literal.Compute(style.ComputedStyle{}, []string{"s", "c"}).Shadows; !reflect.DeepEqual(got, []style.Shadow{{X: 1, Y: 1, Color: rgba(0, 0, 0, 255)}}) {
		t.Errorf("literal box-shadow recoloured: %+v", got)
	}
	matrix, _ := compileFixture(t, cssFixtures+"matrix")
	stock := rgba(0, 0, 0, 26)
	if got, want := matrix.Compute(style.ComputedStyle{}, []string{"shadow-md"}).Shadows, []style.Shadow{{Y: 4, Blur: 6, Spread: -1, Color: stock, Tintable: true}, {Y: 2, Blur: 4, Spread: -2, Color: stock, Tintable: true}}; !reflect.DeepEqual(got, want) {
		t.Errorf("stock shadow-md: %+v, want %+v", got, want)
	}
}

func TestGradient(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	token := func(oklch string) color.Color {
		c, err := color.Parse(oklch)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	indigo600, pink600, zinc500 := token("oklch(51.1% 0.262 276.966)"), token("oklch(59.2% 0.249 0.584)"), token("oklch(55.2% 0.016 285.938)")
	clear := rgba(0, 0, 0, 0)
	cases := map[string]style.Gradient{
		"bg-linear-to-r from-indigo-600 to-pink-600": {
			GradientLine: style.GradientLine{Kind: style.GradientLinear, Direction: style.ToRight, Space: style.OKLab},
			From:         style.GradientStop{Color: indigo600}, Via: style.GradientStop{Color: clear, Position: 0.5}, To: style.GradientStop{Color: pink600, Position: 1},
		},
		"bg-linear-to-tr from-indigo-600 from-10% via-zinc-500 via-40% to-pink-600 to-90%": {
			GradientLine: style.GradientLine{Kind: style.GradientLinear, Direction: style.ToTopRight, Space: style.OKLab},
			From:         style.GradientStop{Color: indigo600, Position: 0.1}, Via: style.GradientStop{Color: zinc500, Position: 0.4}, To: style.GradientStop{Color: pink600, Position: 0.9}, HasVia: true,
		},
		"bg-linear-to-b/srgb to-pink-600": {
			GradientLine: style.GradientLine{Kind: style.GradientLinear, Direction: style.ToBottom, Space: style.SRGB},
			From:         style.GradientStop{Color: clear}, Via: style.GradientStop{Color: clear, Position: 0.5}, To: style.GradientStop{Color: pink600, Position: 1},
		},
		"bg-linear-45 bg-none from-indigo-600": {
			From: style.GradientStop{Color: indigo600}, Via: style.GradientStop{Color: clear, Position: 0.5}, To: style.GradientStop{Color: clear, Position: 1},
		},
		"bg-linear-45": {
			GradientLine: style.GradientLine{Kind: style.GradientLinear, Direction: style.GradientAngle, Angle: 45, Space: style.OKLab},
			From:         style.GradientStop{Color: clear}, Via: style.GradientStop{Color: clear, Position: 0.5}, To: style.GradientStop{Color: clear, Position: 1},
		},
	}
	for classes, want := range cases {
		got := sheet.Compute(style.ComputedStyle{}, strings.Fields(classes)).Gradient
		t.Logf("%s: %+v", classes, got)
		if got != want {
			t.Errorf("%s:\n got %+v\nwant %+v", classes, got, want)
		}
	}
	warned := appWarnings(t, "bg-")
	want := map[string]Category{"bg-radial": Unsupported, "bg-conic": Unsupported, "bg-[linear-gradient(to_right,red,blue)]": Unsupported, "bg-linear-to-l/oklch": Approximated}
	if !reflect.DeepEqual(warned, want) {
		t.Errorf("gradient warnings %v, want %v", warned, want)
	}
}

func TestAlpha(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	got := sheet.Compute(style.ComputedStyle{}, strings.Fields("bg-zinc-950/90 text-zinc-100/50 border-zinc-700/25 opacity-50"))
	t.Logf("background %+v color %+v border %+v opacity %v", got.Background, got.Color, got.BorderColor, got.Opacity)
	if want := rgba(9, 9, 11, 230); got.Background != want {
		t.Errorf("bg-zinc-950/90: %+v, want %+v (alpha 0.9)", got.Background, want)
	}
	if want := rgba(244, 244, 245, 128); got.Color != want {
		t.Errorf("text-zinc-100/50: %+v, want %+v", got.Color, want)
	}
	if want := rgba(63, 63, 70, 64); got.BorderColor != want {
		t.Errorf("border-zinc-700/25: %+v, want %+v", got.BorderColor, want)
	}
	if got.Opacity != 0.5 {
		t.Errorf("opacity-50: %v, want 0.5", got.Opacity)
	}
	if card := sheet.Compute(style.ComputedStyle{}, []string{"bg-card/50"}); card.Background != rgba(255, 255, 255, 128) {
		t.Errorf("bg-card/50: %+v, want white at alpha 0.5", card.Background)
	}
}

func TestTokens(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	zinc950, zinc900, zinc500, zinc200, white := rgba(9, 9, 11, 255), rgba(24, 24, 27, 255), rgba(113, 113, 123, 255), rgba(228, 228, 231, 255), rgba(255, 255, 255, 255)
	got := sheet.Compute(style.ComputedStyle{}, strings.Fields("bg-card text-card-foreground border-border"))
	if got.Background != white || got.Color != zinc950 || got.BorderColor != zinc200 {
		t.Errorf("card: background %+v color %+v border %+v, want white, zinc-950, zinc-200", got.Background, got.Color, got.BorderColor)
	}
	if got := sheet.Compute(style.ComputedStyle{}, []string{"text-muted-foreground"}).Color; got != zinc500 {
		t.Errorf("text-muted-foreground: %+v, want zinc-500", got)
	}
	if got := sheet.Compute(style.ComputedStyle{}, []string{"bg-primary"}).Background; got != zinc900 {
		t.Errorf("bg-primary: %+v, want zinc-900", got)
	}
	src, err := os.ReadFile(appFixture + "/output.css")
	if err != nil {
		t.Fatal(err)
	}
	rules, warnings, err := Compile(string(src))
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range warnings {
		if strings.Contains(w.Class, "ground") || strings.Contains(w.Class, "dark") || strings.Contains(w.Class, "border-") {
			t.Errorf("token warning: %s", w)
		}
	}
	palette := map[string]map[style.Scheme]color.Color{}
	for _, r := range rules {
		if r.When.States != 0 || len(r.Decls) == 0 {
			continue
		}
		if palette[r.Class] == nil {
			palette[r.Class] = map[style.Scheme]color.Color{}
		}
		palette[r.Class][r.When.Scheme] = r.Decls[len(r.Decls)-1].Color
	}
	for _, class := range []string{"bg-background", "text-foreground", "bg-card", "text-card-foreground", "bg-popover", "text-popover-foreground", "bg-primary", "text-primary-foreground", "bg-secondary", "text-secondary-foreground", "bg-muted", "text-muted-foreground", "bg-accent", "text-accent-foreground", "bg-destructive", "border-border", "border-input"} {
		t.Logf("%-26s light %v dark %v", class, palette[class][style.SchemeAny].RGBA, palette[class][style.SchemeDark].RGBA)
	}
	want := map[string]map[style.Scheme]color.Color{
		"bg-card":          {style.SchemeAny: white, style.SchemeDark: zinc900},
		"border-border":    {style.SchemeAny: zinc200, style.SchemeDark: rgba(255, 255, 255, 26)},
		"dark:bg-zinc-900": {style.SchemeDark: zinc900},
		"p-2":              nil,
		"bg-zinc-950/90":   {style.SchemeAny: rgba(9, 9, 11, 230)},
	}
	for class, schemes := range want {
		if schemes == nil {
			if _, dark := palette[class][style.SchemeDark]; dark {
				t.Errorf("%s: a dark rule for a rule with no themed colour", class)
			}
			continue
		}
		if !reflect.DeepEqual(palette[class], schemes) {
			t.Errorf("%s: %+v, want %+v", class, palette[class], schemes)
		}
	}
}
