package motion

import (
	"math"
	"time"

	konst "github.com/pehcastro/twind/internal/konst/motion"
)

type transitionKind uint8

const (
	tween transitionKind = iota
	spring
)

type Transition struct {
	kind                       transitionKind
	sigma, stiff, kappa, reach float64
	duration                   time.Duration
	easing                     Easing
}

func Tween(duration time.Duration, easing Easing) Transition {
	return Transition{kind: tween, duration: duration, easing: easing}
}

func Spring(stiffness, damping, mass float64) Transition {
	sigma, stiff := damping/(2*mass), stiffness/mass
	kappa := sigma*sigma - stiff
	return Transition{kind: spring, sigma: sigma, stiff: stiff, kappa: kappa, reach: max(sigma*sigma, math.Abs(kappa))}
}

func SpringDefault() Transition { return Spring(170, 26, 1) }

func SpringGentle() Transition { return Spring(120, 14, 1) }

func SpringSnappy() Transition { return Spring(400, 40, 1) }

func SpringBouncy() Transition { return Spring(180, 12, 1) }

type response struct{ a, b, c, d float64 }

func (tr *Transition) response(dt time.Duration) response {
	h, halvings := float64(dt)*(1/float64(time.Second)), 0
	for tr.reach*h*h > konst.SeriesReach {
		h, halvings = h/2, halvings+1
	}
	sh := tr.sigma * h
	cosh, sinhc := coshSinhc(sh * sh)
	c, s := coshSinhc(tr.kappa * h * h)
	decay := cosh - sh*sinhc
	a, b := decay*(c+sh*s), decay*h*s
	for range halvings {
		a, b = a*a-tr.stiff*b*b, 2*a*b-2*tr.sigma*b*b
	}
	return response{a, b, -tr.stiff * b, a - 2*tr.sigma*b}
}

func coshSinhc(z float64) (c, s float64) {
	const f2, f3, f4, f5, f6, f7 = 2, 6, 24, 120, 720, 5040
	const f8, f9, f10, f11 = f7 * 8, f7 * 8 * 9, f7 * 8 * 9 * 10, f7 * 8 * 9 * 10 * 11
	zz := z * z
	c = (1 + z*(1.0/f2)) + zz*((1.0/f4+z*(1.0/f6))+zz*(1.0/f8+z*(1.0/f10)))
	s = (1 + z*(1.0/f3)) + zz*((1.0/f5+z*(1.0/f7))+zz*(1.0/f9+z*(1.0/f11)))
	return c, s
}

type animation struct {
	live          bool
	start, last   time.Duration
	to            [4]float64
	value, motion [4]float64
	tr            Transition
	from          [4]float64
}

func (an *animation) spring(r response) {
	for c := range an.value {
		x, v := an.value[c]-an.to[c], an.motion[c]
		an.value[c], an.motion[c] = an.to[c]+r.a*x+r.b*v, r.c*x+r.d*v
	}
}

func (an *animation) resting() bool {
	for c := range an.value {
		if !(math.Abs(an.value[c]-an.to[c]) < konst.RestDelta && math.Abs(an.motion[c]) < konst.RestSpeed) {
			return false
		}
	}
	return true
}

type ID int32

type sharedResponse struct {
	sigma, stiff float64
	dt           time.Duration
	r            response
}

type Timeline struct {
	Reduced bool
	slots   []animation
	free    []ID
	done    []ID
	running int
	shared  [1 << konst.SharedResponseBits]sharedResponse
}

func (t *Timeline) Start(now time.Duration, from, to Value, tr Transition) ID {
	an := animation{live: true, tr: tr, start: now, last: now, from: from.ch, to: to.ch, value: from.ch}
	t.running++
	if n := len(t.free); n > 0 {
		id := t.free[n-1]
		t.free = t.free[:n-1]
		t.slots[id] = an
		return id
	}
	t.slots = append(t.slots, an)
	return ID(len(t.slots) - 1)
}

func (t *Timeline) Retarget(id ID, now time.Duration, to Value) {
	an := &t.slots[id]
	t.advance(an, now)
	an.from, an.to, an.start = an.value, to.ch, now
}

func (t *Timeline) Step(now time.Duration) (done []ID, running bool) {
	t.free = append(t.free, t.done...)
	t.done = t.done[:0]
	t.shared = [len(t.shared)]sharedResponse{}
	for i := range t.slots {
		an := &t.slots[i]
		if an.live && (t.Reduced || t.advance(an, now)) {
			an.live, an.value, an.motion = false, an.to, [4]float64{}
			t.done = append(t.done, ID(i))
			t.running--
		}
	}
	return t.done, t.running > 0
}

func (t *Timeline) advance(an *animation, now time.Duration) bool {
	switch an.tr.kind {
	case tween:
		p := 1.0
		if an.tr.duration > 0 {
			p = min(float64(max(now-an.start, 0))/float64(an.tr.duration), 1)
		}
		e := an.tr.easing.at(p)
		for c := range an.value {
			an.value[c] = an.from[c] + (an.to[c]-an.from[c])*e
		}
		return p == 1
	case spring:
		if now > an.last {
			an.spring(t.response(&an.tr, now-an.last))
			an.last = now
		}
		return an.resting() || now-an.start >= konst.SettleLimit
	}
	panic("motion: unknown transition")
}

func (t *Timeline) response(tr *Transition, dt time.Duration) response {
	mix := (math.Float64bits(tr.sigma) ^ math.Float64bits(tr.stiff)) * konst.SharedResponseMix
	s := &t.shared[mix>>(64-konst.SharedResponseBits)]
	if s.dt == dt && s.sigma == tr.sigma && s.stiff == tr.stiff {
		return s.r
	}
	r := tr.response(dt)
	s.sigma, s.stiff, s.dt, s.r = tr.sigma, tr.stiff, dt, r
	return r
}

func (t *Timeline) Value(id ID) Value { return Value{t.slots[id].value} }

func (t *Timeline) Settle(id ID) time.Duration {
	an := t.slots[id]
	if an.tr.kind == tween {
		return an.start + an.tr.duration
	}
	r := an.tr.response(konst.SettleSample)
	settled := an.last
	for at := an.last; at < an.start+konst.SettleLimit; at += konst.SettleSample {
		if !an.resting() {
			settled = at + konst.SettleSample
		}
		an.spring(r)
	}
	return settled
}
