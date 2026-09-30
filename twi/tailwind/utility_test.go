package tailwind

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/theme"
)

func quiet(t *testing.T, match func(class string) bool) {
	t.Helper()
	for class, c := range appWarnings(t, "") {
		if match(class) {
			t.Errorf("%s warned %s", class, c)
		}
	}
}

func hasPrefix(prefixes ...string) func(string) bool {
	return func(class string) bool {
		for _, p := range prefixes {
			if strings.HasPrefix(class, p) {
				return true
			}
		}
		return false
	}
}

func TestTransition(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	standard, ease := style.Easing{X1: 0.4, X2: 0.2, Y2: 1}, style.Easing{X1: 0.25, Y1: 0.1, X2: 0.25, Y2: 1}
	colors := style.TransitionColor | style.TransitionBackground | style.TransitionBorderColor | style.TransitionGradient
	ms := time.Millisecond
	cases := map[string]style.Transition{
		"":                              {Properties: style.TransitionAll, Easing: ease},
		"transition":                    {Properties: style.TransitionAll, Duration: 150 * ms, Easing: standard},
		"transition-all":                {Properties: style.TransitionAll, Duration: 150 * ms, Easing: standard},
		"transition-colors":             {Properties: colors, Duration: 150 * ms, Easing: standard},
		"transition-opacity":            {Properties: style.TransitionOpacity, Duration: 150 * ms, Easing: standard},
		"transition-shadow":             {Properties: style.TransitionShadow, Duration: 150 * ms, Easing: standard},
		"transition-transform":          {Properties: style.TransitionTranslate, Duration: 150 * ms, Easing: standard},
		"transition-[color,box-shadow]": {Properties: style.TransitionColor | style.TransitionShadow, Duration: 150 * ms, Easing: standard},
		"transition transition-none":    {Duration: 150 * ms, Easing: standard},
		"transition duration-300":       {Properties: style.TransitionAll, Duration: 300 * ms, Easing: standard},
		"duration-300 transition":       {Properties: style.TransitionAll, Duration: 300 * ms, Easing: standard},
		"transition delay-100":          {Properties: style.TransitionAll, Duration: 150 * ms, Delay: 100 * ms, Easing: standard},
		"transition ease-linear":        {Properties: style.TransitionAll, Duration: 150 * ms, Easing: style.Easing{X2: 1, Y2: 1}},
		"transition ease-in":            {Properties: style.TransitionAll, Duration: 150 * ms, Easing: style.Easing{X1: 0.4, X2: 1, Y2: 1}},
		"transition ease-out":           {Properties: style.TransitionAll, Duration: 150 * ms, Easing: style.Easing{X2: 0.2, Y2: 1}},
		"transition ease-in-out":        {Properties: style.TransitionAll, Duration: 150 * ms, Easing: standard},
		"transition ease-[cubic-bezier(0.1,0.2,0.3,0.4)]": {Properties: style.TransitionAll, Duration: 150 * ms, Easing: style.Easing{X1: 0.1, Y1: 0.2, X2: 0.3, Y2: 0.4}},
	}
	for classes, want := range cases {
		if got := sheet.Compute(style.ComputedStyle{}, strings.Fields(classes)).Transition; got != want {
			t.Errorf("%q: %+v, want %+v", classes, got, want)
		}
	}
	animations := map[string]style.Animation{
		"":               {Iterations: 1, Easing: ease},
		"animate-spin":   {Keyframes: style.KeyframesSpin, Duration: time.Second, Easing: style.Easing{X2: 1, Y2: 1}, Iterations: 1, Infinite: true},
		"animate-ping":   {Keyframes: style.KeyframesPing, Duration: time.Second, Easing: style.Easing{X2: 0.2, Y2: 1}, Iterations: 1, Infinite: true},
		"animate-pulse":  {Keyframes: style.KeyframesPulse, Duration: 2 * time.Second, Easing: style.Easing{X1: 0.4, X2: 0.6, Y2: 1}, Iterations: 1, Infinite: true},
		"animate-bounce": {Keyframes: style.KeyframesBounce, Duration: time.Second, Easing: ease, Iterations: 1, Infinite: true},
		"animate-none":   {Iterations: 1, Easing: ease},
	}
	for classes, want := range animations {
		if got := sheet.Compute(style.ComputedStyle{}, strings.Fields(classes)).Animation; got != want {
			t.Errorf("%q: %+v, want %+v", classes, got, want)
		}
	}
	quiet(t, hasPrefix("transition", "duration-", "ease-", "delay-", "animate-"))
	for name, src := range map[string]string{
		"unknown keyframes": ".a { animation: wiggle 1s infinite; }",
		"two durations":     ".a { animation: spin 1s 2s; }",
		"steps easing":      ".a { transition-timing-function: steps(4); }",
		"width transition":  ".a { transition-property: width; }",
	} {
		if _, warnings, err := Compile("@layer utilities { " + src + " }"); err != nil || len(warnings) != 1 {
			t.Errorf("%s: error %v warnings %v, want one warning", name, err, warnings)
		}
	}
}

