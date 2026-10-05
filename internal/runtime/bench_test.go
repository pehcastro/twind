package runtime_test

import (
	"fmt"
	"image"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/runtime"
	"github.com/pehcastro/twind/internal/runtime/testdata/selection"
	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/input"
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

func BenchmarkDragToFrame(b *testing.B) {
	sheet, err := selection.Styles()
	if err != nil {
		b.Fatal(err)
	}
	card := strings.Fields(selection.Card)
	var rows []render.Node
	for r := range 3 {
		var cards []render.Node
		for c := range 4 {
			var texts []render.Node
			for t := range 3 {
				texts = append(texts, render.Node{Text: fmt.Sprintf("card %d %d paragraph %d: %s", r, c, t, selection.Wrapped)})
			}
			cards = append(cards, render.Node{Classes: card, Children: texts})
		}
		rows = append(rows, render.Node{Classes: []string{"flex", "flex-row", "gap-2"}, Children: cards})
	}
	tree := runtime.Tree{Root: render.Node{Classes: []string{"flex", "flex-col", "gap-1", "p-1", "h-full", "bg-black", "text-white"}, Children: rows}}
	be := &countingBackend{events: make(chan input.Event), written: make(chan int)}
	rt := runtime.New(runtime.Config{Clock: &steppingClock{}, Sheet: sheet, Profile: color.TrueColor})
	done := make(chan error, 1)
	go func() { done <- rt.Run(be, func() runtime.Tree { return tree }) }()
	<-be.written
	be.events <- input.MouseEvent{X: 4, Y: 2, Button: input.MouseLeft, Action: input.MousePress}
	var samples []time.Duration
	total, at, now := 0, 0, stopwatch(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		at++
		start := now()
		be.events <- input.MouseEvent{X: 8 + at%12, Y: 3 + at%5, Button: input.MouseLeft, Action: input.MouseMove}
		total += <-be.written
		samples = append(samples, now()-start)
	}
	b.StopTimer()
	rt.Quit()
	if err := <-done; err != nil {
		b.Fatal(err)
	}
	slices.Sort(samples)
	b.ReportMetric(float64(samples[len(samples)/2].Nanoseconds()), "p50-ns/move")
	b.ReportMetric(float64(samples[len(samples)*95/100].Nanoseconds()), "p95-ns/move")
	b.ReportMetric(float64(total)/float64(len(samples)), "bytes/move")
}

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
