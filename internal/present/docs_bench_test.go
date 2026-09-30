package present_test

import (
	"image"
	"slices"
	"testing"
	"time"

	docsapp "github.com/twind-dev/twind/apps/documentation"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/terminal"
)

type docsBackend struct {
	events  chan input.Event
	written chan []byte
}

func (be *docsBackend) Write(p []byte) (int, error) {
	be.written <- p
	return len(p), nil
}

func (be *docsBackend) Events() <-chan input.Event           { return be.events }
func (be *docsBackend) Sync() bool                           { return true }
func (be *docsBackend) Exit() error                          { return nil }
func (be *docsBackend) Size() (width, height int, err error) { return 120, 36, nil }
func (be *docsBackend) Capabilities() terminal.Capabilities {
	return terminal.Capabilities{Sync: true, Graphics: terminal.GraphicsSixel, CellPixels: image.Pt(10, 20)}
}

type docsClock struct{ calls int64 }

func (c *docsClock) Now() time.Time {
	c.calls++
	return time.Unix(0, 0).Add(time.Duration(c.calls) * time.Second)
}

func (c *docsClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

func BenchmarkFirstFrameDocs(b *testing.B) {
	sheet, err := docsapp.Styles()
	if err != nil {
		b.Fatal(err)
	}
	var samples []time.Duration
	bytes := 0
	for b.Loop() {
		be := &docsBackend{events: make(chan input.Event), written: make(chan []byte)}
		start := time.Now()
		rt := twi.New(twi.Backend(be, &docsClock{}), twi.Styles(sheet), twi.ColorProfile(color.TrueColor))
		done := make(chan error, 1)
		go func() { done <- rt.Run(docsapp.App(rt)) }()
		bytes = len(<-be.written)
		samples = append(samples, time.Since(start))
		b.StopTimer()
		rt.Quit()
		for waiting := true; waiting; {
			select {
			case <-be.written:
			case err := <-done:
				if err != nil {
					b.Fatal(err)
				}
				waiting = false
			}
		}
		b.StartTimer()
	}
	slices.Sort(samples)
	b.ReportMetric(float64(samples[len(samples)*95/100].Nanoseconds()), "p95-ns/frame")
	b.ReportMetric(float64(bytes), "bytes/frame")
}
