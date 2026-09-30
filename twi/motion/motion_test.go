package motion

import (
	"math"
	"testing"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/motion"
	"github.com/twind-dev/twind/twi/color"
)

const frame = 16 * time.Millisecond

func bisectBezier(x1, y1, x2, y2, x float64) float64 {
	curve := func(a, b, u float64) float64 {
		return 3*(1-u)*(1-u)*u*a + 3*(1-u)*u*u*b + u*u*u
	}
	lo, hi := 0.0, 1.0
	for range 200 {
		mid := (lo + hi) / 2
		if curve(x1, x2, mid) < x {
			lo = mid
		} else {
			hi = mid
		}
	}
	return curve(y1, y2, (lo+hi)/2)
}

func TestEaseInOutMatchesBisection(t *testing.T) {
	ease := EaseInOut()
	for _, x := range []float64{0.25, 0.5, 0.75} {
		want := bisectBezier(0.4, 0, 0.2, 1, x)
		if got := ease.at(x); math.Abs(got-want) > 1e-4 {
			t.Errorf("ease-in-out at %v = %v, want %v", x, got, want)
		}
	}
	for _, e := range []Easing{EaseIn(), EaseOut(), EaseInOut(), CubicBezier(0.9, -0.5, 0.1, 1.5)} {
		for x := 0.0; x <= 1; x += 1.0 / 64 {
			want := bisectBezier(e.x1, e.y1, e.x2, e.y2, x)
			if got := e.at(x); math.Abs(got-want) > 1e-4 {
				t.Errorf("%+v at %v = %v, want %v", e, x, got, want)
			}
		}
	}
}

func TestEaseSteps(t *testing.T) {
	cases := []struct {
		jump Jump
		want []float64
	}{
		{JumpEnd, []float64{0, 0, 1.0 / 3, 2.0 / 3, 1}},
		{JumpStart, []float64{1.0 / 3, 1.0 / 3, 2.0 / 3, 1, 1}},
		{JumpNone, []float64{0, 0, 0.5, 1, 1}},
		{JumpBoth, []float64{0.25, 0.25, 0.5, 0.75, 1}},
	}
	for _, c := range cases {
		e := Steps(3, c.jump)
		for i, x := range []float64{0, 0.2, 0.4, 0.7, 1} {
			if got := e.at(x); math.Abs(got-c.want[i]) > 1e-12 {
				t.Errorf("steps(3, %v) at %v = %v, want %v", c.jump, x, got, c.want[i])
			}
		}
	}
}

func TestEaseTweenClampsTime(t *testing.T) {
	var tl Timeline
	id := tl.Start(time.Second, Float(10), Float(20), Tween(100*time.Millisecond, EaseOut()))
	tl.Step(500 * time.Millisecond)
	if got := tl.Value(id).Float(); got != 10 {
		t.Errorf("before start = %v, want 10", got)
	}
	done, running := tl.Step(2 * time.Second)
	if got := tl.Value(id).Float(); got != 20 || len(done) != 1 || running {
		t.Errorf("after end = %v, done %v, running %v", got, done, running)
	}
}

func runSpring(t *testing.T, tr Transition, step time.Duration) (peak float64, peakAt, stop, settle time.Duration) {
	t.Helper()
	var tl Timeline
	id := tl.Start(0, Float(0), Float(1), tr)
	settle = tl.Settle(id)
	for now := step; now < konst.SettleLimit; now += step {
		done, _ := tl.Step(now)
		value := tl.Value(id).Float()
		if math.IsNaN(value) {
			t.Fatalf("value NaN at %v", now)
		}
		if value > peak {
			peak, peakAt = value, now
		}
		if len(done) == 1 {
			if value != 1 {
				t.Fatalf("rest value %v, want 1", value)
			}
			return peak, peakAt, now, settle
		}
	}
	t.Fatal("spring never came to rest")
	return
}

func TestSpringCriticalNoOvershoot(t *testing.T) {
	peak, _, stop, settle := runSpring(t, Spring(100, 20, 1), time.Millisecond/10)
	if peak > 1 {
		t.Errorf("critically damped peak %v, want at most 1", peak)
	}
	if stop > settle {
		t.Errorf("stopped at %v, reported settle %v", stop, settle)
	}
	stepResponse(t, Spring(100, 20, 1), func(s float64) float64 { return 1 - math.Exp(-10*s)*(1+10*s) })
	t.Logf("critical: stop %v, settle %v", stop, settle)
}

