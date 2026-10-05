package motion

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/style"
)

const ms = time.Millisecond

func oklch(t *testing.T, l, c, h float64) color.RGBA {
	t.Helper()
	parsed, err := color.Parse(fmt.Sprintf("oklch(%.6f %.6f %.6f)", l, c, h))
	if err != nil {
		t.Fatal(err)
	}
	return parsed.RGBA
}

func literal(c color.RGBA) color.Color { return color.Color{Kind: color.Literal, RGBA: c} }

func card(bg color.RGBA, opacity float64) style.ComputedStyle {
	return style.ComputedStyle{
		Opacity:     opacity,
		Background:  literal(bg),
		BorderColor: color.Color{Kind: color.Current},
		Transition:  style.Transition{Properties: style.TransitionAll, Duration: 200 * ms, Easing: style.Easing{X2: 1, Y2: 1}},
	}
}

func near(a, b color.RGBA, within int) bool {
	d := func(x, y uint8) bool { return int(x)-int(y) <= within && int(y)-int(x) <= within }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B) && d(a.A, b.A)
}

func show(s *Styles, key Key, prev, next *style.ComputedStyle, now time.Duration) (style.ComputedStyle, Pose) {
	var a Animated
	out := *next
	if s.Frame(key, prev, next, now, &a) {
		s.Overlay(&a, &out)
	}
	return out, a.Pose
}

func idle(t *testing.T, s *Styles) {
	t.Helper()
	if at, moving := s.Wake(); moving || len(s.entries) != 0 || slices.ContainsFunc(s.index, func(i int32) bool { return i != 0 }) {
		t.Errorf("wake %v moving %v, %d entries left", at, moving, len(s.entries))
	}
}

func TestStylesColourMidpointOKLab(t *testing.T) {
	zinc100, zinc900 := oklch(t, 0.967, 0.001, 286.375), oklch(t, 0.21, 0.006, 285.885)
	rad := math.Pi / 180
	ma := (0.001*math.Cos(286.375*rad) + 0.006*math.Cos(285.885*rad)) / 2
	mb := (0.001*math.Sin(286.375*rad) + 0.006*math.Sin(285.885*rad)) / 2
	want := oklch(t, (0.967+0.21)/2, math.Hypot(ma, mb), math.Atan2(mb, ma)/rad)
	a, b := card(zinc100, 1), card(zinc900, 1)
	var s Styles
	show(&s, 1, nil, &a, 0)
	if got, _ := show(&s, 1, &a, &b, 0); got.Background.RGBA != zinc100 {
		t.Errorf("at 0 ms %v, want %v", got.Background.RGBA, zinc100)
	}
	got, _ := show(&s, 1, &b, &b, 100*ms)
	if !near(got.Background.RGBA, want, 1) {
		t.Errorf("at 100 ms %v, want the OKLab midpoint %v", got.Background.RGBA, want)
	}
	mean := color.RGBA{R: uint8((int(zinc100.R) + int(zinc900.R)) / 2), G: uint8((int(zinc100.G) + int(zinc900.G)) / 2), B: uint8((int(zinc100.B) + int(zinc900.B)) / 2), A: 255}
	if near(want, mean, 4) {
		t.Fatalf("OKLab midpoint %v too close to the sRGB mean %v to tell them apart", want, mean)
	}
	if got.Background.Kind != color.Literal || got.BorderColor.Kind != color.Current || got.Color.Kind != color.Unset {
		t.Errorf("untouched properties changed: border %v, text %v", got.BorderColor, got.Color)
	}
	if at, moving := s.Wake(); !moving || at != 100*ms {
		t.Errorf("mid-flight wake %v moving %v, want 100ms true", at, moving)
	}
	if got, _ := show(&s, 1, &b, &b, 200*ms); got.Background != b.Background {
		t.Errorf("at 200 ms %v, want %v", got.Background, b.Background)
	}
	idle(t, &s)
	t.Logf("zinc-100 %v, zinc-900 %v, OKLab midpoint %v, sRGB mean %v", zinc100, zinc900, want, mean)
}

