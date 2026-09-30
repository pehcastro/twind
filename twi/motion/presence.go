package motion

import (
	"time"

	"github.com/twind-dev/twind/twi/style"
)

type Offset struct{ Opacity, Scale, X, Y float64 }

func Still() Offset { return Offset{Opacity: 1, Scale: 1} }

type Presence struct {
	Offset   Offset
	Duration time.Duration
	Delay    time.Duration
	Fill     style.Fill
	Easing   Easing
}

func (p Presence) Total() time.Duration { return p.Delay + p.Duration }

func (p Presence) Enter(elapsed time.Duration) Offset { return p.at(elapsed, p.Offset, Still()) }

func (p Presence) Exit(elapsed time.Duration) Offset { return p.at(elapsed, Still(), p.Offset) }

func (p Presence) at(elapsed time.Duration, from, to Offset) Offset {
	t, on := progress(elapsed-p.Delay, p.Duration, 1, false, p.Fill)
	if !on {
		return Still()
	}
	return from.toward(to, p.Easing.at(t))
}

func (o Offset) toward(to Offset, t float64) Offset {
	return Offset{
		Opacity: o.Opacity + (to.Opacity-o.Opacity)*t,
		Scale:   o.Scale + (to.Scale-o.Scale)*t,
		X:       o.X + (to.X-o.X)*t,
		Y:       o.Y + (to.Y-o.Y)*t,
	}
}