func TestRing(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	current := color.Color{Kind: color.Current}
	ring, err := color.Parse("oklch(70.5% 0.015 286.067)")
	if err != nil {
		t.Fatal(err)
	}
	white := rgba(255, 255, 255, 255)
	cases := []struct {
		classes      string
		outer, inset []style.Shadow
	}{
		{"ring", []style.Shadow{{Spread: 1, Color: current}}, nil},
		{"ring-1", []style.Shadow{{Spread: 1, Color: current}}, nil},
		{"ring-2", []style.Shadow{{Spread: 2, Color: current}}, nil},
		{"ring-[3px]", []style.Shadow{{Spread: 3, Color: current}}, nil},
		{"ring-ring", nil, nil},
		{"ring-2 ring-ring", []style.Shadow{{Spread: 2, Color: ring}}, nil},
		{"ring-2 ring-offset-2", []style.Shadow{{Spread: 2, Color: white}, {Spread: 4, Color: current}}, nil},
		{"ring-2 ring-offset-2 ring-offset-background", []style.Shadow{{Spread: 2, Color: white}, {Spread: 4, Color: current}}, nil},
		{"ring-2 ring-inset", nil, []style.Shadow{{Spread: 2, Color: current, Inset: true}}},
		{"shadow-md ring-2", append([]style.Shadow{{Spread: 2, Color: current}}, md(black(26))...), nil},
		{"shadow-md ring-2 shadow-none", []style.Shadow{{Spread: 2, Color: current}}, nil},
		{"inset-shadow-sm ring-2 ring-inset", nil, append(insetSm(black(13)), style.Shadow{Spread: 2, Color: current, Inset: true})},
	}
	for _, tc := range cases {
		got := sheet.Compute(style.ComputedStyle{}, strings.Fields(tc.classes))
		t.Logf("%s: outer %+v inset %+v", tc.classes, got.Shadows, got.InsetShadows)
		if !reflect.DeepEqual(got.Shadows, tc.outer) || !reflect.DeepEqual(got.InsetShadows, tc.inset) {
			t.Errorf("%s: outer %+v inset %+v, want %+v %+v", tc.classes, got.Shadows, got.InsetShadows, tc.outer, tc.inset)
		}
	}
	focused := style.NodeState{States: style.StateFocusVisible}
	if got := sheet.ComputeState(style.ComputedStyle{}, strings.Fields("ring-ring focus-visible:ring-2"), focused).Shadows; !reflect.DeepEqual(got, []style.Shadow{{Spread: 2, Color: ring}}) {
		t.Errorf("ring-ring then focus-visible:ring-2: %+v, want the ring colour kept", got)
	}
	half := ring
	half.RGBA.A = 128
	if got := sheet.ComputeState(style.ComputedStyle{}, strings.Fields("focus-visible:ring-ring/50 focus-visible:ring-[3px]"), focused).Shadows; !reflect.DeepEqual(got, []style.Shadow{{Spread: 3, Color: half}}) {
		t.Errorf("shadcn focus ring: %+v", got)
	}
	var rose theme.Theme
	for _, th := range theme.Builtin() {
		if th.Name == "rose" && th.Scheme == theme.Dark {
			rose = th
		}
	}
	roseRing := rose.Tokens[theme.Ring]
	themed := sheet.WithTheme(&rose)
	if got := themed.Compute(style.ComputedStyle{}, strings.Fields("ring-2 ring-ring")).Shadows; !reflect.DeepEqual(got, []style.Shadow{{Spread: 2, Color: roseRing}}) {
		t.Errorf("ring-ring under rose dark: %+v, want %+v", got, roseRing)
	}
	if got := themed.Compute(style.ComputedStyle{}, []string{"shadow-[0_0_0_1px_var(--color-ring)]"}).Shadows; len(got) != 1 || got[0].Color != roseRing {
		t.Errorf("shadow-[0_0_0_1px_var(--color-ring)] under rose dark: %+v, want colour %+v", got, roseRing)
	}
	if got := sheet.Compute(style.ComputedStyle{}, []string{"shadow-[0_0_0_1px_var(--color-ring)]"}).Shadows; len(got) != 1 || got[0].Color != ring {
		t.Errorf("shadow-[0_0_0_1px_var(--color-ring)] with no theme: %+v, want the build ring %+v", got, ring)
	}
	quiet(t, func(class string) bool { return strings.Contains(class, "ring") })
	rules, warnings, err := Compile("@layer utilities { .r { --tw-ring-shadow: 0 0 0 2px #000; box-shadow: var(--tw-shadow, 0 0 #0000), var(--tw-ring-shadow); } }")
	if err != nil || len(warnings) != 0 {
		t.Fatal(err, warnings)
	}
	literal, err := style.NewSheet(1, rules)
	if err != nil {
		t.Fatal(err)
	}
	if got := literal.Compute(style.ComputedStyle{}, []string{"r"}).Shadows; !reflect.DeepEqual(got, []style.Shadow{{Spread: 2, Color: black(255)}}) {
		t.Errorf("a ring with a literal colour: %+v", got)
	}
}

