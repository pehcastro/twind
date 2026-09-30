package runtime

import (
	"errors"
	"fmt"
	"image"
	"io"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/runtime"
	"github.com/twind-dev/twind/internal/present"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/terminal"
)

type Backend interface {
	io.Writer
	Events() <-chan input.Event
	Size() (width, height int, err error)
	Sync() bool
	Exit() error
}

type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

type Tree struct {
	Root render.Node
	Keys []func(input.KeyEvent)
}

type Config struct {
	Clock    Clock
	Sheet    style.Sheet
	Profile  color.Profile
	Graphics *terminal.Graphics
}

type PanicError struct {
	Value any
	Stack []byte
}

func (p *PanicError) Error() string { return fmt.Sprintf("runtime: panic: %v\n%s", p.Value, p.Stack) }

type Runtime struct {
	cfg     Config
	wake    chan struct{}
	changed atomic.Bool

	mu      sync.Mutex
	queue   []func()
	running []func()

	app           func() Tree
	quitting      bool
	dirty         bool
	width, height int
	keys          []func(input.KeyEvent)
	screen        *present.Screen
	lastFrame     time.Time
	texts, stale  map[string]scene.Text
	sanitize      func(string) scene.Text
}

func New(cfg Config) *Runtime {
	r := &Runtime{
		cfg:   cfg,
		wake:  make(chan struct{}, 1),
		texts: map[string]scene.Text{},
		stale: map[string]scene.Text{},
	}
	r.sanitize = func(raw string) scene.Text {
		t, ok := r.texts[raw]
		if !ok {
			if t, ok = r.stale[raw]; !ok {
				t = scene.Sanitize(raw)
			}
			r.texts[raw] = t
		}
		return t
	}
	return r
}

func (r *Runtime) Dispatch(f func()) {
	r.mu.Lock()
	r.queue = append(r.queue, f)
	r.mu.Unlock()
	r.wakeUp()
}

func (r *Runtime) Invalidate() {
	r.changed.Store(true)
	r.wakeUp()
}

func (r *Runtime) wakeUp() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func (r *Runtime) Quit() { r.Dispatch(func() { r.quitting = true }) }

func (r *Runtime) Run(b Backend, app func() Tree) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = errors.Join(&PanicError{Value: v, Stack: debug.Stack()}, b.Exit())
		}
	}()
	r.app = app
	return errors.Join(r.loop(b), b.Exit())
}

func (r *Runtime) loop(b Backend) error {
	var err error
	if r.width, r.height, err = b.Size(); err != nil {
		return err
	}
	r.dirty = true
	events := b.Events()
	var throttle <-chan time.Time
	for {
		now := r.cfg.Clock.Now()
		if r.quitting {
			return nil
		}
		if r.changed.Swap(false) {
			r.dirty = true
		}
		if r.dirty && throttle == nil {
			if wait := r.lastFrame.Add(konst.FrameInterval).Sub(now); wait > 0 {
				throttle = r.cfg.Clock.After(wait)
			} else if err := r.frame(b, now); err != nil {
				return err
			}
		}
		select {
		case ev, ok := <-events:
			if !ok {
				return errors.New("runtime: input closed")
			}
			r.handle(ev)
		case <-r.wake:
			r.mu.Lock()
			r.queue, r.running = r.running[:0], r.queue
			r.mu.Unlock()
			for _, f := range r.running {
				f()
			}
			clear(r.running)
		case <-throttle:
			throttle = nil
		}
	}
}

func (r *Runtime) handle(ev input.Event) {
	switch ev := ev.(type) {
	case input.KeyEvent:
		if ev.Key == input.KeyRune && ev.Rune == 'c' && ev.Modifiers == input.ModCtrl && !ev.Release {
			r.quitting = true
			return
		}
		for _, h := range r.keys {
			h(ev)
		}
	case input.ResizeEvent:
		r.width, r.height, r.dirty = ev.Width, ev.Height, true
	case input.MouseEvent, input.PasteEvent, input.FocusEvent, input.ReplyEvent:
	default:
		panic(fmt.Sprintf("runtime: unknown event %T", ev))
	}
}

func (r *Runtime) frame(b Backend, now time.Time) error {
	tree := r.app()
	r.keys = tree.Keys
	root, err := render.Scene(tree.Root, render.Frame{
		Sheet:    r.cfg.Sheet,
		Width:    r.width,
		Height:   layout.Length{Unit: layout.Cells, Value: r.height},
		Sanitize: r.sanitize,
	})
	if err != nil {
		return err
	}
	r.texts, r.stale = r.stale, r.texts
	clear(r.texts)
	graphics, cell := r.surface(b)
	if r.screen == nil {
		r.screen = &present.Screen{Out: b, Profile: r.cfg.Profile, Graphics: graphics, Sync: b.Sync()}
	}
	r.screen.Cell = cell
	if err := r.screen.Frame(root, r.width, r.height); err != nil {
		return err
	}
	r.dirty, r.lastFrame = false, now
	if _, after := r.surface(b); after != cell {
		r.Invalidate()
	}
	return nil
}

func (r *Runtime) surface(b Backend) (terminal.Graphics, image.Point) {
	reporter, ok := b.(interface{ Capabilities() terminal.Capabilities })
	if !ok {
		return terminal.GraphicsNone, image.Point{}
	}
	caps := reporter.Capabilities()
	if r.cfg.Graphics != nil {
		caps.Graphics = *r.cfg.Graphics
	}
	if caps.Graphics == terminal.GraphicsNone || caps.CellPixels.X <= 0 || caps.CellPixels.Y <= 0 {
		return terminal.GraphicsNone, image.Point{}
	}
	return caps.Graphics, caps.CellPixels
}
