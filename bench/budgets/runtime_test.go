package budgets_test

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/runtime/testdata/pill"
)

type backend struct {
	events  chan input.Event
	written chan int
	sync    bool
}

func (be *backend) Write(p []byte) (int, error) {
	be.written <- len(p)
	return len(p), nil
}

func (be *backend) Events() <-chan input.Event           { return be.events }
func (be *backend) Size() (width, height int, err error) { return columns, rows, nil }
func (be *backend) Sync() bool                           { return be.sync }
func (be *backend) Exit() error                          { return nil }

type steppingClock struct{ calls atomic.Int64 }

func (c *steppingClock) Now() time.Time {
	return time.Unix(0, 0).Add(time.Duration(c.calls.Add(1)) * noThrottle)
}

func (c *steppingClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

func BenchmarkKeyToFrame(b *testing.B) {
	sheet, err := Styles()
	if err != nil {
		b.Fatal(err)
	}
	now := clock(b)

	b.Run("drive", func(b *testing.B) {
		d := drive.New(keys, drive.Size(columns, rows), drive.Styles(sheet))
		samples := make([]time.Duration, 0, b.N*keysPerOp)
		b.ResetTimer()
		for range b.N * keysPerOp {
			start := now()
			d.Press("a")
			samples = append(samples, now()-start)
		}
		b.StopTimer()
		if want := "keys " + strconv.Itoa(len(samples)) + "\n"; !strings.HasPrefix(d.Frame().Text(), want) {
			b.Fatalf("last frame does not start with %q:\n%s", want, d.Frame().Text())
		}
		if err := d.Err(); err != nil {
			b.Fatal(err)
		}
		if err := d.Close(); err != nil {
			b.Fatal(err)
		}
		percentiles(b, "", samples)
	})

	for _, sync := range []bool{true, false} {
		b.Run("sync="+strconv.FormatBool(sync), func(b *testing.B) {
			be := &backend{events: make(chan input.Event), written: make(chan int), sync: sync}
			rt := twi.New(twi.Backend(be, &steppingClock{}), twi.Styles(sheet), twi.ColorProfile(color.TrueColor))
			done := make(chan error, 1)
			go func() { done <- rt.Run(keys(rt)) }()
			first := <-be.written

			samples := make([]time.Duration, 0, b.N*keysPerOp)
			total := 0
			b.ResetTimer()
			for range b.N * keysPerOp {
				start := now()
				be.events <- input.KeyEvent{Key: input.KeyRune, Rune: 'a'}
				total += <-be.written
				samples = append(samples, now()-start)
			}
			b.StopTimer()
			rt.Quit()
			if err := <-done; err != nil {
				b.Fatal(err)
			}
			percentiles(b, "", samples)
			b.ReportMetric(float64(first), "first-bytes/frame")
			b.ReportMetric(float64(total)/float64(len(samples)), "bytes/frame")
		})
	}
}

func BenchmarkFocusToFrame(b *testing.B) {
	sheet, err := pill.Styles()
	if err != nil {
		b.Fatal(err)
	}
	now := clock(b)
	be := &backend{events: make(chan input.Event), written: make(chan int), sync: true}
	rt := twi.New(twi.Backend(be, &steppingClock{}), twi.Styles(sheet), twi.ColorProfile(color.TrueColor))
	done := make(chan error, 1)
	go func() { done <- rt.Run(pill.Many(focusPills)(rt)) }()
	<-be.written
	samples := make([]time.Duration, 0, b.N)
	total := 0
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		start := now()
		be.events <- input.KeyEvent{Key: input.KeyTab}
		total += <-be.written
		samples = append(samples, now()-start)
	}
	b.StopTimer()
	rt.Quit()
	if err := <-done; err != nil {
		b.Fatal(err)
	}
	percentiles(b, "", samples)
	b.ReportMetric(float64(total)/float64(len(samples)), "bytes/frame")
}

func BenchmarkIdle(b *testing.B) {
	b.Run("none", func(b *testing.B) { idle(b) })
	b.Run("timer", func(b *testing.B) { idle(b, "-timer", "1s") })
}

func BenchmarkAppIdle(b *testing.B) { idle(b, "-app", "surfaces", "-window", "60s") }

func idle(b *testing.B, args ...string) {
	exe := filepath.Join(b.TempDir(), "idle.exe")
	if out, err := exec.Command("go", "build", "-o", exe, "github.com/twind-dev/twind/bench/scenarios/idle").CombinedOutput(); err != nil {
		b.Fatalf("%v\n%s", err, out)
	}
	var cpu, wall, wakes, bytes, resident, private, heap, fired, spent float64
	b.ResetTimer()
	for range b.N {
		out, err := exec.Command(exe, args...).Output()
		if err != nil {
			b.Fatal(err)
		}
		var c, w, k, n, r, p, h, f, s float64
		if _, err := fmt.Sscan(string(out), &c, &w, &k, &n, &r, &p, &h, &f, &s); err != nil {
			b.Fatalf("idle printed %q: %v", out, err)
		}
		cpu, wall, wakes, bytes, fired, spent = cpu+c, wall+w, wakes+k, bytes+n, fired+f, spent+s
		resident, private, heap = max(resident, r), max(private, p), max(heap, h)
	}
	b.StopTimer()
	b.ReportMetric(100*cpu/wall, "cpu-%")
	b.ReportMetric(spent/float64(b.N), "cpu-cycles")
	b.ReportMetric(wakes/float64(b.N), "wakeups")
	b.ReportMetric(wakes/(wall/float64(time.Second)), "wakeups/s")
	b.ReportMetric(fired/float64(b.N), "timer-fires")
	if fired > 0 {
		b.ReportMetric(wakes/fired, "wakeups/fire")
	}
	b.ReportMetric(bytes/float64(b.N), "idle-bytes")
	b.ReportMetric(resident/mebi, "rss-MB")
	b.ReportMetric(private/mebi, "private-MB")
	b.ReportMetric(heap/mebi, "heap-MB")
}
