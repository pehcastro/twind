package runtime_test

import (
	"errors"
	"image"
	"regexp"
	goruntime "runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/runtime"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/testdata/counter"
	"github.com/twind-dev/twind/twi/testdata/hello"
)

type backend struct {
	events        chan input.Event
	width, height int
	frames        chan string
	exits         atomic.Int32
	caps          terminal.Capabilities
	zoom          image.Point
}

func newBackend(width, height int) *backend {
	return &backend{events: make(chan input.Event), width: width, height: height, frames: make(chan string, 1<<16)}
}

func (b *backend) Write(p []byte) (int, error) {
	b.frames <- string(p)
	if b.zoom != (image.Point{}) {
		b.caps.CellPixels, b.zoom = b.zoom, image.Point{}
	}
	return len(p), nil
}

func (b *backend) Capabilities() terminal.Capabilities  { return b.caps }
func (b *backend) Events() <-chan input.Event           { return b.events }
func (b *backend) Size() (width, height int, err error) { return b.width, b.height, nil }
func (b *backend) Sync() bool                           { return false }
func (b *backend) Exit() error {
	b.exits.Add(1)
	return nil
}

type clock struct{ wakes atomic.Int64 }

func (c *clock) Now() time.Time {
	c.wakes.Add(1)
	return time.Now()
}

func (c *clock) After(d time.Duration) <-chan time.Time { return time.After(d) }

type run struct {
	rt    *twi.Runtime
	b     *backend
	clock *clock
	done  chan error
}

func start(build func(rt *twi.Runtime) func() twi.Node) run {
	return launch(newBackend(20, 3), build)
}

func launch(b *backend, build func(rt *twi.Runtime) func() twi.Node, opts ...twi.RenderOption) run {
	r := run{b: b, clock: &clock{}, done: make(chan error, 1)}
	r.rt = twi.New(append([]twi.RenderOption{twi.Backend(r.b, r.clock), twi.ColorProfile(color.None)}, opts...)...)
	app := build(r.rt)
	go func() { r.done <- r.rt.Run(app) }()
	return r
}

func static(app func() twi.Node) func(*twi.Runtime) func() twi.Node {
	return func(*twi.Runtime) func() twi.Node { return app }
}

func (r run) next(t *testing.T) string {
	t.Helper()
	timeout := time.NewTimer(2 * time.Second)
	defer timeout.Stop()
	select {
	case f := <-r.b.frames:
		return f
	case <-timeout.C:
		t.Fatal("no frame within 2s")
		return ""
	}
}

func (r run) quiet(t *testing.T) {
	t.Helper()
	select {
	case f := <-r.b.frames:
		t.Fatalf("unexpected frame %q", f)
	case <-time.After(100 * time.Millisecond):
	}
}

func (r run) result(t *testing.T) error {
	t.Helper()
	timeout := time.NewTimer(2 * time.Second)
	defer timeout.Stop()
	select {
	case err := <-r.done:
		return err
	case <-timeout.C:
		t.Fatal("Run did not return within 2s")
		return nil
	}
}

func (r run) stop(t *testing.T) error {
	t.Helper()
	r.rt.Quit()
	return r.result(t)
}

func key(r rune) input.KeyEvent { return input.KeyEvent{Rune: r} }

func runtimeAllocations() int64 {
	for range 3 {
		goruntime.GC()
	}
	n, _ := goruntime.MemProfile(nil, true)
	records := make([]goruntime.MemProfileRecord, n+64)
	n, _ = goruntime.MemProfile(records, true)
	var total int64
	for _, rec := range records[:n] {
		frames := goruntime.CallersFrames(rec.Stack())
		for more := true; more; {
			var f goruntime.Frame
			f, more = frames.Next()
			if strings.HasPrefix(f.Function, "github.com/twind-dev/twind/twi/runtime.") {
				total += rec.AllocObjects
				break
			}
		}
	}
	return total
}

func TestIdle(t *testing.T) {
	rate := goruntime.MemProfileRate
	goruntime.MemProfileRate = 1
	defer func() { goruntime.MemProfileRate = rate }()
	r := start(counter.New)
	r.next(t)
	time.Sleep(50 * time.Millisecond)
	var before, after goruntime.MemStats
	wakes, owned := r.clock.wakes.Load(), runtimeAllocations()
	goruntime.ReadMemStats(&before)
	time.Sleep(time.Second)
	goruntime.ReadMemStats(&after)
	idleWakes, idleOwned := r.clock.wakes.Load()-wakes, runtimeAllocations()-owned
	t.Logf("idle 1s: %d wakes, %d allocations under twi/runtime; process-wide %d allocations, %d bytes", idleWakes, idleOwned, after.Mallocs-before.Mallocs, after.TotalAlloc-before.TotalAlloc)
	if idleWakes != 0 || idleOwned != 0 {
		t.Fatalf("idle runtime woke %d times and allocated %d times", idleWakes, idleOwned)
	}
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
}