func stepResponse(t *testing.T, tr Transition, want func(seconds float64) float64) {
	t.Helper()
	var tl Timeline
	id := tl.Start(0, Float(0), Float(1), tr)
	for now := frame; now < 400*time.Millisecond; now += frame {
		tl.Step(now)
		if got, want := tl.Value(id).Float(), want(now.Seconds()); math.Abs(got-want) > 1e-6 {
			t.Errorf("step response at %v = %v, want %v", now, got, want)
		}
	}
}

func TestSpringUnderdampedOvershoot(t *testing.T) {
	zeta := 4 / (2 * math.Sqrt(100))
	want := 1 + math.Exp(-zeta*math.Pi/math.Sqrt(1-zeta*zeta))
	wantAt := time.Duration(math.Pi / (10 * math.Sqrt(1-zeta*zeta)) * float64(time.Second))
	peak, peakAt, stop, settle := runSpring(t, Spring(100, 4, 1), time.Millisecond/10)
	if math.Abs(peak-want) > 1e-3 || (peakAt-wantAt).Abs() > time.Millisecond {
		t.Errorf("underdamped peak %v at %v, want %v at %v", peak, peakAt, want, wantAt)
	}
	if stop > settle {
		t.Errorf("stopped at %v, reported settle %v", stop, settle)
	}
	stepResponse(t, Spring(100, 4, 1), func(s float64) float64 {
		wd := 10 * math.Sqrt(1-zeta*zeta)
		return 1 - math.Exp(-zeta*10*s)*(math.Cos(wd*s)+zeta/math.Sqrt(1-zeta*zeta)*math.Sin(wd*s))
	})
	t.Logf("underdamped: peak %v, stop %v, settle %v", peak, stop, settle)
}

func TestSpringOverdampedAndFramesSettle(t *testing.T) {
	for _, tr := range []Transition{Spring(100, 60, 1), SpringDefault(), SpringGentle(), SpringSnappy(), SpringBouncy()} {
		peak, _, stop, settle := runSpring(t, tr, frame)
		if stop > settle+frame {
			t.Errorf("zeta %.3f stopped at %v, reported settle %v", tr.zeta, stop, settle)
		}
		if tr.zeta >= 1 && peak > 1 {
			t.Errorf("zeta %.3f overdamped peak %v", tr.zeta, peak)
		}
	}
	fast, slow := -30+10*math.Sqrt(8), -30-10*math.Sqrt(8)
	stepResponse(t, Spring(100, 60, 1), func(s float64) float64 {
		return 1 - (fast*math.Exp(slow*s)-slow*math.Exp(fast*s))/(fast-slow)
	})
}

func TestRetargetContinuous(t *testing.T) {
	for _, tr := range []Transition{Spring(170, 12, 1), Spring(100, 20, 1), Spring(100, 60, 1), Tween(300*time.Millisecond, EaseInOut())} {
		var tl, twin Timeline
		id := tl.Start(0, Bounds(0, 0, 10, 4), Bounds(40, 10, 20, 8), tr)
		twin.Start(0, Bounds(0, 0, 10, 4), Bounds(40, 10, 20, 8), tr)
		now := 90 * time.Millisecond
		tl.Step(80 * time.Millisecond)
		twin.Step(now)
		before, beforeSpeed := twin.slots[id].value, twin.slots[id].motion
		tl.Retarget(id, now, Bounds(-20, 5, 30, 2))
		after, afterSpeed := tl.slots[id].value, tl.slots[id].motion
		for c := range before {
			if math.Abs(before[c]-after[c]) > 1e-6 {
				t.Errorf("zeta %.3f channel %d value %v then %v", tr.zeta, c, before[c], after[c])
			}
			if tr.kind == spring && math.Abs(beforeSpeed[c]-afterSpeed[c]) > 1e-6 {
				t.Errorf("zeta %.3f channel %d velocity %v then %v", tr.zeta, c, beforeSpeed[c], afterSpeed[c])
			}
		}
		if tr.kind == spring && beforeSpeed == [4]float64{} {
			t.Errorf("zeta %.3f retargeted at rest, the test proves nothing", tr.zeta)
		}
		if tr.kind != spring {
			continue
		}
		tl.Step(now + time.Microsecond)
		for c, got := range tl.Value(id).ch {
			if want := before[c] + beforeSpeed[c]*1e-6; math.Abs(got-want) > 1e-6 {
				t.Errorf("zeta %.3f channel %d one microsecond after retarget %v, want %v", tr.zeta, c, got, want)
			}
		}
	}
}

