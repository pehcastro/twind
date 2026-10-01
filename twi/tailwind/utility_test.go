package tailwind

import (
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/css"
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
		"transition-transform":          {Properties: style.TransitionTranslate | style.TransitionScale, Duration: 150 * ms, Easing: standard},
		"transition-[scale]":            {Properties: style.TransitionScale, Duration: 150 * ms, Easing: standard},
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
		"":               {Iterations: 1, Easing: ease, Enter: still(), Exit: still()},
		"animate-spin":   {Keyframes: style.KeyframesSpin, Duration: time.Second, Easing: style.Easing{X2: 1, Y2: 1}, Iterations: 1, Infinite: true, Enter: still(), Exit: still()},
		"animate-ping":   {Keyframes: style.KeyframesPing, Duration: time.Second, Easing: style.Easing{X2: 0.2, Y2: 1}, Iterations: 1, Infinite: true, Enter: still(), Exit: still()},
		"animate-pulse":  {Keyframes: style.KeyframesPulse, Duration: 2 * time.Second, Easing: style.Easing{X1: 0.4, X2: 0.6, Y2: 1}, Iterations: 1, Infinite: true, Enter: still(), Exit: still()},
		"animate-bounce": {Keyframes: style.KeyframesBounce, Duration: time.Second, Easing: ease, Iterations: 1, Infinite: true, Enter: still(), Exit: still()},
		"animate-none":   {Iterations: 1, Easing: ease, Enter: still(), Exit: still()},
	}
	for classes, want := range animations {
		if got := sheet.Compute(style.ComputedStyle{}, strings.Fields(classes)).Animation; got != want {
			t.Errorf("%q: %+v, want %+v", classes, got, want)
		}
	}
	quiet(t, func(class string) bool {
		return hasPrefix("transition", "duration-", "ease-", "delay-", "animate-")(class) && class != "animate-accordion-down" && class != "animate-caret-blink"
	})
	for name, src := range map[string]string{
		"unknown keyframes": ".a { animation: wiggle 1s infinite; }",
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
	twind := theme.Default()
	twindRing := twind.Tokens[theme.Ring]
	themed := sheet.WithTheme(&twind)
	if got := themed.Compute(style.ComputedStyle{}, strings.Fields("ring-2 ring-ring")).Shadows; !reflect.DeepEqual(got, []style.Shadow{{Spread: 2, Color: twindRing}}) {
		t.Errorf("ring-ring under twind dark: %+v, want %+v", got, twindRing)
	}
	if got := themed.Compute(style.ComputedStyle{}, []string{"shadow-[0_0_0_1px_var(--color-ring)]"}).Shadows; len(got) != 1 || got[0].Color != twindRing {
		t.Errorf("shadow-[0_0_0_1px_var(--color-ring)] under twind dark: %+v, want colour %+v", got, twindRing)
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
		"whitespace-pre-line":     style.WhiteSpacePreLine,
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
	if want := map[string]Category{"whitespace-break-spaces": Approximated}; !reflect.DeepEqual(warned, want) {
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

func TestColorMix(t *testing.T) {
	const halo40 = "dark:shadow-[0_0_0_1px_var(--color-destructive),0_0_0_3px_color-mix(in_oklab,var(--color-destructive)_40%,transparent)]"
	const halo20 = "shadow-[0_0_0_1px_var(--color-destructive),0_0_0_3px_color-mix(in_oklab,var(--color-destructive)_20%,transparent)]"
	red400, err := color.Parse("oklch(70.4% 0.191 22.216)")
	if err != nil {
		t.Fatal(err)
	}
	red600, err := color.Parse("oklch(57.7% 0.245 27.325)")
	if err != nil {
		t.Fatal(err)
	}
	faded := func(c color.Color, mix float64) color.Color {
		c.RGBA.A = uint8(math.Round(float64(c.RGBA.A) * mix / 100))
		return c
	}
	halo := func(c color.Color, mix float64) []style.Shadow {
		return []style.Shadow{
			{Spread: 1, Color: c, Tintable: true, Token: theme.Destructive, Mix: 100},
			{Spread: 3, Color: faded(c, mix), Tintable: true, Token: theme.Destructive, Mix: mix},
		}
	}
	sheet, _ := compileFixture(t, appFixture)
	dark := theme.Default()
	light := dark.WithScheme(theme.Light)
	destructive := dark.Tokens[theme.Destructive]
	cases := []struct {
		name    string
		sheet   style.Sheet
		classes string
		want    []style.Shadow
	}{
		{"/40 under twind dark", sheet.WithTheme(&dark), halo40, halo(destructive, 40)},
		{"/40 under twind light", sheet.WithTheme(&light), halo40, nil},
		{"/20 with no theme", sheet, halo20, halo(red600, 20)},
		{"/20 under twind dark", sheet.WithTheme(&dark), halo20, halo(destructive, 20)},
	}
	for _, tc := range cases {
		if got := tc.sheet.Compute(style.ComputedStyle{}, []string{tc.classes}).Shadows; !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %+v, want %+v", tc.name, got, tc.want)
		}
	}
	rules, _, err := Compile(compilerCorpus)
	if err != nil {
		t.Fatal(err)
	}
	var built []style.Shadow
	for _, r := range rules {
		for _, d := range r.Decls {
			if r.Class == halo40 && d.Property == style.PropShadow {
				built = d.Shadows
			}
		}
	}
	if want := halo(red400, 40); !reflect.DeepEqual(built, want) {
		t.Errorf("%s as built: %+v, want red-400 at 40%%: %+v", halo40, built, want)
	}
	if got := sheet.WithTheme(&dark).Compute(style.ComputedStyle{}, []string{"dark:bg-destructive/40"}).Background; got != faded(destructive, 40) {
		t.Errorf("dark:bg-destructive/40: %+v, want the dark destructive token at 40%%", got)
	}
	quiet(t, func(class string) bool {
		return class == halo40 || class == halo20 || class == "dark:bg-destructive/40"
	})
}

func TestNoSilentDrop(t *testing.T) {
	src, err := os.ReadFile(appFixture + "/output.css")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := css.Parse(string(src))
	if err != nil {
		t.Fatal(err)
	}
	rules, warnings, err := Compile(string(src))
	if err != nil {
		t.Fatal(err)
	}
	heard := map[string]bool{}
	for _, r := range rules {
		heard[r.Class] = true
	}
	for _, w := range warnings {
		heard[w.Class] = true
	}
	var walk func(nodes []css.Node, utilities bool)
	walk = func(nodes []css.Node, utilities bool) {
		for _, n := range nodes {
			switch n := n.(type) {
			case css.AtRule:
				walk(n.Block, utilities || n.Name == "layer" && text(n.Prelude) == "utilities")
			case css.Rule:
				for _, sel := range selectorList(n.Selector) {
					if r, _ := selector(sel, style.Condition{}); utilities && !heard[r.Class] {
						t.Errorf("%s: no rule and no warning", r.Class)
					}
				}
			}
		}
	}
	walk(nodes, false)
	rules, warnings, err = Compile("@layer utilities { .a { box-shadow: var(--tw-inset-shadow), var(--tw-shadow); } }")
	if err != nil || len(rules) != 0 || len(warnings) != 1 {
		t.Errorf("a box-shadow reading only unset layers: rules %+v warnings %v, want one warning", rules, warnings)
	}
}

func TestFlexWrap(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	for classes, want := range map[string]style.Wrapping{
		"":                      style.NoWrap,
		"flex-wrap":             style.Wrap,
		"flex-wrap-reverse":     style.WrapReverse,
		"flex-nowrap":           style.NoWrap,
		"flex-nowrap flex-wrap": style.Wrap,
	} {
		if got := sheet.Compute(style.ComputedStyle{}, strings.Fields(classes)).Wrap; got != want {
			t.Errorf("%q: %d, want %d", classes, got, want)
		}
	}
	if got := sheet.Compute(sheet.Compute(style.ComputedStyle{}, []string{"flex-wrap"}), nil).Wrap; got != style.NoWrap {
		t.Errorf("child of flex-wrap: %d, want no wrap: flex-wrap is not inherited", got)
	}
	if _, warnings, err := Compile("@layer utilities { .a { flex-wrap: sideways; } }"); err != nil || len(warnings) != 1 {
		t.Errorf("flex-wrap: sideways: error %v warnings %v, want one warning", err, warnings)
	}
	quiet(t, hasPrefix("flex-wrap", "flex-nowrap"))
}