func TestStylesTransparentPremultiplied(t *testing.T) {
	red := color.RGBA{R: 220, G: 38, B: 38, A: 255}
	a, b := card(color.RGBA{}, 1), card(red, 1)
	var s Styles
	show(&s, 1, &a, &b, 0)
	got, _ := show(&s, 1, &b, &b, 100*ms)
	if want := (color.RGBA{R: 220, G: 38, B: 38, A: 128}); !near(got.Background.RGBA, want, 1) {
		t.Errorf("transparent to red at half %v, want %v", got.Background.RGBA, want)
	}
}

func TestStylesRetargetContinuous(t *testing.T) {
	zinc100, zinc900, red := oklch(t, 0.967, 0.001, 286.375), oklch(t, 0.21, 0.006, 285.885), oklch(t, 0.637, 0.237, 25.331)
	a, b, c := card(zinc100, 1), card(zinc900, 0), card(red, 0.5)
	var s Styles
	show(&s, 1, &a, &b, 0)
	before, _ := show(&s, 1, &b, &b, 80*ms)
	after, _ := show(&s, 1, &b, &c, 80*ms)
	if after.Background != before.Background || math.Abs(after.Opacity-before.Opacity) > 1e-12 {
		t.Errorf("retarget jumped: %v %v then %v %v", before.Background, before.Opacity, after.Background, after.Opacity)
	}
	if before.Opacity != 0.6 {
		t.Errorf("opacity at 80 ms %v, want 0.6", before.Opacity)
	}
	next, _ := show(&s, 1, &c, &c, 82*ms)
	if !near(next.Background.RGBA, after.Background.RGBA, 2) || math.Abs(next.Opacity-(0.6-0.1*2.0/200)) > 1e-9 {
		t.Errorf("2 ms after retarget %v %v, from %v %v", next.Background, next.Opacity, after.Background, after.Opacity)
	}
	if got, _ := show(&s, 1, &c, &c, 280*ms); got.Background != c.Background || got.Opacity != 0.5 {
		t.Errorf("retarget ends at %v %v, want %v 0.5 after the full 200 ms", got.Background, got.Opacity, c.Background)
	}
	idle(t, &s)
}

func TestStylesReversalShortened(t *testing.T) {
	a, b := card(color.RGBA{A: 255}, 1), card(color.RGBA{A: 255}, 0)
	var s Styles
	opacity := func(prev, next *style.ComputedStyle, now time.Duration) float64 {
		got, _ := show(&s, 1, prev, next, now)
		return got.Opacity
	}
	opacity(&a, &b, 0)
	checks := []struct {
		prev, next *style.ComputedStyle
		at         time.Duration
		want       float64
	}{
		{&b, &b, 50 * ms, 0.75},
		{&b, &a, 50 * ms, 0.75},
		{&a, &a, 75 * ms, 0.875},
		{&a, &b, 75 * ms, 0.875},
		{&b, &b, 75*ms + 87500*time.Microsecond, 0.4375},
		{&b, &b, 249 * ms, 0.875 * 1 / 175},
		{&b, &b, 250 * ms, 0},
	}
	for _, c := range checks {
		if got := opacity(c.prev, c.next, c.at); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("at %v opacity %v, want %v", c.at, got, c.want)
		}
	}
	idle(t, &s)
	a.Transition.Delay, b.Transition.Delay = 100*ms, 100*ms
	opacity(&a, &b, time.Second)
	if got := opacity(&b, &a, time.Second+50*ms); got != 1 || s.entries[0].tracks[propOpacity].duration != 0 {
		t.Errorf("reversal during the delay: %v over %v, want 1 and no duration", got, s.entries[0].tracks[propOpacity].duration)
	}
	opacity(&a, &a, time.Second+150*ms)
	idle(t, &s)
}