func TestCounterFramePerKey(t *testing.T) {
	r := start(counter.New)
	first := r.next(t)
	t.Logf("first frame %q", first)
	if !strings.Contains(first, "count 0") {
		t.Fatalf("first frame %q", first)
	}
	for i := 1; i <= 5; i++ {
		r.b.events <- key('+')
		if f := r.next(t); !strings.HasSuffix(f, strconv.Itoa(i)) {
			t.Fatalf("frame after key %d is %q", i, f)
		}
	}
	r.b.events <- key('x')
	r.b.events <- key('-')
	if f := r.next(t); !strings.HasSuffix(f, "4") {
		t.Fatalf("frame after - is %q", f)
	}
	r.quiet(t)
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
	if n := len(r.b.frames); n != 0 {
		t.Fatalf("%d frames after the last key", n)
	}
}

func TestHundredSetsOneFrame(t *testing.T) {
	var s *twi.Signal[int]
	r := start(func(rt *twi.Runtime) func() twi.Node {
		s = twi.NewSignal(rt, 0)
		return func() twi.Node {
			return twi.Element(
				twi.OnKey(func(input.KeyEvent) {
					for range 100 {
						s.Set(s.Get() + 1)
					}
				}),
				twi.Text("n="+strconv.Itoa(s.Get())),
			)
		}
	})
	r.next(t)
	r.b.events <- key('x')
	if f := r.next(t); !strings.HasSuffix(f, "100") {
		t.Fatalf("frame after 100 sets in one handler is %q", f)
	}
	r.quiet(t)

	blocked, release := make(chan struct{}), make(chan struct{})
	r.rt.Dispatch(func() {
		close(blocked)
		<-release
	})
	<-blocked
	for range 100 {
		r.rt.Dispatch(func() { s.Set(s.Get() + 1) })
	}
	close(release)
	if f := r.next(t); !strings.HasSuffix(f, "2") {
		t.Fatalf("frame after 100 dispatched sets is %q", f)
	}
	r.quiet(t)
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
}

func TestDispatchFromGoroutines(t *testing.T) {
	var s *twi.Signal[int]
	r := start(func(rt *twi.Runtime) func() twi.Node {
		s = twi.NewSignal(rt, 0)
		return func() twi.Node { return twi.Text(strconv.Itoa(s.Get())) }
	})
	r.next(t)
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			for range 1000 {
				r.rt.Dispatch(func() { s.Set(s.Get() + 1) })
			}
		})
	}
	wg.Wait()
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
	frames, turns := len(r.b.frames)+1, r.clock.wakes.Load()
	t.Logf("final %d, %d frames over %d scheduler turns", s.Get(), frames, turns)
	if s.Get() != 50000 {
		t.Fatalf("final state %d, want 50000", s.Get())
	}
	if int64(frames) > turns {
		t.Fatalf("%d frames over %d turns", frames, turns)
	}
}

