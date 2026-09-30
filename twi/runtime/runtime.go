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
	"github.com/twind-dev/twind/twi/events"
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
	Root   render.Node
	Keys   []func(input.KeyEvent)
	Events Node
}

type Config struct {
	Clock       Clock
	Sheet       style.Sheet
	Profile     color.Profile
	Graphics    *terminal.Graphics
	NoClipboard bool
}

type PanicError struct {
	Value any
	Stack []byte
}

func (p *PanicError) Error() string { return fmt.Sprintf("runtime: panic: %v\n%s", p.Value, p.Stack) }

type Runtime struct {
	cfg      Config
	wake     chan struct{}
	changed  atomic.Bool
	quitting atomic.Bool

	mu      sync.Mutex
	queue   []func()
	running []func()
	timers  []*Timer
	now     time.Time
	awake   bool

	app           func() Tree
	dirty         bool
	width, height int
	keys          []func(input.KeyEvent)
	doc           document
	focus         events.FocusManager[*Elem]
	screen        *present.Screen
	tree          render.Tree
	nodes         render.Node
	scene         scene.Node
	pointer       pointer
	sel           selection
	out           io.Writer
	pointed       bool
	ringless      bool
	revealed      *Elem
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

func (r *Runtime) Restyle(apply func()) {
	r.Dispatch(func() {
		apply()
		r.tree.Restyle()
		r.dirty = true
	})
}

func (r *Runtime) wakeUp() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func (r *Runtime) Quit() {
	r.quitting.Store(true)
	r.wakeUp()
}

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
	r.dirty, r.out = true, b
	events := b.Events()
	var ev input.Event
	var alarm <-chan time.Time
	var alarmAt time.Time
	for {
		now := r.cfg.Clock.Now()
		r.mu.Lock()
		r.now, r.awake = now, true
		r.mu.Unlock()
		if ev != nil {
			if err := r.handle(ev); err != nil {
				return err
			}
			ev = nil
		}
		r.drain()
		r.expire(now)
		if r.quitting.Load() {
			r.drain()
			return nil
		}
		if r.changed.Swap(false) {
			r.dirty = true
		}
		var frameAt time.Time
		if r.dirty || r.pointer.moved {
			if frameAt = r.lastFrame.Add(konst.FrameInterval); !frameAt.After(now) {
				frameAt = time.Time{}
				if err := r.draw(b, now); err != nil {
					return err
				}
			}
		}
		next := r.sleep()
		if next.IsZero() || !frameAt.IsZero() && frameAt.Before(next) {
			next = frameAt
		}
		switch {
		case next.IsZero():
		case !next.After(now):
			r.wakeUp()
		case alarm == nil || next.Before(alarmAt):
			alarm, alarmAt = r.cfg.Clock.After(next.Sub(now)), next
		}
		select {
		case e, ok := <-events:
			if !ok {
				return errors.New("runtime: input closed")
			}
			ev = e
		case <-r.wake:
		case <-alarm:
			alarm = nil
		}
	}
}

func (r *Runtime) drain() {
	r.mu.Lock()
	r.queue, r.running = r.running[:0], r.queue
	r.mu.Unlock()
	for _, f := range r.running {
		f()
	}
	clear(r.running)
}

func (r *Runtime) handle(ev input.Event) error {
	switch ev := ev.(type) {
	case input.KeyEvent:
		c := ev.Key == input.KeyRune && ev.Rune == 'c' && !ev.Release
		switch {
		case c && r.sel.shown && (ev.Modifiers == input.ModCtrl || ev.Modifiers == input.ModMeta):
			return r.copySelection()
		case c && ev.Modifiers == input.ModCtrl:
			r.quitting.Store(true)
			return nil
		case ev.Key == input.KeyEscape && !ev.Release && r.sel.shown:
			r.sel.clear()
			r.dirty = true
		}
		r.pointed = false
		prevented := r.focus.Key(&r.doc, ev).DefaultPrevented()
		r.refocused()
		if prevented {
			return nil
		}
		for _, h := range r.keys {
			h(ev)
		}
		r.activate(ev)
		r.scrollKey(ev)
	case input.MouseEvent:
		r.point(ev)
		r.refocused()
	case input.ResizeEvent:
		r.width, r.height, r.dirty = ev.Width, ev.Height, true
	case input.PasteEvent, input.FocusEvent, input.ReplyEvent:
	default:
		panic(fmt.Sprintf("runtime: unknown event %T", ev))
	}
	return nil
}

