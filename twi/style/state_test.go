package style_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/tailwind"
	"github.com/twind-dev/twind/twi/theme"
)

func TestStateFocusVisibleRing(t *testing.T) {
	sheet := appSheet(t)
	ring := []style.Shadow{{Spread: 2, Color: color.Color{Kind: color.Current}}}
	for _, tc := range []struct {
		name   string
		states style.State
		want   []style.Shadow
	}{
		{"no state", 0, nil},
		{"focus only", style.StateFocus | style.StateHover, nil},
		{"focus-visible", style.StateFocusVisible, ring},
		{"focus and focus-visible", style.StateFocus | style.StateFocusVisible, ring},
	} {
		got := sheet.ComputeState(style.ComputedStyle{}, []string{"focus-visible:ring-2"}, style.NodeState{States: tc.states}).Shadows
		t.Logf("%s: %+v", tc.name, got)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: shadows %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestStateAttributes(t *testing.T) {
	sheet := appSheet(t)
	accent, card := rgba(244, 244, 245, 255), rgba(255, 255, 255, 255)
	attrs := func(pairs ...string) style.NodeState {
		var n style.NodeState
		for i := 0; i < len(pairs); i += 2 {
			n.Attrs = append(n.Attrs, style.Attr{Name: pairs[i], Value: pairs[i+1]})
		}
		return n
	}
	for _, tc := range []struct {
		classes string
		node    style.NodeState
		want    color.Color
	}{
		{"data-[state=open]:bg-accent", style.NodeState{}, color.Color{}},
		{"data-[state=open]:bg-accent", attrs("data-state", "closed"), color.Color{}},
		{"data-[state=open]:bg-accent", attrs("aria-state", "open"), color.Color{}},
		{"data-[state=open]:bg-accent", attrs("data-state", "open"), accent},
		{"data-[state=open]:bg-accent", attrs("data-side", "top", "data-state", "open"), accent},
		{"bg-card data-[state=open]:bg-accent", attrs("data-state", "open"), accent},
		{"data-[state=open]:bg-accent bg-card", attrs("data-state", "open"), accent},
		{"data-[state=open]:bg-accent bg-card", style.NodeState{}, card},
		{"aria-selected:bg-muted", attrs("aria-selected", "true"), accent},
		{"aria-selected:bg-muted", attrs("aria-selected", "false"), color.Color{}},
		{"checked:bg-primary", style.NodeState{States: style.StateChecked}, rgba(24, 24, 27, 255)},
		{"focus-within:bg-accent", style.NodeState{States: style.StateFocus}, color.Color{}},
		{"focus-within:bg-accent", style.NodeState{States: style.StateFocusWithin}, accent},
	} {
		got := sheet.ComputeState(style.ComputedStyle{}, strings.Fields(tc.classes), tc.node).Background
		if got != tc.want {
			t.Errorf("%s with %+v: background %+v, want %+v", tc.classes, tc.node, got, tc.want)
		}
	}
	for _, tc := range []struct {
		node style.NodeState
		want float64
	}{
		{style.NodeState{}, 1},
		{attrs("data-disabled", ""), 0.5},
		{attrs("data-disabled", "true"), 0.5},
		{style.NodeState{States: style.StateDisabled}, 0.5},
		{style.NodeState{States: style.StateDisabled, Attrs: attrs("data-state", "closed").Attrs}, 0},
	} {
		if got := sheet.ComputeState(style.ComputedStyle{}, strings.Fields("data-[state=closed]:opacity-0 disabled:opacity-50 data-disabled:opacity-50"), tc.node).Opacity; got != tc.want {
			t.Errorf("opacity with %+v: %v, want %v", tc.node, got, tc.want)
		}
	}
}

func TestStateCascade(t *testing.T) {
	sheet := appSheet(t)
	hover := style.NodeState{States: style.StateHover}
	dark := sheet.WithTheme(builtin(t, "twind", theme.Dark))
	light := sheet.WithTheme(builtin(t, "twind", theme.Light))
	if got := light.ComputeState(style.ComputedStyle{}, []string{"dark:hover:bg-input/50"}, hover).Background; got.Kind != color.Unset {
		t.Errorf("dark:hover under light with hover: %+v", got)
	}
	if got := dark.ComputeState(style.ComputedStyle{}, []string{"dark:hover:bg-input/50"}, style.NodeState{}).Background; got.Kind != color.Unset {
		t.Errorf("dark:hover under dark without hover: %+v", got)
	}
	if got := dark.ComputeState(style.ComputedStyle{}, []string{"dark:hover:bg-input/50"}, hover).Background; got != rgba(36, 31, 46, 128) {
		t.Errorf("dark:hover under dark with hover: %+v, want the input token at half alpha", got)
	}
	classes := strings.Fields("bg-card text-card-foreground border shadow-md p-2 hover:bg-muted focus-visible:ring-2 data-[state=open]:bg-accent")
	if plain, stateless := sheet.Compute(style.ComputedStyle{}, classes), sheet.ComputeState(style.ComputedStyle{}, classes, style.NodeState{}); !reflect.DeepEqual(plain, stateless) {
		t.Errorf("Compute and ComputeState with no state differ:\n%+v\n%+v", plain, stateless)
	}
	rules, _, err := tailwind.Compile(`@layer utilities { @media (width >= 80px) { .sm\:p-1:hover { padding: 1px; } .sm\:p-2 { padding: 2px; } } }`)
	if err != nil {
		t.Fatal(err)
	}
	wide, err := style.NewSheet(1, rules)
	if err != nil {
		t.Fatal(err)
	}
	if got := wide.ComputeState(style.ComputedStyle{}, []string{"sm:p-1", "sm:p-2"}, hover).Padding.Top; got.Value != 0 {
		t.Errorf("a breakpoint rule applied: padding %+v", got)
	}
}
