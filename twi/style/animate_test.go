package style_test

import (
	"strings"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi/style"
)

func TestAnimateReadsTailwindDuration(t *testing.T) {
	sheet := appSheet(t)
	ms, ease := time.Millisecond, style.Easing{X1: 0.25, Y1: 0.1, X2: 0.25, Y2: 1}
	for classes, want := range map[string]time.Duration{
		"":                          0,
		"duration-200":              0,
		"animate-in":                150 * ms,
		"animate-in duration-200":   200 * ms,
		"duration-200 animate-out":  200 * ms,
		"animate-spin duration-200": time.Second,
		"animate-in duration-200 animation-duration-300": 300 * ms,
	} {
		if got := sheet.Compute(style.ComputedStyle{}, strings.Fields(classes)).Animation.Duration; got != want {
			t.Errorf("%q: animation duration %v, want %v", classes, got, want)
		}
	}
	for classes, want := range map[string]style.Easing{
		"animate-in":            ease,
		"animate-in ease-out":   {X2: 0.2, Y2: 1},
		"animate-spin ease-out": {X2: 1, Y2: 1},
	} {
		if got := sheet.Compute(style.ComputedStyle{}, strings.Fields(classes)).Animation.Easing; got != want {
			t.Errorf("%q: animation easing %+v, want %+v", classes, got, want)
		}
	}
	parent := sheet.Compute(style.ComputedStyle{}, []string{"duration-200", "ease-out"})
	if got := sheet.Compute(parent, []string{"animate-in"}).Animation; got.Duration != 150*ms || got.Easing != ease {
		t.Errorf("animate-in under a duration-200 ease-out parent: %v %+v, want 150ms and ease: the variables do not inherit", got.Duration, got.Easing)
	}
	if got := sheet.Compute(style.ComputedStyle{}, strings.Fields("transition duration-300 ease-out")).Transition; got.Duration != 300*ms || got.Easing != (style.Easing{X2: 0.2, Y2: 1}) {
		t.Errorf("transition duration-300 ease-out: %+v, want 300ms ease-out", got)
	}
}

func TestAnimatePoseStartsStill(t *testing.T) {
	sheet := appSheet(t)
	still := style.Pose{Opacity: 1, Scale: 1}
	closed := style.NodeState{Attrs: []style.Attr{{Name: "data-state", Value: "closed"}}}
	for _, classes := range []string{"", "animate-in", "animate-out", "data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95"} {
		if got := sheet.ComputeState(style.ComputedStyle{}, strings.Fields(classes), closed).Animation; got.Enter != still || got.Exit != still {
			t.Errorf("%q closed: enter %+v exit %+v, want both still", classes, got.Enter, got.Exit)
		}
	}
	parent := sheet.ComputeState(style.ComputedStyle{}, strings.Fields("animate-out fade-out-0 zoom-out-95"), closed)
	if got := sheet.Compute(parent, nil).Animation; got.Keyframes != style.KeyframesNone || got.Exit != still {
		t.Errorf("child of a fading parent: %+v, want no animation and a still exit pose", got)
	}
}

func TestScaleStartsAtOne(t *testing.T) {
	sheet := appSheet(t)
	if got := sheet.Compute(style.ComputedStyle{}, nil); got.ScaleX != 1 || got.ScaleY != 1 {
		t.Errorf("no class: scale %v %v, want 1 1", got.ScaleX, got.ScaleY)
	}
	parent := sheet.Compute(style.ComputedStyle{}, []string{"scale-95"})
	if got := sheet.Compute(parent, nil); got.ScaleX != 1 || got.ScaleY != 1 {
		t.Errorf("child of scale-95: scale %v %v, want 1 1: scale does not inherit", got.ScaleX, got.ScaleY)
	}
}
