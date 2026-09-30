package ui

import (
	"time"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/ui"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/style"
)

const spinnerFrames = "⣾⣽⣻⢿⡿⣟⣯⣷"

type Spinner struct {
	rt    *twi.Runtime
	at    int
	drawn bool
	tick  *twi.Timer
}

func NewSpinner(rt *twi.Runtime) *Spinner { return &Spinner{rt: rt} }

func (s *Spinner) Node(options ...twi.NodeOption) twi.Node {
	s.drawn = true
	if s.tick == nil {
		s.tick = s.rt.After(konst.SpinnerTurn/time.Duration(utf8.RuneCountInString(spinnerFrames)), s.turn)
	}
	glyph := []rune(spinnerFrames)[s.at]
	return part("shrink-0", append([]twi.NodeOption{twi.Tag(style.ElementSVG), twi.Data("slot", "spinner"), twi.Text(string(glyph))}, options...))
}

func (s *Spinner) turn() {
	s.tick = nil
	if s.drawn {
		s.drawn = false
		s.at = (s.at + 1) % utf8.RuneCountInString(spinnerFrames)
		s.rt.Invalidate()
	}
}
