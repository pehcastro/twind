package motion

import "time"

type Offset struct{ Opacity, Scale, X, Y float64 }

func Still() Offset { return Offset{Opacity: 1, Scale: 1} }

type Presence struct {
	Offset   Offset
	Duration time.Duration
	Easing   Easing
}

func (p Presence) Enter(elapsed time.Duration) Offset {
	return p.Offset.toward(Still(), p.eased(elapsed))
}

func (p Presence) Exit(elapsed time.Duration) Offset {
	return Still().toward(p.Offset, p.eased(elapsed))
}

func (p Presence) eased(elapsed time.Duration) float64 {
	if elapsed >= p.Duration {
		return 1
	}
	return p.Easing.at(float64(max(elapsed, 0)) / float64(p.Duration))
}

func (o Offset) toward(to Offset, t float64) Offset {
	return Offset{
		Opacity: o.Opacity + (to.Opacity-o.Opacity)*t,
		Scale:   o.Scale + (to.Scale-o.Scale)*t,
		X:       o.X + (to.X-o.X)*t,
		Y:       o.Y + (to.Y-o.Y)*t,
	}
}
