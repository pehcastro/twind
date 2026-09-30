package events

import "github.com/twind-dev/twind/twi/input"

type Type uint8

const (
	KeyDown Type = iota
	KeyUp
	MouseDown
	MouseUp
	MouseMove
	Click
	Focus
	Blur
)

type Phase uint8

const (
	Capture Phase = iota + 1
	AtTarget
	Bubble
)

type Listener[N comparable] struct {
	Capture bool
	Handle  func(*Event[N])
}

type Tree[N comparable] interface {
	Root() N
	Parent(N) (N, bool)
	Children(N) []N
	Listeners(N, Type) []Listener[N]
	Focusable(N) bool
	TabIndex(N) int
	Disabled(N) bool
}

type Event[N comparable] struct {
	Type  Type
	Key   input.KeyEvent
	Mouse input.MouseEvent

	target, current N
	phase           Phase
	prevented       bool
	stopped         bool
}

func (e *Event[N]) Target() N              { return e.target }
func (e *Event[N]) Current() N             { return e.current }
func (e *Event[N]) Phase() Phase           { return e.phase }
func (e *Event[N]) PreventDefault()        { e.prevented = true }
func (e *Event[N]) DefaultPrevented() bool { return e.prevented }
func (e *Event[N]) StopPropagation()       { e.stopped = true }

func (k Type) bubbles() bool {
	switch k {
	case KeyDown, KeyUp, MouseDown, MouseUp, MouseMove, Click:
		return true
	case Focus, Blur:
		return false
	}
	panic("events: unknown type")
}

func Dispatch[N comparable](t Tree[N], target N, e *Event[N]) {
	path := []N{target}
	for n, ok := t.Parent(target); ok; n, ok = t.Parent(n) {
		path = append(path, n)
	}
	e.target = target
	e.phase = Capture
	for i := len(path) - 1; i > 0; i-- {
		if !e.run(t, path[i], true) {
			return
		}
	}
	e.phase = AtTarget
	if !e.run(t, target, true) || !e.run(t, target, false) || !e.Type.bubbles() {
		return
	}
	e.phase = Bubble
	for _, n := range path[1:] {
		if !e.run(t, n, false) {
			return
		}
	}
}

func (e *Event[N]) run(t Tree[N], n N, capture bool) bool {
	e.current = n
	for _, l := range t.Listeners(n, e.Type) {
		if l.Capture != capture {
			continue
		}
		l.Handle(e)
		if e.stopped {
			return false
		}
	}
	return true
}
