package runtime_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
)

type timeline struct {
	rt  *twi.Runtime
	log []string
}

func (tl *timeline) note(entry string) func() {
	return func() {
		tl.log = append(tl.log, entry)
		tl.rt.Invalidate()
	}
}

func (tl *timeline) app(setup func()) drive.App {
	return func(rt *twi.Runtime) func() twi.Node {
		tl.rt = rt
		setup()
		return func() twi.Node { return twi.Text("fired " + strings.Join(tl.log, " ")) }
	}
}

func (tl *timeline) drive(t *testing.T, setup func()) *drive.Driver {
	t.Helper()
	d := drive.New(tl.app(setup), drive.Size(60, 2))
	t.Cleanup(func() {
		if err := d.Err(); err != nil {
			t.Error(err)
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	return d
}

func shown(d *drive.Driver) string {
	return strings.TrimSpace(strings.SplitN(d.Frame().Text(), "\n", 2)[0])
}

func TestTimerFiresAtItsDeadline(t *testing.T) {
	cases := []struct{ script, want string }{
		{"wait 699ms\nframe f\n", "fired"},
		{"wait 700ms\nframe f\n", "fired 700ms"},
		{"wait 699ms\nwait 1ms\nframe f\n", "fired 700ms"},
		{"wait 350ms\nwait 349ms\nframe f\n", "fired"},
	}
	for _, c := range cases {
		tl := &timeline{}
		out := t.TempDir()
		app := tl.app(func() { tl.rt.After(700*time.Millisecond, tl.note("700ms")) })
		if err := drive.RunScript(strings.NewReader("size 30x1\n"+c.script), app, out); err != nil {
			t.Fatal(err)
		}
		frame, err := os.ReadFile(filepath.Join(out, "f.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.TrimSpace(string(frame)); got != c.want {
			t.Errorf("script %q: frame %q, want %q", c.script, got, c.want)
		}
	}
}

func TestTimerStop(t *testing.T) {
	tl := &timeline{}
	d := tl.drive(t, func() {
		early := tl.rt.After(100*time.Millisecond, tl.note("early"))
		late := tl.rt.After(700*time.Millisecond, tl.note("late"))
		tl.rt.After(200*time.Millisecond, func() {
			early.Stop()
			late.Stop()
			late.Stop()
		})
		var twin *twi.Timer
		tl.rt.After(500*time.Millisecond, func() {
			twin.Stop()
			tl.note("first")()
		})
		twin = tl.rt.After(500*time.Millisecond, tl.note("twin"))
		tl.rt.After(600*time.Millisecond, tl.note("kept"))
		tl.rt.After(700*time.Millisecond, tl.note("sibling"))
	})
	d.Advance(time.Second)
	if got := shown(d); got != "fired early first kept sibling" {
		t.Errorf("got %q, want the stopped late and twin never to fire", got)
	}
}

func TestTimersFireInOrder(t *testing.T) {
	tl := &timeline{}
	d := tl.drive(t, func() {
		for i, tenths := range []int{3, 1, 2, 1, 3, 2, 1, 3, 2, 1} {
			tl.rt.After(time.Duration(tenths)*100*time.Millisecond, tl.note(string(rune('0'+i))))
		}
	})
	d.Advance(150 * time.Millisecond)
	if got := shown(d); got != "fired 1 3 6 9" {
		t.Errorf("at 150ms got %q", got)
	}
	d.Advance(time.Second)
	if got := shown(d); got != "fired 1 3 6 9 2 5 8 0 4 7" {
		t.Errorf("got %q, want deadline order and first in, first out on a tie", got)
	}
}

func TestTimerFromTimer(t *testing.T) {
	tl := &timeline{}
	d := tl.drive(t, func() {
		tl.rt.After(300*time.Millisecond, func() {
			tl.note("a")()
			tl.rt.After(400*time.Millisecond, tl.note("b"))
			tl.rt.After(0, tl.note("now"))
		})
	})
	for _, step := range []struct {
		wait time.Duration
		want string
	}{{300 * time.Millisecond, "fired a now"}, {399 * time.Millisecond, "fired a now"}, {time.Millisecond, "fired a now b"}} {
		d.Advance(step.wait)
		if got := shown(d); got != step.want {
			t.Errorf("after %v more: got %q, want %q", step.wait, got, step.want)
		}
	}
}

func TestTimerFromAnotherGoroutine(t *testing.T) {
	tl := &timeline{}
	d := tl.drive(t, func() {})
	d.Advance(time.Hour)
	tl.rt.After(200*time.Millisecond, tl.note("x"))
	d.Advance(199 * time.Millisecond)
	if got := shown(d); got != "fired" {
		t.Errorf("199ms after a timer set while the loop slept: %q", got)
	}
	d.Advance(time.Millisecond)
	if got := shown(d); got != "fired x" {
		t.Errorf("200ms after a timer set while the loop slept: %q", got)
	}
}

func TestTimerWakesOncePerTick(t *testing.T) {
	var fires atomic.Int64
	r := start(func(rt *twi.Runtime) func() twi.Node {
		var tick func()
		tick = func() {
			fires.Add(1)
			rt.After(time.Second, tick)
		}
		rt.After(time.Second, tick)
		return func() twi.Node { return twi.Text("tick") }
	})
	r.next(t)
	time.Sleep(500 * time.Millisecond)
	wakes, fired := r.clock.wakes.Load(), fires.Load()
	time.Sleep(3 * time.Second)
	wakes, fired = r.clock.wakes.Load()-wakes, fires.Load()-fired
	t.Logf("3s with a repeating 1s timer: %d fires, %d wakes", fired, wakes)
	if fired != 3 || wakes != fired {
		t.Errorf("%d fires and %d wakes in 3s, want 3 and 3", fired, wakes)
	}
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
}
