package drive

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	konst "github.com/twind-dev/twind/internal/konst/drive"
	ikonst "github.com/twind-dev/twind/internal/konst/input"
	rkonst "github.com/twind-dev/twind/internal/konst/runtime"
	tkonst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/style"
)

type App func(rt *twi.Runtime) func() twi.Node

type Option func(*config)

type config struct {
	width, height int
	sheet         style.Sheet
}

func Size(width, height int) Option {
	return func(c *config) { c.width, c.height = width, height }
}

func Styles(sheet style.Sheet) Option { return func(c *config) { c.sheet = sheet } }

type Driver struct {
	rt      *twi.Runtime
	clock   *clock
	events  chan input.Event
	decoder input.Decoder
	screen  *screen
	exited  chan struct{}
	runErr  error
	err     error
	held    bool
}

func New(app App, opts ...Option) *Driver {
	cfg := config{width: konst.Width, height: konst.Height}
	for _, o := range opts {
		o(&cfg)
	}
	d := &Driver{
		clock:  &clock{now: time.Unix(konst.EpochUnix, 0).UTC()},
		events: make(chan input.Event),
		screen: &screen{cells: buffer.New(cfg.width, cfg.height)},
		exited: make(chan struct{}),
	}
	d.rt = twi.New(twi.Backend(&backend{d.screen, d.events}, d.clock), twi.Styles(cfg.sheet), twi.ColorProfile(color.TrueColor))
	view := app(d.rt)
	go func() {
		d.runErr = d.rt.Run(view)
		close(d.exited)
	}()
	d.settle()
	return d
}

func (d *Driver) Press(name string) {
	k, err := parseKey(name)
	if err != nil {
		d.fail(err)
		return
	}
	d.feed(input.Encode(k))
}

func (d *Driver) Type(s string) {
	if strings.ContainsFunc(s, unicode.IsControl) {
		d.fail(fmt.Errorf("drive: Type takes text, press a control key by name: %q", s))
		return
	}
	d.feed([]byte(s))
}

func (d *Driver) Wheel(x, y, notches int) {
	button := tkonst.WheelDownReport
	if notches < 0 {
		button, notches = tkonst.WheelUpReport, -notches
	}
	for range notches {
		d.report("wheel", x, y, button, 'M')
	}
}

func (d *Driver) Move(x, y int) {
	button := ikonst.MouseMotion | ikonst.MouseNoButton
	if d.held {
		button = ikonst.MouseMotion
	}
	d.report("move", x, y, button, 'M')
}

func (d *Driver) Down(x, y int) {
	d.held = true
	d.report("down", x, y, 0, 'M')
}

func (d *Driver) Up(x, y int) {
	d.held = false
	d.report("up", x, y, 0, 'm')
}

func (d *Driver) Click(x, y int) {
	d.Down(x, y)
	d.Up(x, y)
}

func (d *Driver) report(verb string, x, y, button int, final byte) {
	if x < 0 || y < 0 || x >= d.screen.cells.Width() || y >= d.screen.cells.Height() {
		d.fail(fmt.Errorf("drive: %s at %d,%d is off the %dx%d screen", verb, x, y, d.screen.cells.Width(), d.screen.cells.Height()))
		return
	}
	d.feed(fmt.Appendf(nil, "%s<%d;%d;%d%c", ikonst.CSI, button, x+1, y+1, final))
}

func (d *Driver) Resize(width, height int) {
	if width < 1 || height < 1 {
		d.fail(fmt.Errorf("drive: cannot resize to %dx%d", width, height))
	}
	if d.err != nil {
		return
	}
	d.screen.cells.Resize(width, height)
	d.feed(fmt.Appendf(nil, "%s%d;%d;%d;0;0t", ikonst.CSI, ikonst.InBandResize, height, width))
}

func (d *Driver) Advance(dt time.Duration) {
	d.settle()
	until := d.clock.Now().Add(dt)
	for {
		t, due := d.clock.advance(until)
		if !due {
			return
		}
		t.fire <- t.at
		d.settle()
	}
}

func (d *Driver) Frame() Frame { return d.screen.frame() }

func (d *Driver) Err() error { return d.err }

func (d *Driver) Clipboard() string { return d.screen.clipboard }

func (d *Driver) Close() error {
	d.rt.Quit()
	<-d.exited
	return d.runErr
}

func (d *Driver) fail(err error) {
	if d.err == nil {
		d.err = err
	}
}

func (d *Driver) feed(b []byte) {
	if d.err != nil {
		return
	}
	for _, ev := range append(d.decoder.Decode(b), d.decoder.Quiet()...) {
		select {
		case d.events <- ev:
		case <-d.exited:
			d.fail(errors.Join(errors.New("drive: the app has exited"), d.runErr))
			return
		}
	}
	d.Advance(rkonst.FrameInterval)
}

func (d *Driver) settle() {
	done := make(chan struct{})
	d.rt.Dispatch(func() { d.rt.Dispatch(func() { d.rt.Dispatch(func() { close(done) }) }) })
	select {
	case <-done:
	case <-d.exited:
		d.fail(d.runErr)
	}
}

type backend struct {
	*screen
	events chan input.Event
}

func (b *backend) Events() <-chan input.Event { return b.events }

func (b *backend) Size() (width, height int, err error) {
	return b.cells.Width(), b.cells.Height(), nil
}

func (b *backend) Sync() bool { return true }

func (b *backend) Exit() error { return nil }

type timer struct {
	at   time.Time
	fire chan time.Time
}

type clock struct {
	mu     sync.Mutex
	now    time.Time
	timers []timer
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := timer{at: c.now.Add(d), fire: make(chan time.Time, 1)}
	c.timers = append(c.timers, t)
	return t.fire
}

func (c *clock) advance(until time.Time) (timer, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	next := -1
	for i, t := range c.timers {
		if !t.at.After(until) && (next < 0 || t.at.Before(c.timers[next].at)) {
			next = i
		}
	}
	if next < 0 {
		c.now = until
		return timer{}, false
	}
	t := c.timers[next]
	c.timers = append(c.timers[:next], c.timers[next+1:]...)
	c.now = t.at
	return t, true
}
