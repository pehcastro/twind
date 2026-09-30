package style_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/theme"
)

func TestShadeReusedAcrossThemesAndClasses(t *testing.T) {
	var live theme.Theme
	swapped := appSheet(t).WithTheme(&live)
	focused := style.NodeState{States: style.StateFocusVisible}
	for _, step := range []struct {
		name    string
		scheme  theme.Scheme
		classes string
	}{
		{"zinc", theme.Light, "ring-2 ring-ring"},
		{"zinc", theme.Dark, "ring-2 ring-ring"},
		{"zinc", theme.Light, "ring-2 ring-ring"},
		{"zinc", theme.Light, "ring-1 ring-ring"},
		{"zinc", theme.Light, "ring-inset ring-2 ring-ring"},
		{"zinc", theme.Light, "ring-offset-2 ring-offset-background ring-2 ring-ring"},
		{"zinc", theme.Dark, "shadow-md"},
		{"zinc", theme.Dark, "shadow-md shadow-red-500"},
		{"zinc", theme.Dark, "shadow-md shadow-red-500/50"},
		{"zinc", theme.Dark, "shadow-md inset-shadow-xs inset-shadow-red-500 ring-1"},
		{"zinc", theme.Dark, "border shadow-xs focus-visible:ring-ring/50 focus-visible:ring-2"},
		{"zinc", theme.Light, "border shadow-xs focus-visible:ring-ring/50 focus-visible:ring-2"},
		{"zinc", theme.Light, "shadow-[0_0_0_1px_var(--color-ring)]"},
		{"rose", theme.Light, "shadow-[0_0_0_1px_var(--color-ring)]"},
		{"zinc", theme.Dark, "shadow-[0_0_0_1px_var(--color-ring)]"},
		{"rose", theme.Dark, "shadow-[0_0_0_1px_var(--color-ring)]"},
	} {
		th := builtin(t, step.name, step.scheme)
		classes := strings.Fields(step.classes)
		live = *th
		got := swapped.ComputeState(style.ComputedStyle{}, classes, focused)
		want := appSheet(t).WithTheme(th).ComputeState(style.ComputedStyle{}, classes, focused)
		if !reflect.DeepEqual(got.Shadows, want.Shadows) || !reflect.DeepEqual(got.InsetShadows, want.InsetShadows) {
			t.Errorf("%s %s %d: shadows %+v inset %+v, fresh sheet gives %+v inset %+v", step.name, step.classes, step.scheme, got.Shadows, got.InsetShadows, want.Shadows, want.InsetShadows)
		}
	}
}

func TestShadeKeysOnLength(t *testing.T) {
	black := color.Color{Kind: color.Literal, RGBA: color.RGBA{A: 255}}
	pair := []style.Shadow{{Y: 1, Color: black}, {Y: 2, Color: black}}
	ring := style.Declaration{Property: style.PropRingWidth, Number: 1}
	sheet, err := style.NewSheet(1, []style.Rule{
		{Class: "one", Decls: []style.Declaration{{Property: style.PropShadow, Shadows: pair[:1]}, ring}},
		{Class: "two", Decls: []style.Declaration{{Property: style.PropShadow, Shadows: pair}, ring}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, class := range []string{"one", "two", "one", "two"} {
		want := map[string]int{"one": 2, "two": 3}[class]
		if got := len(sheet.Compute(style.ComputedStyle{}, []string{class}).Shadows); got != want {
			t.Errorf("step %d, %s: %d shadows, want %d", i, class, got, want)
		}
	}
}
