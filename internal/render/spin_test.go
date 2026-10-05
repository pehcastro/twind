package render_test

import (
	"math"
	"testing"
	"time"

	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/render/testdata/sheet"
)

const ms = time.Millisecond

func spinner(extra ...string) render.Node {
	return screen(render.Node{Classes: classes(append([]string{sheet.Spinning}, extra...)...)})
}

func (r *motionRun) turn(now time.Duration, root render.Node, want float64) {
	r.t.Helper()
	got := r.at(now, root).Children[0].Turn
	r.t.Logf("graphics %v at %4d ms: turn %.3f", r.frame.Graphics, now.Milliseconds(), got)
	if math.Abs(got-want) > 1e-6 {
		r.t.Errorf("graphics %v at %v: turn %v, want %v", r.frame.Graphics, now, got, want)
	}
}

func TestSpinTurnsTheScene(t *testing.T) {
	r := newMotionRun(t)
	r.frame.Graphics = true
	for _, step := range []struct {
		now  time.Duration
		turn float64
	}{{0, 0}, {250 * ms, 0.25}, {500 * ms, 0.5}, {1250 * ms, 0.25}} {
		r.turn(step.now, spinner(), step.turn)
		r.wake(true)
	}
}

func TestSpinWithoutGraphicsIsIdle(t *testing.T) {
	r := newMotionRun(t)
	for _, now := range []time.Duration{0, 250 * ms, 500 * ms} {
		r.turn(now, spinner(), 0)
		r.wake(false)
	}
	r.frame.Graphics = true
	r.turn(750*ms, spinner(), 0.75)
	r.wake(true)
}

func TestSpinHiddenIsIdle(t *testing.T) {
	for _, hide := range []string{sheet.Invisible, sheet.Hidden, sheet.Empty} {
		t.Run(hide, func(t *testing.T) {
			r := newMotionRun(t)
			r.frame.Graphics = true
			for _, now := range []time.Duration{0, 250 * ms} {
				r.at(now, spinner(hide))
				r.wake(false)
			}
			r.turn(500*ms, spinner(), 0.5)
			r.wake(true)
		})
	}
}

func TestSpinReducedMotion(t *testing.T) {
	r := newMotionRun(t)
	r.frame.Graphics, r.frame.ReducedMotion = true, true
	r.turn(250*ms, spinner(), 0)
	r.wake(false)
}

func TestTurnIdleSparesZeroSizeExit(t *testing.T) {
	r := newMotionRun(t)
	r.at(0, screen(render.Node{Classes: classes(sheet.Vanishing)}))
	r.wake(true)
}
