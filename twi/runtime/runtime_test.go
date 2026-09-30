package runtime_test

import (
	"errors"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/runtime"
	"github.com/twind-dev/twind/twi/testdata/counter"
)

type backend struct {
	events        chan input.Event
	width, height int
	frames        chan string
	exits         atomic.Int32
}

func (b *backend) Write(p []byte) (int, error) {
	b.frames <- string(p)
	return len(p), nil
}

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
	r := run{
		b:     &backend{events: make(chan input.Event), width: 20, height: 3, frames: make(chan string, 1<<16)},
		clock: &clock{},
		done:  make(chan error, 1),
	}
	r.rt = twi.New(twi.Backend(r.b, r.clock))
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
