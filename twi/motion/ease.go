package motion

import (
	"math"

	konst "github.com/pehcastro/twind/internal/konst/motion"
)

type easingKind uint8

const (
	linear easingKind = iota
	bezier
	steps
)

type Jump uint8

const (
	JumpEnd Jump = iota
	JumpStart
	JumpNone
	JumpBoth
)

type Easing struct {
	kind           easingKind
	x1, y1, x2, y2 float64
	count, lift    float64
	jumps          float64
}

func Linear() Easing { return Easing{kind: linear} }

func CubicBezier(x1, y1, x2, y2 float64) Easing {
	x1, x2 = min(max(x1, 0), 1), min(max(x2, 0), 1)
	if x1 == y1 && x2 == y2 {
		return Linear()
	}
	return Easing{kind: bezier, x1: x1, y1: y1, x2: x2, y2: y2}
}

func EaseIn() Easing { return CubicBezier(0.4, 0, 1, 1) }

func EaseOut() Easing { return CubicBezier(0, 0, 0.2, 1) }

func EaseInOut() Easing { return CubicBezier(0.4, 0, 0.2, 1) }

func Steps(count int, jump Jump) Easing {
	e := Easing{kind: steps, count: float64(max(count, 1))}
	switch jump {
	case JumpEnd:
		e.jumps = e.count
	case JumpStart:
		e.jumps, e.lift = e.count, 1
	case JumpNone:
		e.jumps = e.count - 1
	case JumpBoth:
		e.jumps, e.lift = e.count+1, 1
	default:
		panic("motion: unknown jump")
	}
	e.jumps = max(e.jumps, 1)
	return e
}

func (e Easing) at(p float64) float64 {
	switch e.kind {
	case linear:
		return p
	case steps:
		return min(math.Floor(p*e.count)+e.lift, e.jumps) / e.jumps
	case bezier:
		u := solveBezier(e.x1, e.x2, p)
		c := 3 * e.y1
		b := 3*(e.y2-e.y1) - c
		return (((1-c-b)*u+b)*u + c) * u
	}
	panic("motion: unknown easing")
}

func solveBezier(x1, x2, x float64) float64 {
	c := 3 * x1
	b := 3*(x2-x1) - c
	a := 1 - c - b
	u := x
	for range konst.BezierNewtonSteps {
		miss := ((a*u+b)*u+c)*u - x
		if math.Abs(miss) < konst.BezierEpsilon {
			return u
		}
		slope := (3*a*u+2*b)*u + c
		if math.Abs(slope) < konst.BezierSlopeFloor {
			break
		}
		u -= miss / slope
	}
	lo, hi := 0.0, 1.0
	u = x
	for range konst.BezierBisectSteps {
		got := ((a*u+b)*u + c) * u
		if math.Abs(got-x) < konst.BezierEpsilon {
			return u
		}
		if got < x {
			lo = u
		} else {
			hi = u
		}
		u = (lo + hi) / 2
	}
	return u
}