func TestFit(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	fit := style.Length{Unit: style.FitContent}
	got := sheet.Compute(style.ComputedStyle{}, strings.Fields("w-fit min-w-fit max-w-fit h-fit"))
	if got.Width != fit || got.MinWidth != fit || got.MaxWidth != fit || got.Height != fit {
		t.Errorf("fit: width %+v min %+v max %+v height %+v, want fit-content", got.Width, got.MinWidth, got.MaxWidth, got.Height)
	}
	quiet(t, func(class string) bool { return strings.HasSuffix(class, "-fit") })
}

func TestSideBorder(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	one, zero := cells(1), cells(0)
	for classes, want := range map[string]style.Edges{
		"border-t":          {Top: one, Right: zero, Bottom: zero, Left: zero},
		"border-r":          {Top: zero, Right: one, Bottom: zero, Left: zero},
		"border-b":          {Top: zero, Right: zero, Bottom: one, Left: zero},
		"border-l":          {Top: zero, Right: zero, Bottom: zero, Left: one},
		"border-x":          {Top: zero, Right: one, Bottom: zero, Left: one},
		"border-y":          {Top: one, Right: zero, Bottom: one, Left: zero},
		"border-t border-l": {Top: one, Right: zero, Bottom: zero, Left: one},
	} {
		got := sheet.Compute(style.ComputedStyle{}, strings.Fields(classes))
		if got.BorderWidth != want || got.BorderStyle != style.BorderSingle {
			t.Errorf("%s: width %+v style %d, want %+v single", classes, got.BorderWidth, got.BorderStyle, want)
		}
	}
	quiet(t, hasPrefix("border-"))
}

func TestWhiteSpace(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	for classes, want := range map[string]style.WhiteSpace{
		"":                        style.WhiteSpaceNormal,
		"whitespace-nowrap":       style.WhiteSpaceNowrap,
		"whitespace-pre":          style.WhiteSpacePre,
		"whitespace-pre-wrap":     style.WhiteSpacePreWrap,
		"whitespace-pre-line":     style.WhiteSpacePreWrap,
		"whitespace-break-spaces": style.WhiteSpacePreWrap,
		"whitespace-normal":       style.WhiteSpaceNormal,
	} {
		if got := sheet.Compute(style.ComputedStyle{}, strings.Fields(classes)).WhiteSpace; got != want {
			t.Errorf("%q: %d, want %d", classes, got, want)
		}
	}
	truncated := sheet.Compute(style.ComputedStyle{}, []string{"truncate"})
	if truncated.WhiteSpace != style.WhiteSpaceNowrap || truncated.TextOverflow != style.TextOverflowEllipsis || truncated.OverflowX != style.OverflowHidden || truncated.OverflowY != style.OverflowHidden {
		t.Errorf("truncate: white-space %d text-overflow %d overflow %d %d", truncated.WhiteSpace, truncated.TextOverflow, truncated.OverflowX, truncated.OverflowY)
	}
	child := sheet.Compute(truncated, nil)
	if child.WhiteSpace != style.WhiteSpaceNowrap || child.TextOverflow != style.TextOverflowClip {
		t.Errorf("child of truncate: white-space %d text-overflow %d, want nowrap inherited and clip", child.WhiteSpace, child.TextOverflow)
	}
	if got := sheet.Compute(style.ComputedStyle{}, strings.Fields("truncate text-clip")).TextOverflow; got != style.TextOverflowClip {
		t.Errorf("truncate text-clip: %d, want clip", got)
	}
	if got := sheet.Compute(style.ComputedStyle{}, []string{"text-ellipsis"}).TextOverflow; got != style.TextOverflowEllipsis {
		t.Errorf("text-ellipsis: %d", got)
	}
	warned := appWarnings(t, "whitespace-")
	if want := map[string]Category{"whitespace-pre-line": Approximated, "whitespace-break-spaces": Approximated}; !reflect.DeepEqual(warned, want) {
		t.Errorf("white-space warnings %v, want %v", warned, want)
	}
	quiet(t, func(class string) bool {
		return class == "truncate" || strings.HasPrefix(class, "text-e") || class == "text-clip"
	})
}

func TestAspect(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	for classes, want := range map[string]float64{"": 0, "aspect-square": 1, "aspect-video": 16.0 / 9, "aspect-[4/3]": 4.0 / 3, "aspect-auto": 0} {
		if got := sheet.Compute(style.ComputedStyle{}, strings.Fields(classes)).AspectRatio; got != want {
			t.Errorf("%q: %v, want %v", classes, got, want)
		}
	}
	quiet(t, hasPrefix("aspect-"))
}