func TestStylesDelayDropReduced(t *testing.T) {
	a, b := card(color.RGBA{A: 255}, 1), card(color.RGBA{R: 255, A: 255}, 0)
	b.Transition.Delay = 100 * ms
	var s Styles
	if got, _ := show(&s, 1, &a, &b, 0); got.Background != a.Background || got.Opacity != 1 {
		t.Errorf("during the delay %v %v, want the start values", got.Background, got.Opacity)
	}
	if at, moving := s.Wake(); !moving || at != 100*ms {
		t.Errorf("delayed wake %v %v, want 100ms", at, moving)
	}
	if got, _ := show(&s, 1, &b, &b, 200*ms); got.Opacity != 0.5 {
		t.Errorf("half way after the delay %v, want 0.5", got.Opacity)
	}
	colours := b
	colours.Transition.Properties = style.TransitionBackground
	if got, _ := show(&s, 1, &b, &colours, 210*ms); got.Opacity != 0 || got.Background == b.Background {
		t.Errorf("opacity dropped from transition-property: %v, want 0 at once; background %v still moving", got.Opacity, got.Background)
	}
	s.Reduced = true
	if got, _ := show(&s, 1, &colours, &colours, 220*ms); got.Background != b.Background {
		t.Errorf("reduced motion kept %v, want %v", got.Background, b.Background)
	}
	idle(t, &s)
	if got, _ := show(&s, 1, &b, &a, 230*ms); got.Background != a.Background || got.Opacity != a.Opacity {
		t.Errorf("reduced motion started a transition: %v %v", got.Background, got.Opacity)
	}
	idle(t, &s)
}

func TestStylesCurrentColourAndUnsetText(t *testing.T) {
	a, b := card(color.RGBA{A: 255}, 1), card(color.RGBA{A: 255}, 1)
	a.Color, b.Color = literal(color.RGBA{R: 255, A: 255}), literal(color.RGBA{B: 255, A: 255})
	var s Styles
	show(&s, 1, &a, &b, 0)
	got, _ := show(&s, 1, &b, &b, 100*ms)
	if got.BorderColor.Kind != color.Literal || got.BorderColor != got.Color || got.Color == a.Color || got.Color == b.Color {
		t.Errorf("currentColor border %v, text %v: want the same mid colour", got.BorderColor, got.Color)
	}
	unset := b
	unset.Color = color.Color{}
	var fresh Styles
	show(&fresh, 2, &b, &unset, 0)
	if got, _ := show(&fresh, 2, &unset, &unset, 100*ms); got.Color.Kind != color.Unset {
		t.Errorf("text colour to the terminal default interpolated to %v", got.Color)
	}
}

func TestStylesShadows(t *testing.T) {
	shade := literal(color.RGBA{A: 64})
	small := []style.Shadow{{Y: 1, Blur: 2, Color: shade}}
	large := []style.Shadow{{Y: 4, Blur: 6, Color: shade}, {Y: 2, Blur: 4, Color: shade}}
	keep := slices.Clone(large)
	a, b := card(color.RGBA{A: 255}, 1), card(color.RGBA{A: 255}, 1)
	a.Shadows, b.Shadows = small, large
	var s Styles
	show(&s, 1, &a, &b, 0)
	got, _ := show(&s, 1, &b, &b, 100*ms)
	want := []style.Shadow{{Y: 3, Blur: 4, Color: shade}, {Y: 1, Blur: 2, Color: literal(color.RGBA{A: 32})}}
	if !slices.Equal(got.Shadows, want) {
		t.Errorf("shadows at half %+v, want %+v", got.Shadows, want)
	}
	if !slices.Equal(b.Shadows, keep) {
		t.Errorf("the next style's shadow slice was written: %+v", b.Shadows)
	}
	if got, _ := show(&s, 1, &b, &b, 200*ms); !slices.Equal(got.Shadows, large) {
		t.Errorf("shadows at the end %+v", got.Shadows)
	}
	idle(t, &s)
}

func pulse() style.Animation {
	return style.Animation{Keyframes: style.KeyframesPulse, Duration: 2 * time.Second, Easing: style.Easing{X1: 0.4, X2: 0.6, Y2: 1}, Infinite: true}
}

