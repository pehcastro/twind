package runtime

import (
	"time"

	"github.com/pehcastro/twind/twi/input"
)

type tracer interface {
	Trace(format string, args ...any)
}

type phases struct {
	to      tracer
	clock   Clock
	started time.Time
	input   time.Time
	drawn   bool
	after   bool
}

func (p *phases) begin(b Backend, clock Clock) {
	p.clock = clock
	p.to, _ = b.(tracer)
	if p.to != nil {
		p.started = clock.Now()
	}
}

func (p *phases) seen(ev input.Event) {
	if p.to == nil || !p.input.IsZero() {
		return
	}
	switch ev.(type) {
	case input.KeyEvent, input.MouseEvent, input.PasteEvent:
		p.input = p.clock.Now()
		p.to.Trace("runtime: first input")
	}
}

func (p *phases) frame() {
	switch {
	case p.to == nil:
	case !p.drawn:
		p.drawn = true
		p.to.Trace("runtime: first frame drawn in %v", p.since(p.started))
	case !p.after && !p.input.IsZero():
		p.after = true
		p.to.Trace("runtime: first frame after the first input, %v after it", p.since(p.input))
	}
}

func (p *phases) since(t time.Time) time.Duration {
	return p.clock.Now().Sub(t).Round(time.Microsecond)
}

func (p *phases) log(format string, args ...any) {
	if p.to != nil {
		p.to.Trace(format, args...)
	}
}
