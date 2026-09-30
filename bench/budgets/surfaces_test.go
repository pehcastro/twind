package budgets_test

import (
	"bytes"
	"image"
	"runtime"
	"testing"
	"time"

	"github.com/twind-dev/twind/bench/scenarios/surfaces"
	playground "github.com/twind-dev/twind/examples/playground/app"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/terminal"
)

type surfaceBackend struct {
	events  chan input.Event
	written chan []byte
	caps    terminal.Capabilities
}

func (be *surfaceBackend) Write(p []byte) (int, error) {
	be.written <- p
	return len(p), nil
}

func (be *surfaceBackend) Events() <-chan input.Event          { return be.events }
func (be *surfaceBackend) Sync() bool                          { return true }
func (be *surfaceBackend) Exit() error                         { return nil }
func (be *surfaceBackend) Capabilities() terminal.Capabilities { return be.caps }
func (be *surfaceBackend) Size() (width, height int, err error) {
	return surfaces.Columns, surfaces.Rows, nil
}

type frames struct{ n, bytes, image, tiles int }

func (f *frames) add(p []byte) {
	f.n++
	f.bytes += len(p)
	for {
		start := bytes.IndexByte(p, '\x1b')
		if start < 0 {
			return
		}
		if !bytes.HasPrefix(p[start:], []byte("\x1bP")) && !bytes.HasPrefix(p[start:], []byte("\x1b_G")) {
			p = p[start+1:]
			continue
		}
		end := start + bytes.Index(p[start:], []byte("\x1b\\")) + 2
		f.image += end - start
		f.tiles++
		p = p[end:]
	}
}

func (f frames) report(b *testing.B) {
	b.ReportMetric(float64(f.bytes)/float64(f.n), "bytes/frame")
	b.ReportMetric(float64(f.image)/float64(f.n), "image-bytes/frame")
	b.ReportMetric(float64(f.tiles)/float64(f.n), "tiles/frame")
}

type scenario struct {
	app  func(*twi.Runtime) func() twi.Node
	opts []twi.RenderOption
}

func (sc scenario) start(caps terminal.Capabilities) (*surfaceBackend, *twi.Runtime, chan error) {
	be := &surfaceBackend{events: make(chan input.Event), written: make(chan []byte), caps: caps}
	rt := twi.New(append([]twi.RenderOption{twi.Backend(be, &steppingClock{}), twi.ColorProfile(color.TrueColor)}, sc.opts...)...)
	done := make(chan error, 1)
	go func() { done <- rt.Run(sc.app(rt)) }()
	return be, rt, done
}

func settle(be *surfaceBackend, rt *twi.Runtime) (written int) {
	for {
		quiet, marker := true, make(chan struct{})
		rt.Dispatch(func() { rt.Dispatch(func() { close(marker) }) })
		for waiting := true; waiting; {
			select {
			case p := <-be.written:
				written, quiet = written+len(p), false
			case <-marker:
				waiting = false
			}
		}
		if quiet {
			return written
		}
	}
}

func stop(b *testing.B, be *surfaceBackend, rt *twi.Runtime, done chan error) {
	rt.Quit()
	for {
		select {
		case <-be.written:
		case err := <-done:
			if err != nil {
				b.Fatal(err)
			}
			return
		}
	}
}

func key(r rune) input.Event { return input.KeyEvent{Key: input.KeyRune, Rune: r} }

type step struct {
	name   string
	before func(i int) []input.Event
	timed  input.Event
	after  []input.Event
}

type timing struct {
	now          func() time.Duration
	out          frames
	extra        int
	first, quiet []time.Duration
	collections  uint32
}

func (t *timing) frame(b *testing.B, be *surfaceBackend, rt *twi.Runtime, begin time.Duration) {
	p := <-be.written
	t.first = append(t.first, t.now()-begin)
	t.extra += settle(be, rt)
	t.quiet = append(t.quiet, t.now()-begin)
	b.StopTimer()
	t.out.add(p)
}

func (t *timing) start(b *testing.B) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	t.collections = m.NumGC
	b.ReportAllocs()
	b.ResetTimer()
}

