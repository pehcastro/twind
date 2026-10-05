package runtime_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/runtime"
	"github.com/pehcastro/twind/twi/terminal"
)

type serialBackend struct {
	*backend
	writes int
}

func (s *serialBackend) Write(p []byte) (int, error) {
	s.writes++
	return s.backend.Write(p)
}

func TestCopyBeforeRunRidesTheFirstFrame(t *testing.T) {
	b := newBackend(10, 1)
	rt := runtime.New(runtime.Config{Clock: &clock{}})
	if err := rt.Copy("early"); err != nil {
		t.Fatal(err)
	}
	r := run{b: b, done: make(chan error, 1)}
	go func() {
		r.done <- rt.Run(b, func() runtime.Tree { return runtime.Tree{Root: render.Node{Text: "copy"}} })
	}()
	if f := r.next(t); !strings.Contains(f, string(terminal.Clipboard("early"))) || !strings.Contains(f, "copy") {
		t.Errorf("first write %q, want the clipboard bytes and the frame in one write", f)
	}
	rt.Quit()
	if err := r.result(t); err != nil {
		t.Fatal(err)
	}
}

func TestCopyFromAnotherGoroutineWritesOnTheLoop(t *testing.T) {
	const copies = 50
	b := &serialBackend{backend: newBackend(10, 1)}
	tick := 0
	rt := twi.New(twi.Backend(b, &clock{}), twi.ColorProfile(color.None))
	done := make(chan error, 1)
	go func() { done <- rt.Run(func() twi.Node { return twi.Text(fmt.Sprint("tick ", tick)) }) }()
	var wg sync.WaitGroup
	wg.Go(func() {
		for range copies {
			if err := rt.Copy("x"); err != nil {
				t.Error(err)
			}
			rt.Dispatch(func() { tick++ })
		}
	})
	wg.Wait()
	seen, clip := 0, string(terminal.Clipboard("x"))
	for deadline := time.After(2 * time.Second); seen < copies; {
		select {
		case f := <-b.frames:
			seen += strings.Count(f, clip)
		case <-deadline:
			t.Fatalf("%d of %d copies reached the terminal", seen, copies)
		}
	}
	rt.Quit()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestFocusFromAnotherGoroutine(t *testing.T) {
	r := start(func(rt *twi.Runtime) func() twi.Node {
		return func() twi.Node {
			return twi.Element(
				twi.Element(twi.Key("a"), twi.Focusable(), twi.Text("a")),
				twi.Element(twi.Key("b"), twi.Focusable(), twi.Text("b")),
			)
		}
	})
	r.next(t)
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				r.rt.Invalidate()
			}
		}
	})
	for range 20 {
		r.rt.HideFocusRings()
		if !r.rt.Focus("b") || !r.rt.Focus("a") {
			t.Error("Focus from another goroutine refused a focusable key")
		}
		if r.rt.Focus("missing") {
			t.Error("Focus from another goroutine took a missing key")
		}
	}
	close(stop)
	wg.Wait()
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
	if r.rt.Focus("a") {
		t.Error("Focus after Run took a key")
	}
}

func TestScrollIntoViewFromAnotherGoroutineWakesTheLoop(t *testing.T) {
	r := launch(newBackend(20, 8), func(rt *twi.Runtime) func() twi.Node {
		rt.HideFocusRings()
		return func() twi.Node {
			opts := []twi.NodeOption{twi.Class("col h-5 w-12 shrink-0 overflow-y-auto")}
			for i := range listRows {
				opts = append(opts, twi.Element(twi.Key(fmt.Sprint("row-", i)), twi.Text(fmt.Sprint("row ", i))))
			}
			return twi.Element(twi.Class("col"), twi.Element(opts...))
		}
	}, twi.Styles(scrollSheet(t)))
	g := blank(r.b)
	g.apply(t, r.next(t))
	r.quiet(t)
	var wg sync.WaitGroup
	wg.Go(func() { r.rt.ScrollIntoView("row-12") })
	wg.Wait()
	g.apply(t, r.next(t))
	g.settle(t, r.b)
	if got := g.first(); got != 12 {
		t.Errorf("the frame shows row %d first, want row 12:\n%s", got, g.text())
	}
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
}
