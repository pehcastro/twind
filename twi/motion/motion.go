package motion

import (
	"math"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/motion"
)

type transitionKind uint8

const (
	tween transitionKind = iota
	spring
)

type Transition struct {
	kind         transitionKind
	duration     time.Duration
	easing       Easing
	omega, zeta  float64
	sigma, swing float64
}

func Tween(duration time.Duration, easing Easing) Transition {
	return Transition{kind: tween, duration: duration, easing: easing}
}

func Spring(stiffness, damping, mass float64) Transition {
	omega, zeta := math.Sqrt(stiffness/mass), damping/(2*math.Sqrt(stiffness*mass))
	return Transition{kind: spring, omega: omega, zeta: zeta, sigma: zeta * omega, swing: omega * math.Sqrt(math.Abs(1-zeta*zeta))}
}

func SpringDefault() Transition { return Spring(170, 26, 1) }

func SpringGentle() Transition { return Spring(120, 14, 1) }

func SpringSnappy() Transition { return Spring(400, 40, 1) }

func SpringBouncy() Transition { return Spring(180, 12, 1) }

type response struct {
	omega, zeta float64
	dt          time.Duration
	a, b, c, d  float64
}

func (tr Transition) response(dt time.Duration) response {
	r := response{omega: tr.omega, zeta: tr.zeta, dt: dt}
	t, w, s, q := dt.Seconds(), tr.omega, tr.sigma, tr.swing
	switch {
	case math.Abs(tr.zeta-1) < konst.CriticalBand:
		e := math.Exp(-w * t)
		r.a, r.b, r.c, r.d = e*(1+w*t), e*t, -e*w*w*t, e*(1-w*t)
	case tr.zeta < 1:
		sin, cos := math.Sincos(q * t)
		e := math.Exp(-s * t)
		turn := e * sin / q
		r.a, r.b, r.c, r.d = e*cos+s*turn, turn, -w*w*turn, e*cos-s*turn
	default:
		r1, r2, span := q-s, -q-s, 2*q
		e1, e2 := math.Exp(r1*t), math.Exp(r2*t)
		r.a, r.b, r.c, r.d = (r1*e2-r2*e1)/span, (e1-e2)/span, r1*r2*(e2-e1)/span, (r1*e1-r2*e2)/span
	}
	return r
}

type animation struct {
	live          bool
	tr            Transition
	start, last   time.Duration
	from, to      [4]float64
	value, motion [4]float64
}

func (an *animation) spring(r *response) {
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

type Timeline struct {
	Reduced   bool
	slots     []animation
	free      []ID
	done      []ID
	running   int
	responses [konst.ResponseCache]response
	next      int
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

func (t *Timeline) response(tr *Transition, dt time.Duration) *response {
	for i := range t.responses {
		if r := &t.responses[i]; r.dt == dt && r.omega == tr.omega && r.zeta == tr.zeta {
			return r
		}
	}
	r := &t.responses[t.next]
	*r = tr.response(dt)
	t.next = (t.next + 1) % len(t.responses)
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
		an.spring(&r)
	}
	return settled
}