func (t *timing) report(b *testing.B) {
	b.StopTimer()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	percentiles(b, "", t.first)
	percentiles(b, "settled-", t.quiet)
	t.out.report(b)
	b.ReportMetric(float64(t.extra)/float64(len(t.first)), "extra-bytes/op")
	b.ReportMetric(float64(m.NumGC-t.collections)/float64(len(t.first)), "gc/op")
}

func (sc scenario) first(b *testing.B, caps terminal.Capabilities) {
	t := timing{now: clock(b)}
	t.start(b)
	for range b.N {
		begin := t.now()
		be, rt, done := sc.start(caps)
		t.frame(b, be, rt, begin)
		stop(b, be, rt, done)
		b.StartTimer()
	}
	t.report(b)
}

func (sc scenario) step(b *testing.B, caps terminal.Capabilities, s step) {
	t := timing{now: clock(b)}
	be, rt, done := sc.start(caps)
	startup := len(<-be.written) + settle(be, rt)
	send := func(events []input.Event) {
		for _, e := range events {
			be.events <- e
			settle(be, rt)
		}
	}
	t.start(b)
	for i := range b.N {
		if s.before != nil {
			b.StopTimer()
			send(s.before(i))
			b.StartTimer()
		}
		begin := t.now()
		be.events <- s.timed
		t.frame(b, be, rt, begin)
		send(s.after)
		b.StartTimer()
	}
	t.report(b)
	stop(b, be, rt, done)
	b.ReportMetric(float64(startup), "first-bytes/frame")
}

func sixel() terminal.Capabilities {
	return terminal.Capabilities{Sync: true, Graphics: terminal.GraphicsSixel, CellPixels: image.Pt(10, 20)}
}

func BenchmarkSurface(b *testing.B) {
	sheet, err := surfaces.Styles()
	if err != nil {
		b.Fatal(err)
	}
	sc := scenario{surfaces.App, []twi.RenderOption{twi.Styles(sheet), twi.Theme(surfaces.Themes()[0])}}
	arms := []struct {
		name      string
		caps      terminal.Capabilities
		firstOnly bool
	}{
		{"sixel", sixel(), false},
		{"kitty", terminal.Capabilities{Sync: true, Graphics: terminal.GraphicsKitty, CellPixels: image.Pt(10, 20)}, true},
		{"none", terminal.Capabilities{Sync: true}, false},
	}
	steps := []step{
		{name: "text", timed: key(surfaces.Count)},
		{name: "hover", timed: key(surfaces.Hover)},
		{name: "theme", timed: key(surfaces.Theme)},
		{name: "resize", timed: input.ResizeEvent{Width: surfaces.Wide, Height: surfaces.Rows}, after: []input.Event{input.ResizeEvent{Width: surfaces.Columns, Height: surfaces.Rows}}},
	}
	for _, arm := range arms {
		b.Run(arm.name+"/first", func(b *testing.B) { sc.first(b, arm.caps) })
		for _, s := range steps {
			if arm.firstOnly {
				break
			}
			b.Run(arm.name+"/"+s.name, func(b *testing.B) { sc.step(b, arm.caps, s) })
		}
	}
}

func BenchmarkPlayground(b *testing.B) {
	sheet, err := playground.Styles()
	if err != nil {
		b.Fatal(err)
	}
	sc := scenario{playground.App, []twi.RenderOption{twi.Styles(sheet)}}
	enter := input.KeyEvent{Key: input.KeyEnter}
	steps := []step{
		{name: "type", timed: key('a'), after: []input.Event{input.KeyEvent{Key: input.KeyBackspace}}},
		{name: "theme", before: func(i int) []input.Event {
			move := map[bool]input.Key{true: input.KeyArrowDown, false: input.KeyArrowUp}[i%2 == 0]
			return []input.Event{key('t'), enter, input.KeyEvent{Key: move}}
		}, timed: enter},
	}
	b.Run("sixel/first", func(b *testing.B) { sc.first(b, sixel()) })
	for _, s := range steps {
		b.Run("sixel/"+s.name, func(b *testing.B) { sc.step(b, sixel(), s) })
	}
}
