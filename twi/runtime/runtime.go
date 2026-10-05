package runtime

import (
	"errors"
	"fmt"
	"image"
	"io"
	"runtime/debug"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	konst "github.com/pehcastro/twind/internal/konst/runtime"
	"github.com/pehcastro/twind/internal/present"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/events"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/layout"
	"github.com/pehcastro/twind/twi/scene"
	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/terminal"
	"github.com/pehcastro/twind/twi/text"
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
	DevState    string
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
	built         Tree
	rebuild       bool
	dirty         bool
	width, height int
	keys          []func(input.KeyEvent)
	doc           document
	focus         events.FocusManager[*Elem]
	screen        *present.Screen
	tree          render.Tree
	nodes         render.Node
	scene         scene.Node
	walker        scene.Walker
	pointer       pointer
	sel           selection
	out           io.Writer
	pointed       bool
	ringless      bool
	ringsHidden   bool
	revealed      *Elem
	intoView      string
	measured      []*Ref
	lastFrame     time.Time
	texts, stale  map[string]scene.Text
	sanitize      func(string) scene.Text
	caps          terminal.Capabilities
	layers, shut  []layer
	opened        int
	start         time.Time
	motionAt      time.Time
	moving        bool
	animating     bool
	zoomed        image.Point
	dev           devState
	phases        phases
}

type layer struct {
	elem  *Elem
	order int
}