func TestSignalWakesItsOwnRuntime(t *testing.T) {
	var s *twi.Signal[int]
	a := start(func(rt *twi.Runtime) func() twi.Node {
		s = twi.NewSignal(rt, 0)
		return func() twi.Node { return twi.Text("a" + strconv.Itoa(s.Get())) }
	})
	a.next(t)
	b := start(static(func() twi.Node {
		return twi.Element(
			twi.OnKey(func(input.KeyEvent) { s.Set(s.Get() + 1) }),
			twi.Text("b"+strconv.Itoa(s.Get())),
		)
	}))
	b.next(t)
	b.b.events <- key('+')
	if f := a.next(t); !strings.HasSuffix(f, "1") {
		t.Fatalf("owner's frame after the set is %q", f)
	}
	b.quiet(t)
	for _, r := range []run{a, b} {
		if err := r.stop(t); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPanicExitsOnce(t *testing.T) {
	r := start(static(func() twi.Node {
		return twi.Element(twi.OnKey(func(input.KeyEvent) { panic("boom") }), twi.Text("x"))
	}))
	r.next(t)
	r.b.events <- key('p')
	err := r.result(t)
	exits := r.b.exits.Load()
	var p *runtime.PanicError
	if !errors.As(err, &p) || p.Value != "boom" {
		t.Fatalf("Run returned %v", err)
	}
	if exits != 1 {
		t.Fatalf("Exit called %d times before Run returned the panic", exits)
	}
}

func TestOnKeyTreeOrder(t *testing.T) {
	var order []string
	listen := func(name string) twi.NodeOption {
		return twi.OnKey(func(input.KeyEvent) { order = append(order, name) })
	}
	r := start(static(func() twi.Node {
		return twi.Element(
			twi.Element(listen("a"), twi.Element(listen("a1"))),
			listen("root"),
			twi.Element(listen("b")),
		)
	}))
	r.next(t)
	r.b.events <- key('k')
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(order, " "); got != "root a a1 b" {
		t.Fatalf("handlers ran in order %q", got)
	}
}

func TestResizeRedrawsEverything(t *testing.T) {
	r := start(counter.New)
	r.next(t)
	r.b.events <- input.ResizeEvent{Width: 10, Height: 2}
	if f := r.next(t); !strings.Contains(f, "count 0") {
		t.Fatalf("frame after resize is %q", f)
	}
	r.b.events <- key('+')
	r.next(t)
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
}

func TestCtrlCQuits(t *testing.T) {
	r := start(counter.New)
	r.next(t)
	r.b.events <- input.KeyEvent{Rune: 'c', Modifiers: input.ModCtrl}
	if err := r.result(t); err != nil || r.b.exits.Load() != 1 {
		t.Fatalf("Ctrl+C: Run returned %v, Exit called %d times", err, r.b.exits.Load())
	}
}

func TestInputClosed(t *testing.T) {
	r := start(counter.New)
	r.next(t)
	close(r.b.events)
	if err := r.result(t); err == nil {
		t.Fatal("Run returned nil after its input closed")
	}
}

func TestRunNeedsAMode(t *testing.T) {
	rt := twi.New()
	if err := rt.Run(counter.New(rt)); err == nil || !strings.Contains(err.Error(), "Fullscreen") {
		t.Fatalf("Run without a mode returned %v", err)
	}
}

func TestTextIsSanitised(t *testing.T) {
	r := start(static(func() twi.Node { return twi.Text("a\x1b[2Jb") }))
	if f := r.next(t); strings.Contains(f, "\x1b[2J") || !strings.Contains(f, "a") {
		t.Fatalf("frame %q", f)
	}
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
}

func TestBackendNeedsAColorProfile(t *testing.T) {
	r := run{b: newBackend(20, 3), done: make(chan error, 1)}
	r.rt = twi.New(twi.Backend(r.b, &clock{}))
	go func() { r.done <- r.rt.Run(counter.New(r.rt)) }()
	if err := r.result(t); err == nil || !strings.Contains(err.Error(), "ColorProfile") {
		t.Fatalf("Run on a backend without a colour profile returned %v", err)
	}
}

var sixelAt = regexp.MustCompile(`\x1b\[(\d+);(\d+)H\x1bP`)

func sixelTiles(frame string) []string {
	var at []string
	for _, m := range sixelAt.FindAllStringSubmatch(frame, -1) {
		at = append(at, m[1]+";"+m[2])
	}
	slices.Sort(at)
	return at
}

func surfaces(t *testing.T, b *backend, opts ...twi.RenderOption) run {
	t.Helper()
	s, err := hello.Styles()
	if err != nil {
		t.Fatal(err)
	}
	return launch(b, hello.Surfaces, append([]twi.RenderOption{twi.Styles(s), twi.ColorProfile(color.TrueColor)}, opts...)...)
}

func TestSixelSendsOnlyChangedSurfaces(t *testing.T) {
	b := newBackend(40, 15)
	b.caps = terminal.Capabilities{Graphics: terminal.GraphicsSixel, CellPixels: image.Pt(10, 20)}
	r := surfaces(t, b)
	if first := r.next(t); len(sixelTiles(first)) == 0 {
		t.Fatalf("first frame on a Sixel backend sent no tile: %q", first)
	}
	r.b.events <- key('t')
	if f := r.next(t); strings.Contains(f, "\x1bP") || !strings.Contains(f, "1") {
		t.Fatalf("a text-only key wrote image bytes or no text: %q", f)
	}
	r.b.events <- key('b')
	card := []string{"4;1", "4;9", "7;1", "7;9"}
	if got := sixelTiles(r.next(t)); !slices.Equal(got, card) {
		t.Fatalf("card background change sent tiles %v, want the card's tiles %v", got, card)
	}
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
}

func TestGraphicsChoice(t *testing.T) {
	sixel := terminal.Capabilities{Graphics: terminal.GraphicsSixel, CellPixels: image.Pt(10, 20)}
	for _, tc := range []struct {
		name   string
		caps   terminal.Capabilities
		env    string
		opts   []twi.RenderOption
		pixels bool
	}{
		{name: "reported", caps: sixel, pixels: true},
		{name: "no cell size", caps: terminal.Capabilities{Graphics: terminal.GraphicsSixel}},
		{name: "option none", caps: sixel, opts: []twi.RenderOption{twi.Graphics(terminal.GraphicsNone)}},
		{name: "option sixel", caps: terminal.Capabilities{CellPixels: image.Pt(10, 20)}, opts: []twi.RenderOption{twi.Graphics(terminal.GraphicsSixel)}, pixels: true},
		{name: "environment wins", caps: sixel, env: "sixel", opts: []twi.RenderOption{twi.Graphics(terminal.GraphicsNone)}, pixels: true},
		{name: "profile 256", caps: sixel, opts: []twi.RenderOption{twi.ColorProfile(color.ANSI256)}, pixels: true},
		{name: "profile 16", caps: sixel, opts: []twi.RenderOption{twi.ColorProfile(color.ANSI16)}},
		{name: "profile attributes", caps: sixel, opts: []twi.RenderOption{twi.ColorProfile(color.Attributes)}},
		{name: "profile none", caps: sixel, env: "sixel", opts: []twi.RenderOption{twi.ColorProfile(color.None)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TWIND_GRAPHICS", tc.env)
			b := newBackend(40, 15)
			b.caps = tc.caps
			r := surfaces(t, b, tc.opts...)
			if f := r.next(t); strings.Contains(f, "\x1bP") != tc.pixels {
				t.Errorf("first frame has Sixel %v, want %v", !tc.pixels, tc.pixels)
			}
			r.b.events <- key('b')
			if f := r.next(t); strings.Contains(f, "\x1bP") != tc.pixels {
				t.Errorf("frame after a background change has Sixel %v, want %v", !tc.pixels, tc.pixels)
			}
			if err := r.stop(t); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestResizeClearsSixelThenSendsAFullFrame(t *testing.T) {
	sixel := terminal.Capabilities{Graphics: terminal.GraphicsSixel, CellPixels: image.Pt(10, 20)}
	b := newBackend(40, 15)
	b.caps = sixel
	r := surfaces(t, b)
	r.next(t)
	r.b.events <- input.ResizeEvent{Width: 50, Height: 15}
	resized := r.next(t)
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
	wide := newBackend(50, 15)
	wide.caps = sixel
	fresh := surfaces(t, wide)
	first := fresh.next(t)
	if err := fresh.stop(t); err != nil {
		t.Fatal(err)
	}
	const clear = "\x1b[0m\x1b[2J"
	if !strings.HasPrefix(resized, clear) {
		t.Errorf("frame after a resize does not start with a reset and a full clear %q: %q", clear, resized[:min(len(resized), 40)])
	}
	if got, want := sixelTiles(resized), sixelTiles(first); !slices.Equal(got, want) {
		t.Errorf("resize sent tiles %v, want every tile of a fresh 50x15 frame %v", got, want)
	}
}

func TestResizeBurstCoalesces(t *testing.T) {
	r := start(counter.New)
	r.next(t)
	begin := time.Now()
	const burst = 40
	for i := range burst {
		r.b.events <- input.ResizeEvent{Width: 20 + i%5, Height: 3}
	}
	elapsed := time.Since(begin)
	time.Sleep(100 * time.Millisecond)
	frames := len(r.b.frames)
	if limit := 2 + int(elapsed/(time.Second/60)); frames > limit {
		t.Errorf("%d resize events in %v gave %d frames, want at most %d, one per 60 fps tick", burst, elapsed, frames, limit)
	}
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
}

func TestCellSizeChangeRedraws(t *testing.T) {
	b := newBackend(40, 15)
	b.caps = terminal.Capabilities{Graphics: terminal.GraphicsSixel, CellPixels: image.Pt(10, 20)}
	b.zoom = image.Pt(12, 24)
	r := surfaces(t, b)
	r.next(t)
	if f := r.next(t); len(sixelTiles(f)) == 0 {
		t.Fatalf("frame after the cell size changed sent no tile: %q", f)
	}
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
}