func TestReducedFinishesOnFirstStep(t *testing.T) {
	tl := Timeline{Reduced: true}
	ids := []ID{
		tl.Start(time.Second, Float(0), Float(1), Tween(time.Hour, EaseIn())),
		tl.Start(time.Second, Float(3), Float(-2), SpringBouncy()),
		tl.Start(time.Second, Color(rgba(255, 0, 0, 255)), Color(rgba(0, 0, 255, 128)), Spring(1, 0, 1)),
	}
	done, running := tl.Step(time.Second)
	if len(done) != len(ids) || running {
		t.Fatalf("done %v running %v, want all done", done, running)
	}
	if tl.Value(ids[0]).Float() != 1 || tl.Value(ids[1]).Float() != -2 || tl.Value(ids[2]).Color() != rgba(0, 0, 255, 128) {
		t.Errorf("values %v %v %v", tl.Value(ids[0]), tl.Value(ids[1]), tl.Value(ids[2]).Color())
	}
}

func rgba(r, g, b, a uint8) color.RGBA { return color.RGBA{R: r, G: g, B: b, A: a} }

func TestColorOKLab(t *testing.T) {
	for c, want := range map[color.RGBA][3]float64{
		rgba(255, 255, 255, 255): {1, 0, 0},
		rgba(255, 0, 0, 255):     {0.627955, 0.224863, 0.125846},
		rgba(0, 255, 0, 255):     {0.866440, -0.233888, 0.179498},
	} {
		lab := Color(c).ch
		for i := range want {
			if got := lab[i]; math.Abs(got-want[i]) > 1e-4 {
				t.Errorf("%v channel %d = %v, want %v", c, i, got, want[i])
			}
		}
	}
}

func TestColorRoundTrip(t *testing.T) {
	for _, c := range []color.RGBA{rgba(0, 0, 0, 0), rgba(255, 255, 255, 255), rgba(255, 0, 0, 255), rgba(12, 200, 99, 7), rgba(5, 9, 3, 255), rgba(128, 128, 128, 200)} {
		if got := Color(c).Color(); got != c {
			t.Errorf("%v round trips to %v", c, got)
		}
	}
}

func TestStepReusesSlotsOnlyAfterNextStep(t *testing.T) {
	var tl Timeline
	a := tl.Start(0, Float(0), Float(5), Tween(frame, Linear()))
	tl.Step(frame)
	b := tl.Start(frame, Float(0), Float(9), Tween(frame, Linear()))
	if a == b || tl.Value(a).Float() != 5 {
		t.Fatalf("slot %d reused before the next step", a)
	}
	tl.Step(2 * frame)
	if c := tl.Start(2*frame, Float(1), Float(2), Tween(frame, Linear())); c != a {
		t.Errorf("slot %d not reused, got %d", a, c)
	}
}

func BenchmarkStep(b *testing.B) {
	presets := func(i int) Transition {
		return []Transition{Tween(time.Hour, EaseInOut()), SpringBouncy(), SpringDefault()}[i%3]
	}
	distinct := func(i int) Transition {
		if i%3 == 0 {
			return presets(i)
		}
		return Spring(100+float64(i), 5+float64(i)/40, 1)
	}
	for _, springs := range []struct {
		name       string
		transition func(int) Transition
	}{{"presets", presets}, {"distinct", distinct}} {
		b.Run(springs.name, func(b *testing.B) {
			var tl Timeline
			for i := range 1000 {
				f := float64(i)
				if i%2 == 0 {
					tl.Start(0, Color(rgba(uint8(i), 20, 200, 255)), Color(rgba(0, uint8(i), 0, 255)), springs.transition(i))
				} else {
					tl.Start(0, Bounds(0, 0, 10, 4), Bounds(f, f, 20, 8), springs.transition(i))
				}
			}
			tl.Step(50 * time.Millisecond)
			now := 50 * time.Millisecond
			b.ReportAllocs()
			for b.Loop() {
				tl.responses = [konst.ResponseCache]response{}
				now += time.Nanosecond
				if _, running := tl.Step(now); !running {
					b.Fatal("animations finished during the benchmark")
				}
			}
			if len(tl.done) != 0 {
				b.Fatalf("%d animations finished during the benchmark", len(tl.done))
			}
		})
	}
}