func (r *Runtime) activate(ev input.KeyEvent) {
	current, ok := r.focus.Current()
	pressed := ev.Key == input.KeyEnter || ev.Key == input.KeyRune && ev.Rune == ' '
	if ok && pressed && ev.Modifiers == 0 && !ev.Release && len(current.node.Click) > 0 && !r.doc.Disabled(current) {
		events.Dispatch(&r.doc, current, &events.Event[*Elem]{Type: events.Click, Key: ev})
	}
}

func (r *Runtime) refocused() {
	r.doc.moved(&r.focus)
	if current, _ := r.focus.Current(); current != r.revealed {
		r.dirty = true
	}
}

func (r *Runtime) draw(b Backend, now time.Time) error {
	if r.pointer.moved {
		r.hover()
		r.dirty = r.changed.Swap(false) || r.dirty
	}
	if !r.dirty {
		return nil
	}
	if err := r.frame(b, now); err != nil {
		return err
	}
	if !r.pointer.seen {
		return nil
	}
	r.hover()
	if r.dirty {
		r.wakeUp()
	}
	return nil
}

func (r *Runtime) frame(b Backend, now time.Time) error {
	tree := r.app()
	r.doc.update(tree.Events, &r.focus)
	if r.changed.Swap(false) {
		tree = r.app()
		r.doc.update(tree.Events, &r.focus)
	}
	r.keys = tree.Keys
	frame := render.Frame{
		Sheet:    r.cfg.Sheet,
		Width:    r.width,
		Height:   layout.Length{Unit: layout.Cells, Value: r.height},
		Sanitize: r.sanitize,
	}
	r.nodes = tree.Root
	current, ok := r.focus.Current()
	if current != r.revealed {
		r.ringless = r.pointed
	}
	if ok {
		at := style.StateFocus | style.StateFocusWithin
		if !r.ringless {
			at |= style.StateFocusVisible
		}
		tree.Root = mark(tree.Root, current.path(), at, style.StateFocusWithin)
	}
	if r.pointer.hovered != nil {
		tree.Root = mark(tree.Root, r.pointer.hovered, style.StateHover, style.StateHover)
	}
	if r.pointer.pressed != nil {
		tree.Root = mark(tree.Root, r.pointer.pressed, style.StateActive, style.StateActive)
	}
	root, err := r.tree.Scene(tree.Root, frame)
	if err != nil {
		return err
	}
	if current != r.revealed {
		r.revealed = current
		if current != nil && r.tree.ScrollIntoView(current.path()) {
			if root, err = r.tree.Scene(tree.Root, frame); err != nil {
				return err
			}
		}
	}
	r.scene = root
	r.texts, r.stale = r.stale, r.texts
	clear(r.texts)
	caps := capabilities(b)
	graphics, cell := r.surface(caps)
	if r.screen == nil {
		r.screen = &present.Screen{Out: b, Profile: r.cfg.Profile, Graphics: graphics, Sync: b.Sync(), Margins: caps.Margins}
	}
	r.screen.Cell = cell
	r.flow()
	if err := r.screen.Frame(r.highlight(root), r.width, r.height); err != nil {
		return err
	}
	r.dirty, r.lastFrame = false, now
	if _, after := r.surface(capabilities(b)); after != cell {
		r.Invalidate()
	}
	return nil
}

func capabilities(b Backend) terminal.Capabilities {
	if reporter, ok := b.(interface{ Capabilities() terminal.Capabilities }); ok {
		return reporter.Capabilities()
	}
	return terminal.Capabilities{}
}

func (r *Runtime) surface(caps terminal.Capabilities) (terminal.Graphics, image.Point) {
	if r.cfg.Graphics != nil {
		caps.Graphics = *r.cfg.Graphics
	}
	if r.cfg.Profile < color.ANSI256 || caps.Graphics == terminal.GraphicsNone || caps.CellPixels.X <= 0 || caps.CellPixels.Y <= 0 {
		return terminal.GraphicsNone, image.Point{}
	}
	return caps.Graphics, caps.CellPixels
}