func TestStylesAnimation(t *testing.T) {
	a := card(color.RGBA{A: 255}, 1)
	a.Animation = pulse()
	var s Styles
	show(&s, 1, nil, &a, time.Second)
	if got, _ := show(&s, 1, &a, &a, 2*time.Second); got.Opacity != 0.5 {
		t.Errorf("pulse one second after mount %v, want 0.5", got.Opacity)
	}
	if at, moving := s.Wake(); !moving || at != 2*time.Second {
		t.Errorf("pulse wake %v %v", at, moving)
	}
	once := a
	once.Animation = style.Animation{Keyframes: style.KeyframesBounce, Duration: time.Second, Iterations: 1}
	if got, pose := show(&s, 1, &a, &once, 3*time.Second); pose.TranslateY != percent(-25) || got.TranslateY != (style.Length{}) {
		t.Errorf("bounce restarted at pose %v style %v, want a -25%% pose over an untouched style", pose.TranslateY, got.TranslateY)
	}
	for _, at := range []time.Duration{4 * time.Second, 5 * time.Second} {
		if _, pose := show(&s, 1, &once, &once, at); pose.TranslateY.Value != 0 {
			t.Errorf("finished bounce at %v translates %v", at, pose.TranslateY)
		}
	}
	if at, moving := s.Wake(); moving {
		t.Errorf("finished animation wakes at %v", at)
	}
	s.Reduced = true
	if got, pose := show(&s, 1, &a, &a, 6*time.Second); got.Opacity != 1 || pose.Opacity != 1 {
		t.Errorf("reduced pulse opacity %v", got.Opacity)
	}
	if _, moving := s.Wake(); moving {
		t.Error("reduced pulse wakes")
	}
	s.Drop(1)
	idle(t, &s)
}

func TestStylesOverlayOnlyMoved(t *testing.T) {
	dark, light := card(color.RGBA{A: 255}, 1), card(color.RGBA{R: 255, G: 255, B: 255, A: 255}, 1)
	faded := dark
	faded.Opacity = 0
	var s Styles
	var a Animated
	if !s.Frame(1, &dark, &light, 0, &a) || !s.Frame(1, &light, &light, 100*ms, &a) {
		t.Fatal("background transition reports no movement")
	}
	if !s.Frame(2, &dark, &faded, 0, &a) || !s.Frame(2, &faded, &faded, 100*ms, &a) {
		t.Fatal("opacity transition reports no movement")
	}
	shown := faded
	shown.Background = literal(color.RGBA{G: 200, A: 255})
	s.Overlay(&a, &shown)
	if shown.Background.RGBA != (color.RGBA{G: 200, A: 255}) || shown.Opacity != 0.5 {
		t.Errorf("overlay of key 2 wrote background %v, opacity %v; want only opacity 0.5", shown.Background, shown.Opacity)
	}
	pulsing := dark
	pulsing.Animation = pulse()
	s.Frame(3, nil, &pulsing, 0, &a)
	if !s.Frame(3, &pulsing, &pulsing, time.Second, &a) || a.Pose.Opacity != 0.5 {
		t.Errorf("a pulse one second in reports no movement, pose %+v", a.Pose)
	}
	still := dark
	still.Animation = style.Animation{Keyframes: style.KeyframesPulse, Duration: time.Second, Iterations: 1}
	s.Frame(4, nil, &still, 0, &a)
	if s.Frame(4, &still, &still, 5*time.Second, &a) {
		t.Error("a finished animation reports movement")
	}
}

