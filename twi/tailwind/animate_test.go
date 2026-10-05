package tailwind

import (
	"strings"
	"testing"
	"time"

	"github.com/pehcastro/twind/twi/style"
)

func still() style.Pose { return style.Pose{Opacity: 1, Scale: 1} }

func percent(n float64) style.Length { return style.Length{Unit: style.Percent, Value: n} }

func animated(t *testing.T, cases map[string]style.Animation, node style.NodeState) {
	t.Helper()
	sheet, _ := compileFixture(t, appFixture)
	for classes, want := range cases {
		if got := sheet.ComputeState(style.ComputedStyle{}, strings.Fields(classes), node).Animation; got != want {
			t.Errorf("%q:\n got %+v\nwant %+v", classes, got, want)
		}
	}
}

func TestAnimateIn(t *testing.T) {
	ease, ms := style.Easing{X1: 0.25, Y1: 0.1, X2: 0.25, Y2: 1}, time.Millisecond
	enter := func(p style.Pose) style.Animation {
		return style.Animation{Keyframes: style.KeyframesEnter, Duration: 150 * ms, Easing: ease, Iterations: 1, Enter: p, Exit: still()}
	}
	with := func(edit func(*style.Pose)) style.Animation {
		p := still()
		edit(&p)
		return enter(p)
	}
	dialog := with(func(p *style.Pose) { p.Opacity, p.Scale = 0, 0.95 })
	dialog.Duration = 200 * ms
	animated(t, map[string]style.Animation{
		"animate-in":                         enter(still()),
		"animate-in fade-in-0":               with(func(p *style.Pose) { p.Opacity = 0 }),
		"animate-in fade-in":                 with(func(p *style.Pose) { p.Opacity = 0 }),
		"animate-in zoom-in-95":              with(func(p *style.Pose) { p.Scale = 0.95 }),
		"animate-in zoom-in":                 with(func(p *style.Pose) { p.Scale = 0 }),
		"animate-in slide-in-from-bottom-2":  with(func(p *style.Pose) { p.TranslateY = cells(2) }),
		"animate-in slide-in-from-top":       with(func(p *style.Pose) { p.TranslateY = percent(-100) }),
		"animate-in slide-in-from-top-[48%]": with(func(p *style.Pose) { p.TranslateY = percent(-48) }),
		"animate-in slide-in-from-left-4":    with(func(p *style.Pose) { p.TranslateX = cells(-4) }),
		"animate-in spin-in-90":              with(func(p *style.Pose) { p.Degrees = 90 }),
		"data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95 duration-200": dialog,
	}, style.NodeState{Attrs: []style.Attr{{Name: "data-state", Value: "open"}}})
	animated(t, map[string]style.Animation{
		"data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95 duration-200": {Iterations: 1, Easing: ease, Enter: still(), Exit: still()},
	}, style.NodeState{Attrs: []style.Attr{{Name: "data-state", Value: "closed"}}})
	quiet(t, hasPrefix("animate-in", "fade-", "zoom-", "slide-in-from-bottom", "slide-in-from-top", "slide-in-from-left", "slide-out-to", "spin-in", "delay-", "animation-duration-", "repeat-", "fill-mode-", "data-[state=open]:", "data-[state=closed]:"))
	warned := appWarnings(t, "")
	for _, class := range []string{"blur-in", "slide-in-from-start-2", "animate-accordion-down", "animate-caret-blink", "direction-reverse"} {
		if _, ok := warned[class]; !ok {
			t.Errorf("%s: no warning, want one naming the class", class)
		}
	}
}

func TestAnimateOut(t *testing.T) {
	ease, ms := style.Easing{X1: 0.25, Y1: 0.1, X2: 0.25, Y2: 1}, time.Millisecond
	exit := func(edit func(*style.Pose)) style.Animation {
		p := still()
		edit(&p)
		return style.Animation{Keyframes: style.KeyframesExit, Duration: 150 * ms, Easing: ease, Iterations: 1, Enter: still(), Exit: p}
	}
	timed := exit(func(*style.Pose) {})
	timed.Duration, timed.Delay, timed.Fill, timed.Infinite, timed.Easing = 300*ms, 100*ms, style.FillForwards, true, style.Easing{X2: 0.2, Y2: 1}
	animated(t, map[string]style.Animation{
		"animate-out fade-out-0":              exit(func(p *style.Pose) { p.Opacity = 0 }),
		"animate-out fade-out-50":             exit(func(p *style.Pose) { p.Opacity = 0.5 }),
		"animate-out slide-out-to-left-1/2":   exit(func(p *style.Pose) { p.TranslateX = percent(-50) }),
		"animate-out slide-out-to-right-full": exit(func(p *style.Pose) { p.TranslateX = percent(100) }),
		"data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95":               exit(func(p *style.Pose) { p.Opacity, p.Scale = 0, 0.95 }),
		"animate-out delay-100 fill-mode-forwards repeat-infinite animation-duration-300 ease-out":                     timed,
		"data-[state=closed]:animate-out delay-100 fill-mode-forwards repeat-infinite animation-duration-300 ease-out": timed,
	}, style.NodeState{Attrs: []style.Attr{{Name: "data-state", Value: "closed"}}})
	for name, src := range map[string]string{
		"reversed direction": ".a { animation: exit 1s reverse; }",
		"three durations":    ".a { animation: exit 1s 2s 3s; }",
		"exit blur":          ".a { --tw-exit-blur: 4px; }",
		"rotation in turns":  ".a { --tw-enter-rotate: 1turn; }",
	} {
		if _, warnings, err := Compile("@layer utilities { " + src + " }"); err != nil || len(warnings) != 1 {
			t.Errorf("%s: error %v warnings %v, want one warning", name, err, warnings)
		}
	}
}

func TestScale(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	for classes, want := range map[string][2]float64{
		"":                    {1, 1},
		"scale-95":            {0.95, 0.95},
		"scale-100":           {1, 1},
		"scale-[1.02]":        {1.02, 1.02},
		"-scale-50":           {-0.5, -0.5},
		"scale-x-50":          {0.5, 1},
		"scale-95 scale-x-50": {0.5, 0.95},
	} {
		got := sheet.Compute(style.ComputedStyle{}, strings.Fields(classes))
		if [2]float64{got.ScaleX, got.ScaleY} != want {
			t.Errorf("%q: scale %v %v, want %v", classes, got.ScaleX, got.ScaleY, want)
		}
	}
	quiet(t, hasPrefix("scale-", "-scale-"))
	for name, src := range map[string]string{
		"z axis":   ".a { scale: 1 2 3; }",
		"a length": ".a { scale: 2px; }",
	} {
		if _, warnings, err := Compile("@layer utilities { " + src + " }"); err != nil || len(warnings) != 1 {
			t.Errorf("%s: error %v warnings %v, want one warning", name, err, warnings)
		}
	}
	rules, warnings, err := Compile("@layer utilities { .a { scale: none; } }")
	if err != nil || len(warnings) != 0 || len(rules) != 1 {
		t.Fatalf("scale: none: rules %+v warnings %v error %v", rules, warnings, err)
	}
	sheet, err = style.NewSheet(1, rules)
	if err != nil {
		t.Fatal(err)
	}
	if got := sheet.Compute(style.ComputedStyle{}, []string{"a"}); got.ScaleX != 1 || got.ScaleY != 1 {
		t.Errorf("scale: none: %v %v, want 1 1", got.ScaleX, got.ScaleY)
	}
}