func New(cfg Config) *Runtime {
	r := &Runtime{
		cfg:   cfg,
		wake:  make(chan struct{}, 1),
		texts: map[string]scene.Text{},
		stale: map[string]scene.Text{},
	}
	r.doc.scene = &r.scene
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

func (r *Runtime) HideFocusRings() {
	r.ringsHidden = true
	r.Invalidate()
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
	r.phases.begin(b)
	if r.cfg.DevState == "" {
		return errors.Join(r.loop(b), b.Exit())
	}
	defer r.listenDev()()
	return errors.Join(r.loop(b), r.saveDev(), b.Exit())
}

func (r *Runtime) loop(b Backend) error {
	var err error
	if r.width, r.height, err = b.Size(); err != nil {
		return err
	}
	r.dirty, r.rebuild, r.out, r.start = true, true, b, r.cfg.Clock.Now()
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
		changed, due := r.changed.Swap(false), !r.motionAt.IsZero() && !r.motionDue().After(now)
		r.rebuild = r.rebuild || changed
		r.moving = due && !changed && !r.dirty && !r.pointer.moved
		if changed || due {
			r.dirty = true
		}
		var frameAt time.Time
		if r.dirty || r.pointer.moved {
			if frameAt = r.lastFrame.Add(konst.FrameInterval); !frameAt.After(now) {
				frameAt = time.Time{}
				if err := r.draw(b, now); err != nil {
					return err
				}
				now = r.cfg.Clock.Now()
			}
		}
		select {
		case <-alarm:
			alarm = nil
			continue
		default:
		}
		next := r.sleep()
		if next.IsZero() || !frameAt.IsZero() && frameAt.Before(next) {
			next = frameAt
		}
		if due := r.motionDue(); !r.motionAt.IsZero() && (next.IsZero() || due.Before(next)) {
			next = due
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

func (r *Runtime) motionDue() time.Time {
	interval := konst.FrameInterval
	if r.animating {
		interval = konst.MotionInterval
	}
	if paced := r.lastFrame.Add(interval); paced.After(r.motionAt) {
		return paced
	}
	return r.motionAt
}

func (r *Runtime) drain() {
	r.mu.Lock()
	r.queue, r.running = r.running[:0], r.queue
	r.mu.Unlock()
	r.rebuild = r.rebuild || len(r.running) > 0
	for _, f := range r.running {
		f()
	}
	clear(r.running)
}

func (r *Runtime) handle(ev input.Event) error {
	r.phases.seen(ev)
	switch ev := ev.(type) {
	case input.KeyEvent:
		r.rebuild = true
		c := ev.Key == input.KeyRune && ev.Rune == 'c' && !ev.Release
		switch {
		case c && r.sel.shown && (ev.Modifiers == input.ModCtrl || ev.Modifiers == input.ModMeta):
			return r.Copy(r.sel.text())
		case c && ev.Modifiers == input.ModCtrl:
			r.quitting.Store(true)
			return nil
		case ev.Key == input.KeyEscape && !ev.Release && r.sel.shown:
			r.sel.clear()
			r.dirty = true
		}
		if !ev.Release && r.hotkey(ev) {
			return nil
		}
		r.dirty = r.dirty || r.ringless
		r.pointed, r.ringless = false, false
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
		r.rebuild = r.rebuild || ev.Action == input.MousePress || ev.Action == input.MouseRelease
		r.point(ev)
		r.refocused()
	case input.ResizeEvent:
		r.width, r.height, r.dirty, r.rebuild = ev.Width, ev.Height, true, true
		if ev.Cell != (image.Point{}) {
			r.zoomed = ev.Cell
		}
	case input.PasteEvent:
		r.rebuild = true
		for e, _ := r.focus.Current(); e != nil; e = e.parent {
			if e.node.Paste != nil {
				e.node.Paste(ev.Text)
				break
			}
		}
	case input.FocusEvent, input.ReplyEvent:
	default:
		panic(fmt.Sprintf("runtime: unknown event %T", ev))
	}
	return nil
}

func (r *Runtime) hotkey(ev input.KeyEvent) bool {
	return slices.ContainsFunc(r.doc.hotkeys, func(e *Elem) bool { return e.node.Hotkey(ev) })
}

func (r *Runtime) activate(ev input.KeyEvent) {
	current, ok := r.focus.Current()
	pressed := ev.Key == input.KeyEnter || ev.Key == input.KeyRune && ev.Rune == ' '
	if ok && pressed && ev.Modifiers == 0 && !ev.Release && current.clickable() && !r.doc.Disabled(current) {
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
		r.move()
		if r.changed.Swap(false) {
			r.dirty, r.rebuild = true, true
		}
	}
	if !r.dirty {
		return nil
	}
	if err := r.frame(b, now); err != nil {
		return err
	}
	r.phases.frame()
	if r.pointer.seen {
		r.hover()
	}
	if r.dirty {
		r.wakeUp()
	}
	return nil
}

func (r *Runtime) Widths() text.Widths { return r.caps.Widths }

func (r *Runtime) frame(b Backend, now time.Time) error {
	r.caps, r.dirty = r.capabilities(b), false
	tree := r.view()
	r.doc.update(tree.Events, &r.focus)
	if r.changed.Swap(false) {
		r.rebuild = true
		tree = r.view()
		r.doc.update(tree.Events, &r.focus)
	}
	graphics, cell := r.surface(r.caps)
	frame := render.Frame{
		Sheet:    r.cfg.Sheet,
		Width:    r.width,
		Height:   layout.Length{Unit: layout.Cells, Value: r.height},
		Sanitize: r.sanitize,
		Cell:     r.caps.CellPixels,
		Widths:   r.caps.Widths,
		Now:      now.Sub(r.start),
		Graphics: graphics != terminal.GraphicsNone,
	}
	r.restoreFocus()
	current, _ := r.focus.Current()
	if current != r.revealed {
		r.ringless = r.pointed
	}
	root := r.marked(tree)
	var err error
	if r.scene, err = r.tree.Scene(root, frame); err != nil {
		return err
	}
	moved := current != r.revealed && current != nil && r.tree.ScrollIntoView(current.path())
	r.revealed = current
	moved = r.restoreScroll() || moved
	if r.intoView != "" {
		moved = r.scrollIntoView() || moved
	}
	if moved {
		if r.scene, err = r.tree.Scene(root, frame); err != nil {
			return err
		}
	}
	for pass := 1; ; pass++ {
		moved := r.measure()
		r.scrolled()
		if pass < konst.MeasurePasses && r.intoView != "" {
			moved = r.scrollIntoView() || moved
		}
		if pass == konst.MeasurePasses || !r.changed.Swap(false) && !moved {
			break
		}
		r.rebuild = true
		tree = r.view()
		r.doc.update(tree.Events, &r.focus)
		if r.scene, err = r.tree.Scene(r.marked(tree), frame); err != nil {
			return err
		}
	}
	r.animating, r.motionAt = !r.motionAt.IsZero(), time.Time{}
	if at, moving := r.tree.Wake(); moving {
		r.motionAt = r.start.Add(at)
	}
	r.texts, r.stale = r.stale, r.texts
	clear(r.texts)
	if r.screen == nil {
		r.screen = &present.Screen{Out: b, Profile: r.cfg.Profile, Sync: b.Sync(), Margins: r.caps.Margins}
		if c, ok := b.(interface{ Covers(cluster string) bool }); ok {
			r.screen.Covers = c.Covers
		}
		if p, ok := b.(interface{ Paint(terminal.Pixels) bool }); ok {
			r.screen.Paint = p.Paint
		}
		if m, ok := b.(interface {
			Marks(color.RGBA) []terminal.Mark
		}); ok {
			r.screen.Marks = m.Marks
		}
	}
	r.screen.Graphics, r.screen.Cell, r.screen.Widths, r.screen.Font, r.screen.Identity, r.screen.Workers = graphics, cell, r.caps.Widths, r.caps.Font, r.caps.Identity, 0
	if r.moving {
		r.screen.Workers = 1
	}
	r.flow()
	if err := r.screen.Frame(r.highlight(r.scene), r.width, r.height); err != nil {
		return err
	}
	r.lastFrame = now
	if r.capabilities(b) != r.caps {
		r.Invalidate()
	}
	return nil
}

func (r *Runtime) view() Tree {
	if r.rebuild || r.doc.heard {
		r.built, r.rebuild, r.doc.heard = r.app(), false, false
	}
	return r.built
}

func (r *Runtime) marked(tree Tree) render.Node {
	r.keys = tree.Keys
	root := r.number(tree.Root)
	r.nodes = root
	if current, ok := r.focus.Current(); ok {
		at := style.StateFocus | style.StateFocusWithin
		if !r.ringsHidden && (!r.ringless || typing(root, current.path())) {
			at |= style.StateFocusVisible
		}
		root = mark(root, current.path(), at, style.StateFocusWithin)
	}
	if r.pointer.hovered != nil {
		root = mark(root, r.pointer.hovered, style.StateHover, style.StateHover)
	}
	if r.pointer.pressed != nil {
		root = mark(root, r.pointer.pressed, style.StateActive, style.StateActive)
	}
	return root
}

func typing(n render.Node, path []int) bool {
	for _, i := range path {
		if i >= len(n.Children) {
			return false
		}
		n = n.Children[i]
	}
	return n.Element == style.ElementInput || n.Element == style.ElementTextarea
}

func (r *Runtime) number(root render.Node) render.Node {
	open := r.shut[:0]
	for _, e := range r.doc.layers {
		if i := slices.IndexFunc(r.layers, func(l layer) bool { return l.elem == e }); i >= 0 {
			open = append(open, r.layers[i])
		} else {
			r.opened++
			open = append(open, layer{e, r.opened})
		}
		root = lift(root, e.path(), open[len(open)-1].order)
	}
	if len(open) == 0 {
		r.opened = 0
	}
	r.layers, r.shut = open, r.layers
	return root
}

func lift(n render.Node, path []int, order int) render.Node {
	if len(path) == 0 {
		n.TopLayer = order
		return n
	}
	n.Children = slices.Clone(n.Children)
	n.Children[path[0]] = lift(n.Children[path[0]], path[1:], order)
	return n
}

func (r *Runtime) capabilities(b Backend) terminal.Capabilities {
	reporter, ok := b.(interface{ Capabilities() terminal.Capabilities })
	if !ok {
		return terminal.Capabilities{}
	}
	caps := reporter.Capabilities()
	if caps.CellPixels == r.zoomed {
		r.zoomed = image.Point{}
	}
	if r.zoomed != (image.Point{}) {
		caps.CellPixels = r.zoomed
	}
	return caps
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
