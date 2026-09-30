package motion

import (
	"math"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/style"
)

var straight = style.Easing{X2: 1, Y2: 1}

func cells(v float64) style.Length   { return style.Length{Unit: style.Cells, Value: v} }
func percent(v float64) style.Length { return style.Length{Unit: style.Percent, Value: v} }

func samePose(a, b Pose) bool {
	close := func(x, y float64) bool { return math.Abs(x-y) < 1e-9 }
	length := func(x, y style.Length) bool { return close(x.Value, y.Value) && (x.Unit == y.Unit || x.Value == 0) }
	return close(a.Opacity, b.Opacity) && close(a.Scale, b.Scale) && close(a.Turn, b.Turn) && length(a.TranslateX, b.TranslateX) && length(a.TranslateY, b.TranslateY)
}

func TestKeyframesEnterExit(t *testing.T) {
	still := style.Pose{Opacity: 1, Scale: 1}
	enter := style.Animation{Keyframes: style.KeyframesEnter, Duration: 200 * ms, Easing: straight, Iterations: 1, Enter: style.Pose{Opacity: 0, Scale: 0.95, TranslateY: cells(2)}, Exit: still}
	exit := style.Animation{Keyframes: style.KeyframesExit, Duration: 150 * ms, Easing: straight, Iterations: 1, Enter: still, Exit: style.Pose{Opacity: 0, Scale: 0.95, TranslateX: percent(-100)}}
	with := func(a style.Animation, change func(*style.Animation)) style.Animation {
		change(&a)
		return a
	}
	forwards := with(exit, func(a *style.Animation) { a.Fill = style.FillForwards })
	delayed := with(enter, func(a *style.Animation) { a.Duration, a.Delay = 100*ms, 100*ms })
	backwards := with(delayed, func(a *style.Animation) { a.Fill = style.FillBackwards })
	twice := with(forwards, func(a *style.Animation) { a.Iterations = 2 })
	half := with(forwards, func(a *style.Animation) { a.Iterations = 1.5 })
	rest := Pose{Opacity: 1, Scale: 1}
	for _, c := range []struct {
		name    string
		a       style.Animation
		opacity float64
		at      time.Duration
		want    Pose
	}{
		{"enter 0 ms", enter, 1, 0, Pose{Opacity: 0, Scale: 0.95, TranslateY: cells(2)}},
		{"enter 100 ms", enter, 1, 100 * ms, Pose{Opacity: 0.5, Scale: 0.975, TranslateY: cells(1)}},
		{"enter 100 ms over opacity 0.8", enter, 0.8, 100 * ms, Pose{Opacity: 0.4, Scale: 0.975, TranslateY: cells(1)}},
		{"enter 200 ms", enter, 1, 200 * ms, rest},
		{"enter before mount", enter, 1, -ms, rest},
		{"exit 0 ms", exit, 1, 0, rest},
		{"exit 75 ms", exit, 1, 75 * ms, Pose{Opacity: 0.5, Scale: 0.975, TranslateX: percent(-50)}},
		{"exit 150 ms, fill none", exit, 1, 150 * ms, rest},
		{"exit 150 ms, fill forwards", forwards, 1, 150 * ms, Pose{Opacity: 0, Scale: 0.95, TranslateX: percent(-100)}},
		{"exit 1 s, fill forwards", forwards, 1, time.Second, Pose{Opacity: 0, Scale: 0.95, TranslateX: percent(-100)}},
		{"exit twice, 225 ms", twice, 1, 225 * ms, Pose{Opacity: 0.5, Scale: 0.975, TranslateX: percent(-50)}},
		{"exit twice, forwards end", twice, 1, time.Second, Pose{Opacity: 0, Scale: 0.95, TranslateX: percent(-100)}},
		{"exit 1.5 times, forwards end", half, 1, time.Second, Pose{Opacity: 0.5, Scale: 0.975, TranslateX: percent(-50)}},
		{"delayed enter 50 ms, fill none", delayed, 1, 50 * ms, rest},
		{"delayed enter 150 ms", delayed, 1, 150 * ms, Pose{Opacity: 0.5, Scale: 0.975, TranslateY: cells(1)}},
		{"delayed enter 50 ms, fill backwards", backwards, 1, 50 * ms, Pose{Opacity: 0, Scale: 0.95, TranslateY: cells(2)}},
		{"delayed enter 200 ms, fill backwards", backwards, 1, 200 * ms, rest},
	} {
		got := Keyframe(c.a, c.opacity, c.at)
		t.Logf("%-38s %+v", c.name, got)
		if !samePose(got, c.want) {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestKeyframesEveryValueNoPanic(t *testing.T) {
	base := card(color.RGBA{A: 255}, 1)
	for k := style.KeyframesNone; k <= style.KeyframesExit; k++ {
		for _, a := range []style.Animation{
			{Keyframes: k},
			{Keyframes: k, Duration: time.Second, Iterations: 1, Enter: style.Pose{Opacity: 1, Scale: 1}, Exit: style.Pose{Opacity: 1, Scale: 1}},
			{Keyframes: k, Duration: time.Second, Infinite: true, Delay: -time.Second, Fill: style.FillBoth},
		} {
			next := base
			next.Animation = a
			var s Styles
			var shown Animated
			for _, at := range []time.Duration{-time.Second, 0, 500 * ms, time.Second, time.Hour} {
				Keyframe(a, 1, at)
				s.Frame(1, &next, &next, at, &shown)
				s.Wake()
				s.Closing(1)
			}
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("a Keyframes value past KeyframesExit did not panic: add it to this walk and to Keyframe")
		}
	}()
	Keyframe(style.Animation{Keyframes: style.KeyframesExit + 1, Duration: time.Second, Iterations: 1}, 1, 0)
}

func TestStylesEnterKeepsClockExitHolds(t *testing.T) {
	open, hovered, closed := card(color.RGBA{A: 255}, 1), card(color.RGBA{A: 255}, 1), card(color.RGBA{A: 255}, 1)
	open.Animation = style.Animation{Keyframes: style.KeyframesEnter, Duration: 200 * ms, Easing: straight, Iterations: 1, Enter: style.Pose{Opacity: 0, Scale: 0.95}, Exit: style.Pose{Opacity: 1, Scale: 1}}
	hovered.Animation = open.Animation
	hovered.Background = literal(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	closed.Animation = style.Animation{Keyframes: style.KeyframesExit, Duration: 150 * ms, Easing: straight, Iterations: 1, Enter: style.Pose{Opacity: 1, Scale: 1}, Exit: style.Pose{Opacity: 0, Scale: 1}}
	var s Styles
	if _, pose := show(&s, 1, nil, &open, time.Second); pose.Opacity != 0 || pose.Scale != 0.95 {
		t.Errorf("enter at mount pose %+v, want opacity 0 scale 0.95", pose)
	}
	var a Animated
	if !s.Frame(1, &open, &open, time.Second+50*ms, &a) || a.Pose.Scale == 1 {
		t.Errorf("an enter that only zooms reports no movement, pose %+v", a.Pose)
	}
	if _, pose := show(&s, 1, &open, &hovered, time.Second+100*ms); pose.Opacity != 0.5 {
		t.Errorf("enter restarted or stopped by a restyle: pose %+v at 100 ms, want opacity 0.5", pose)
	}
	if s.Frame(1, &hovered, &hovered, time.Second+300*ms, &a) {
		t.Errorf("finished enter reports movement, pose %+v", a.Pose)
	}
	if at, moving := s.Wake(); moving {
		t.Errorf("finished enter wakes at %v", at)
	}
	if _, pose := show(&s, 1, &hovered, &open, 2*time.Second); pose.Opacity != 1 || s.Closing(1) {
		t.Errorf("a restyle after the enter restarted it: pose %+v, closing %v", pose, s.Closing(1))
	}
	for _, step := range []struct {
		after   time.Duration
		opacity float64
		closing bool
	}{{0, 1, true}, {75 * ms, 0.5, true}, {149 * ms, -1, true}, {150 * ms, 1, false}, {time.Second, 1, false}} {
		_, pose := show(&s, 1, &open, &closed, 3*time.Second+step.after)
		open = closed
		t.Logf("exit %3d ms: opacity %.3f closing %v", step.after.Milliseconds(), pose.Opacity, s.Closing(1))
		if step.opacity >= 0 && math.Abs(pose.Opacity-step.opacity) > 1e-9 || s.Closing(1) != step.closing {
			t.Errorf("exit %v: pose %+v closing %v, want opacity %v closing %v", step.after, pose, s.Closing(1), step.opacity, step.closing)
		}
		if _, moving := s.Wake(); moving != step.closing {
			t.Errorf("exit %v: wake moving %v, want %v", step.after, moving, step.closing)
		}
	}
	s.Reduced = true
	delayed := closed
	delayed.Animation.Delay = time.Second
	show(&s, 2, nil, &delayed, 0)
	if s.Closing(2) {
		t.Error("reduced motion holds an exit")
	}
}

func TestStylesEnterDelayWakesAtStart(t *testing.T) {
	a := card(color.RGBA{A: 255}, 1)
	a.Animation = style.Animation{Keyframes: style.KeyframesEnter, Duration: 100 * ms, Delay: 100 * ms, Easing: straight, Iterations: 1, Enter: style.Pose{Opacity: 0, Scale: 1}, Exit: style.Pose{Opacity: 1, Scale: 1}}
	var s Styles
	if got, _ := show(&s, 1, nil, &a, 0); got.Opacity != 1 {
		t.Errorf("fill none during the delay shows opacity %v, want the base 1", got.Opacity)
	}
	if at, moving := s.Wake(); !moving || at != 100*ms {
		t.Errorf("during the delay wake %v moving %v, want 100ms true", at, moving)
	}
	if got, _ := show(&s, 1, &a, &a, 150*ms); got.Opacity != 0.5 {
		t.Errorf("delayed enter at 150 ms opacity %v, want 0.5", got.Opacity)
	}
}

func TestStylesScaleTransition(t *testing.T) {
	from := card(color.RGBA{A: 255}, 1)
	from.ScaleX, from.ScaleY = 1, 1
	from.Transition = style.Transition{Properties: style.TransitionScale, Duration: 200 * ms, Easing: straight}
	to := from
	to.ScaleX, to.ScaleY = 0.5, 0.25
	to.TranslateY = cells(4)
	var s Styles
	if got, _ := show(&s, 1, &from, &to, 0); got.TranslateY != cells(4) {
		t.Errorf("transition-[scale] moved translate: %v, want the target 4 cells at once", got.TranslateY)
	}
	if got, _ := show(&s, 1, &to, &to, 100*ms); got.ScaleX != 0.75 || got.ScaleY != 0.625 {
		t.Errorf("transition-[scale] scale at 100 ms %v %v, want 0.75 0.625", got.ScaleX, got.ScaleY)
	}
	if got, _ := show(&s, 1, &to, &to, 200*ms); got.ScaleX != 0.5 || got.ScaleY != 0.25 {
		t.Errorf("scale at the end %v %v", got.ScaleX, got.ScaleY)
	}
	idle(t, &s)
	for _, flag := range []style.TransitionProperty{style.TransitionColor, style.TransitionTranslate} {
		from.Transition.Properties, to.Transition.Properties = flag, flag
		if got, _ := show(&s, 2, &from, &to, 0); got.ScaleX != 0.5 {
			t.Errorf("transition %08b moved scale: %v", flag, got.ScaleX)
		}
		show(&s, 2, &to, &to, time.Second)
		idle(t, &s)
	}
}

func TestPresenceEnterExitDelayFill(t *testing.T) {
	p := Presence{Offset: Offset{Opacity: 0, Scale: 0.5, X: -4}, Duration: 100 * ms, Delay: 100 * ms, Easing: Linear()}
	if got := p.Enter(50 * ms); got != Still() {
		t.Errorf("enter during the delay, fill none: %+v, want still", got)
	}
	if got := p.Enter(150 * ms); got != (Offset{Opacity: 0.5, Scale: 0.75, X: -2}) {
		t.Errorf("enter at 150 ms: %+v", got)
	}
	if got := p.Exit(150 * ms); got != (Offset{Opacity: 0.5, Scale: 0.75, X: -2}) {
		t.Errorf("exit at 150 ms: %+v", got)
	}
	if p.Total() != 200*ms {
		t.Errorf("total %v, want 200ms", p.Total())
	}
	p.Fill = style.FillBackwards
	if got := p.Enter(50 * ms); got != p.Offset {
		t.Errorf("enter during the delay, fill backwards: %+v, want %+v", got, p.Offset)
	}
	p.Fill = style.FillForwards
	if got := p.Exit(time.Second); got != p.Offset {
		t.Errorf("exit after the end, fill forwards: %+v, want %+v", got, p.Offset)
	}
}
