package runtime_test

import (
	"image"
	"slices"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/runtime"
	"github.com/twind-dev/twind/twi/terminal"
)

type countingBackend struct {
	events  chan input.Event
	written chan int
	caps    terminal.Capabilities
}

func (b *countingBackend) Write(p []byte) (int, error) {
	b.written <- len(p)
	return len(p), nil
}

func (b *countingBackend) Capabilities() terminal.Capabilities  { return b.caps }
func (b *countingBackend) Events() <-chan input.Event           { return b.events }
func (b *countingBackend) Size() (width, height int, err error) { return 120, 40, nil }
func (b *countingBackend) Sync() bool                           { return true }
func (b *countingBackend) Exit() error                          { return nil }

type steppingClock struct{ calls atomic.Int64 }

func (c *steppingClock) Now() time.Time {
	return time.Unix(0, 0).Add(time.Duration(c.calls.Add(1)) * time.Second)
}

func (c *steppingClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

func BenchmarkWheelToFrame(b *testing.B) {
	const rows, view = 200, 20
	items := make([]render.Node, rows)
	for i := range items {
		items[i] = render.Node{Text: "row " + strconv.Itoa(i)}
	}
	tree := runtime.Tree{Root: render.Node{Classes: []string{"row"}, Children: []render.Node{
		{Classes: []string{"col", "h-20", "w-12", "shrink-0", "overflow-y-auto"}, Children: items},
		{Classes: []string{"col"}, Children: []render.Node{{Text: "fixed one"}, {Text: "fixed two"}}},
	}}}
	for _, arm := range []struct {
		name string
		caps terminal.Capabilities
	}{
		{"cells", terminal.Capabilities{}},
		{"sixel-margins", terminal.Capabilities{Graphics: terminal.GraphicsSixel, CellPixels: image.Pt(10, 20), Margins: true}},
	} {
		b.Run(arm.name, func(b *testing.B) {
			be := &countingBackend{events: make(chan input.Event), written: make(chan int), caps: arm.caps}
			rt := runtime.New(runtime.Config{Clock: &steppingClock{}, Sheet: scrollSheet(b), Profile: color.TrueColor})
			done := make(chan error, 1)
			go func() { done <- rt.Run(be, func() runtime.Tree { return tree }) }()
			<-be.written
			var samples []time.Duration
			total, down, at := 0, true, 0
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				down = down && at+3 <= rows-view || !down && at < 3
				button := input.MouseWheelUp
				if down {
					button, at = input.MouseWheelDown, at+3
				} else {
					at -= 3
				}
				start := time.Now()
				be.events <- input.MouseEvent{X: 2, Y: 5, Button: button, Action: input.MouseScroll}
				total += <-be.written
				samples = append(samples, time.Since(start))
			}
			b.StopTimer()
			rt.Quit()
			if err := <-done; err != nil {
				b.Fatal(err)
			}
			slices.Sort(samples)
			b.ReportMetric(float64(samples[len(samples)*95/100].Nanoseconds()), "p95-ns/notch")
			b.ReportMetric(float64(total)/float64(len(samples)), "bytes/notch")
		})
	}
}
