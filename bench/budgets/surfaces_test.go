package budgets_test

import (
	"bytes"
	"image"
	"testing"
	"time"

	"github.com/twind-dev/twind/bench/scenarios/surfaces"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/style"
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
		start := bytes.Index(p, []byte("\x1bP"))
		if kitty := bytes.Index(p, []byte("\x1b_G")); kitty >= 0 && (start < 0 || kitty < start) {
			start = kitty
		}
		if start < 0 {
			return
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

func start(sheet style.Sheet, caps terminal.Capabilities) (*surfaceBackend, *twi.Runtime, chan error) {
	be := &surfaceBackend{events: make(chan input.Event), written: make(chan []byte), caps: caps}
	rt := twi.New(twi.Backend(be, &steppingClock{}), twi.Styles(sheet), twi.Theme(surfaces.Themes()[0]), twi.ColorProfile(color.TrueColor))
	done := make(chan error, 1)
	go func() { done <- rt.Run(surfaces.App(rt)) }()
	return be, rt, done
}

func stop(b *testing.B, rt *twi.Runtime, done chan error) {
	rt.Quit()
	if err := <-done; err != nil {
		b.Fatal(err)
	}
}

func key(r rune) input.Event { return input.KeyEvent{Key: input.KeyRune, Rune: r} }

func BenchmarkSurface(b *testing.B) {
	sheet, err := surfaces.Styles()
	if err != nil {
		b.Fatal(err)
	}
	arms := []struct {
		name      string
		caps      terminal.Capabilities
		firstOnly bool
	}{
		{"sixel", terminal.Capabilities{Sync: true, Graphics: terminal.GraphicsSixel, CellPixels: image.Pt(10, 20)}, false},
		{"kitty", terminal.Capabilities{Sync: true, Graphics: terminal.GraphicsKitty, CellPixels: image.Pt(10, 20)}, true},
		{"none", terminal.Capabilities{Sync: true}, false},
	}
	steps := []struct {
		name        string
		event, back input.Event
	}{
		{"text", key(surfaces.Count), nil},
		{"hover", key(surfaces.Hover), nil},
		{"theme", key(surfaces.Theme), nil},
		{"resize", input.ResizeEvent{Width: surfaces.Wide, Height: surfaces.Rows}, input.ResizeEvent{Width: surfaces.Columns, Height: surfaces.Rows}},
	}
	for _, arm := range arms {
		b.Run(arm.name+"/first", func(b *testing.B) {
			now := clock(b)
			samples := make([]time.Duration, b.N)
			var out frames
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				begin := now()
				be, rt, done := start(sheet, arm.caps)
				p := <-be.written
				samples[i] = now() - begin
				out.add(p)
				stop(b, rt, done)
			}
			b.StopTimer()
			percentiles(b, samples)
			out.report(b)
		})
		for _, step := range steps {
			if arm.firstOnly {
				break
			}
			b.Run(arm.name+"/"+step.name, func(b *testing.B) {
				now := clock(b)
				be, rt, done := start(sheet, arm.caps)
				var first, out frames
				first.add(<-be.written)
				samples := make([]time.Duration, b.N)
				b.ReportAllocs()
				b.ResetTimer()
				for i := range b.N {
					begin := now()
					be.events <- step.event
					p := <-be.written
					samples[i] = now() - begin
					out.add(p)
					if step.back != nil {
						be.events <- step.back
						<-be.written
					}
				}
				b.StopTimer()
				stop(b, rt, done)
				percentiles(b, samples)
				out.report(b)
				b.ReportMetric(float64(first.bytes), "first-bytes/frame")
			})
		}
	}
}
