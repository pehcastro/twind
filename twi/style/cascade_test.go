package style_test

import (
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/theme"
)

func edges(first style.Property, cells float64) []style.Declaration {
	var decls []style.Declaration
	for p := first; p < first+4; p++ {
		decls = append(decls, style.Declaration{Property: p, Length: style.Length{Value: cells}})
	}
	return decls
}

func TestShorthandAndLonghandByRank(t *testing.T) {
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "p-2", Decls: edges(style.PropPaddingTop, 2)},
		{Class: "pt-4", Decls: []style.Declaration{{Property: style.PropPaddingTop, Length: style.Length{Value: 4}}}},
		{Class: "p-8", Decls: edges(style.PropPaddingTop, 8)},
		{Class: "gap-3", Decls: []style.Declaration{{Property: style.PropRowGap, Length: style.Length{Value: 3}}, {Property: style.PropColumnGap, Length: style.Length{Value: 3}}}},
		{Class: "gap-x-5", Decls: []style.Declaration{{Property: style.PropColumnGap, Length: style.Length{Value: 5}}}},
		{Class: "twice", Decls: append(edges(style.PropMarginTop, 1), style.Declaration{Property: style.PropMarginLeft, Length: style.Length{Value: 6}})},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		classes                  []string
		top, right, bottom, left float64
		row, column, marginLeft  float64
	}{
		{[]string{"pt-4", "p-2"}, 4, 2, 2, 2, 0, 0, 6},
		{[]string{"p-2", "pt-4"}, 4, 2, 2, 2, 0, 0, 6},
		{[]string{"p-8", "pt-4"}, 8, 8, 8, 8, 0, 0, 6},
		{[]string{"pt-4", "p-8", "p-2"}, 8, 8, 8, 8, 0, 0, 6},
		{[]string{"gap-x-5", "gap-3"}, 0, 0, 0, 0, 3, 5, 6},
	} {
		got := sheet.Compute(style.ComputedStyle{}, append(tc.classes, "twice"))
		p := got.Padding
		if p.Top.Value != tc.top || p.Right.Value != tc.right || p.Bottom.Value != tc.bottom || p.Left.Value != tc.left || got.RowGap.Value != tc.row || got.ColumnGap.Value != tc.column || got.Margin.Left.Value != tc.marginLeft || got.Margin.Top.Value != 1 {
			t.Errorf("%v: padding %v %v %v %v, gaps %v %v, margin %v %v", tc.classes, p.Top.Value, p.Right.Value, p.Bottom.Value, p.Left.Value, got.RowGap.Value, got.ColumnGap.Value, got.Margin.Top.Value, got.Margin.Left.Value)
		}
	}
}

func TestSchemeTwinWithoutThemeToken(t *testing.T) {
	light, dark := rgba(255, 255, 255, 255), rgba(9, 9, 11, 255)
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "bg-card", Decls: []style.Declaration{{Property: style.PropBackground, Color: light, Token: theme.Card, Mix: konst.OpaquePercent}}},
		{Class: "bg-card", When: style.Condition{Scheme: style.SchemeDark}, Decls: []style.Declaration{{Property: style.PropBackground, Color: dark, Token: theme.Card, Mix: konst.OpaquePercent}}},
		{Class: "hover:bg-card", When: style.Condition{States: style.StateHover}, Decls: []style.Declaration{{Property: style.PropBackground, Color: light, Token: theme.Card, Mix: konst.OpaquePercent}}},
		{Class: "hover:bg-card", When: style.Condition{States: style.StateHover, Scheme: style.SchemeDark}, Decls: []style.Declaration{{Property: style.PropBackground, Color: dark, Token: theme.Card, Mix: konst.OpaquePercent}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	bare := func(scheme theme.Scheme) *theme.Theme {
		th := *builtin(t, "zinc", scheme)
		th.Tokens[theme.Card] = color.Color{}
		return &th
	}
	full := builtin(t, "zinc", theme.Dark)
	for _, tc := range []struct {
		name  string
		theme *theme.Theme
		want  color.Color
	}{
		{"no theme", nil, light},
		{"light theme without the token", bare(theme.Light), light},
		{"dark theme without the token", bare(theme.Dark), dark},
		{"dark theme with the token", full, full.Tokens[theme.Card]},
	} {
		if got := sheet.WithTheme(tc.theme).Compute(style.ComputedStyle{}, []string{"bg-card"}).Background; got != tc.want {
			t.Errorf("%s: background %v, want %v", tc.name, got, tc.want)
		}
		if got := sheet.WithTheme(tc.theme).ComputeState(style.ComputedStyle{}, []string{"hover:bg-card"}, style.NodeState{States: style.StateHover}).Background; got != tc.want {
			t.Errorf("%s, hovered: background %v, want %v", tc.name, got, tc.want)
		}
		if got := sheet.WithTheme(tc.theme).Compute(style.ComputedStyle{}, []string{"hover:bg-card"}).Background; got != (color.Color{}) {
			t.Errorf("%s, not hovered: background %v, want none", tc.name, got)
		}
	}
}

func TestNewSheetRejectsUnknownPropertiesAndTooManyRules(t *testing.T) {
	for _, p := range []style.Property{0, style.PropRadiusBottomLeft + 1, 255} {
		if _, err := style.NewSheet(konst.IRVersion, []style.Rule{{Decls: []style.Declaration{{Property: p}}}}); err == nil {
			t.Errorf("property %d: no error", p)
		}
	}
	if _, err := style.NewSheet(konst.IRVersion, make([]style.Rule, konst.MaxRules+1)); err == nil {
		t.Errorf("%d rules: no error", konst.MaxRules+1)
	}
	if _, err := style.NewSheet(konst.IRVersion, make([]style.Rule, konst.MaxRules)); err != nil {
		t.Errorf("%d rules: %v", konst.MaxRules, err)
	}
}
