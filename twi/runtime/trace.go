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
	started time.Time
	input   time.Time
	drawn   bool
	after   bool
}

func (p *phases) begin(b Backend) {
	p.to, _ = b.(tracer)
	if p.to != nil {
		p.started = time.Now()
	}
}

func (p *phases) seen(ev input.Event) {
	if p.to == nil || !p.input.IsZero() {
		return
	}
	switch ev.(type) {
	case input.KeyEvent, input.MouseEvent, input.PasteEvent:
		p.input = time.Now()
		p.to.Trace("runtime: first input")
	}
}

func (p *phases) frame() {
	switch {
	case p.to == nil:
	case !p.drawn:
		p.drawn = true
		p.to.Trace("runtime: first frame drawn in %v", time.Since(p.started).Round(time.Microsecond))
	case !p.after && !p.input.IsZero():
		p.after = true
		p.to.Trace("runtime: first frame after the first input, %v after it", time.Since(p.input).Round(time.Microsecond))
	}
}

func (p *phases) log(format string, args ...any) {
	if p.to != nil {
		p.to.Trace(format, args...)
	}
}
