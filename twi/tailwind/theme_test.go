package tailwind

import (
	"os"
	"testing"

	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/theme"
	"github.com/pehcastro/twind/twi/theme/shadcn"
)

func TestSidebarChartSelectionTokens(t *testing.T) {
	src, err := os.ReadFile(appFixture + "/output.css")
	if err != nil {
		t.Fatal(err)
	}
	rules, _, err := Compile(string(src))
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := style.NewSheet(1, rules)
	if err != nil {
		t.Fatal(err)
	}
	preset, err := os.ReadFile("../theme/testdata/zinc.css")
	if err != nil {
		t.Fatal(err)
	}
	zinc, _, err := shadcn.Parse("zinc.css", preset)
	if err != nil {
		t.Fatal(err)
	}
	zincLight, zincDark, twindDark := zinc.WithScheme(theme.Light), zinc.WithScheme(theme.Dark), theme.Default()
	background := func(c style.ComputedStyle) color.Color { return c.Background }
	foreground := func(c style.ComputedStyle) color.Color { return c.Color }
	cases := []struct {
		class string
		token theme.Token
		pick  func(style.ComputedStyle) color.Color
	}{
		{"bg-sidebar", theme.Sidebar, background},
		{"border-sidebar-border", theme.SidebarBorder, func(c style.ComputedStyle) color.Color { return c.BorderColor }},
		{"bg-chart-3", theme.Chart3, background},
		{"text-sidebar-accent-foreground", theme.SidebarAccentForeground, foreground},
		{"bg-sidebar-primary", theme.SidebarPrimary, background},
		{"bg-selection", theme.Selection, background},
		{"text-selection-foreground", theme.SelectionForeground, foreground},
	}
	for _, tc := range cases {
		byScheme := map[style.Scheme]style.Declaration{}
		for _, r := range rules {
			if r.Class == tc.class && r.When.States == 0 && len(r.Decls) > 0 {
				byScheme[r.When.Scheme] = r.Decls[len(r.Decls)-1]
			}
		}
		light, ok := byScheme[style.SchemeAny]
		if !ok {
			t.Errorf("%s: no rule in the fixture", tc.class)
			continue
		}
		if light.Token != tc.token {
			t.Errorf("%s: token %v, want %v", tc.class, light.Token, tc.token)
		}
		if want := zincLight.Tokens[tc.token]; light.Color != want {
			t.Errorf("%s: build colour %+v, want zinc light %+v", tc.class, light.Color.RGBA, want.RGBA)
		}
		dark, ok := byScheme[style.SchemeDark]
		if !ok {
			dark = light
		}
		if want := zincDark.Tokens[tc.token]; dark.Color != want {
			t.Errorf("%s: dark build colour %+v, want zinc dark %+v", tc.class, dark.Color.RGBA, want.RGBA)
		}
		if got, want := tc.pick(sheet.WithTheme(&twindDark).Compute(style.ComputedStyle{}, []string{tc.class})), twindDark.Tokens[tc.token]; got != want {
			t.Errorf("%s under twind dark: %+v, want %+v", tc.class, got.RGBA, want.RGBA)
		}
	}
}