func TestKeyframesTailwind(t *testing.T) {
	ping := style.Animation{Keyframes: style.KeyframesPing, Duration: time.Second, Easing: style.Easing{X2: 0.2, Y2: 1}, Infinite: true}
	spin := style.Animation{Keyframes: style.KeyframesSpin, Duration: time.Second, Easing: style.Easing{X2: 1, Y2: 1}, Infinite: true}
	bounce := style.Animation{Keyframes: style.KeyframesBounce, Duration: time.Second, Easing: style.Easing{X1: 0.25, Y1: 0.1, X2: 0.25, Y2: 1}, Infinite: true}
	fall := bisectBezier(0.8, 0, 1, 1, 0.5)
	cases := []struct {
		name    string
		a       style.Animation
		opacity float64
		at      time.Duration
		want    Pose
	}{
		{"pulse 0 s", pulse(), 1, 0, Pose{Opacity: 1, Scale: 1}},
		{"pulse 0.5 s", pulse(), 1, 500 * ms, Pose{Opacity: 0.75, Scale: 1}},
		{"pulse 1 s", pulse(), 1, time.Second, Pose{Opacity: 0.5, Scale: 1}},
		{"pulse 2 s", pulse(), 1, 2 * time.Second, Pose{Opacity: 1, Scale: 1}},
		{"pulse base 0.8, 0 s", pulse(), 0.8, 0, Pose{Opacity: 0.8, Scale: 1}},
		{"pulse base 0.8, 1 s", pulse(), 0.8, time.Second, Pose{Opacity: 0.5, Scale: 1}},
		{"ping 0.75 s", ping, 1, 750 * ms, Pose{Opacity: 0, Scale: 2}},
		{"ping 0.9 s", ping, 1, 900 * ms, Pose{Opacity: 0, Scale: 2}},
		{"ping 0.375 s", ping, 1, 375 * ms, Pose{Opacity: 1 - bisectBezier(0, 0, 0.2, 1, 0.5), Scale: 1 + bisectBezier(0, 0, 0.2, 1, 0.5)}},
		{"spin 0.25 s", spin, 1, 250 * ms, Pose{Opacity: 1, Scale: 1, Turn: 0.25}},
		{"bounce 0 s", bounce, 1, 0, Pose{Opacity: 1, Scale: 1, TranslateY: percent(-25)}},
		{"bounce 0.25 s", bounce, 1, 250 * ms, Pose{Opacity: 1, Scale: 1, TranslateY: percent(-25 * (1 - fall))}},
		{"bounce 0.5 s", bounce, 1, 500 * ms, Pose{Opacity: 1, Scale: 1, TranslateY: percent(0)}},
		{"bounce 0.75 s", bounce, 1, 750 * ms, Pose{Opacity: 1, Scale: 1, TranslateY: percent(-25 * bisectBezier(0, 0, 0.2, 1, 0.5))}},
		{"bounce 1 s", bounce, 1, time.Second, Pose{Opacity: 1, Scale: 1, TranslateY: percent(-25)}},
	}
	for _, c := range cases {
		got := Keyframe(c.a, c.opacity, c.at)
		if math.Abs(got.Opacity-c.want.Opacity) > 1e-4 || math.Abs(got.Scale-c.want.Scale) > 1e-4 || math.Abs(got.Turn-c.want.Turn) > 1e-4 ||
			got.TranslateY.Unit != c.want.TranslateY.Unit || math.Abs(got.TranslateY.Value-c.want.TranslateY.Value) > 1e-3 {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestStylesEncodeTableMatchesPow(t *testing.T) {
	var table srgb
	table.build()
	for i := -10; i <= 1_000_010; i++ {
		v := float64(i) / 1_000_000
		if got, want := table.encode(v), encode(v); got != want {
			t.Fatalf("linear %v encodes to %d, want %d", v, got, want)
		}
	}
}

func BenchmarkStylesColourFrame(b *testing.B) {
	from, to := card(color.RGBA{R: 244, G: 244, B: 245, A: 255}, 1), card(color.RGBA{R: 24, G: 24, B: 27, A: 255}, 1)
	to.Transition.Duration = time.Hour
	for _, n := range []Key{100, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			var s Styles
			var a Animated
			shown := make([]style.ComputedStyle, n)
			for k := range n {
				shown[k] = to
				s.Frame(k, &from, &to, 0, &a)
			}
			now, again := time.Minute, to
			b.ReportAllocs()
			for b.Loop() {
				now += time.Millisecond
				for k := range n {
					if s.Frame(k, &to, &again, now, &a) {
						s.Overlay(&a, &shown[k])
					}
				}
			}
			for k := range shown {
				if shown[k].Background == to.Background || shown[k].Background == from.Background {
					b.Fatalf("element %d shows %v, not a mid colour", k, shown[k].Background)
				}
			}
			if _, moving := s.Wake(); !moving || len(s.entries) != int(n) {
				b.Fatalf("%d entries moving %v", len(s.entries), moving)
			}
		})
	}
}
