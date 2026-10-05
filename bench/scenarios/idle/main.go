package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/pehcastro/twind/bench/scenarios/surfaces"
	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/testdata/counter"
)

type backend struct {
	events        chan input.Event
	bytes         atomic.Int64
	drawn         chan struct{}
	width, height int
	caps          terminal.Capabilities
}

func (b *backend) Write(p []byte) (int, error) {
	if b.bytes.Add(int64(len(p))) == int64(len(p)) {
		close(b.drawn)
	}
	return len(p), nil
}

func (b *backend) Events() <-chan input.Event           { return b.events }
func (b *backend) Size() (width, height int, err error) { return b.width, b.height, nil }
func (b *backend) Sync() bool                           { return true }
func (b *backend) Exit() error                          { return nil }
func (b *backend) Capabilities() terminal.Capabilities  { return b.caps }

type clock struct{ wakes atomic.Int64 }

func (c *clock) Now() time.Time {
	c.wakes.Add(1)
	return time.Now()
}

func (c *clock) After(d time.Duration) <-chan time.Time { return time.After(d) }

func main() {
	app := flag.String("app", "counter", "counter, 80x24 without graphics, or surfaces, 120x40 with Sixel and a 10x20 cell")
	window := flag.Duration("window", 10*time.Second, "idle time measured after the first frame")
	period := flag.Duration("timer", 0, "the period of one timer that sets itself again each time it fires, 0 for no timer")
	flag.Parse()
	b := &backend{events: make(chan input.Event), drawn: make(chan struct{}), width: 80, height: 24}
	c := &clock{}
	opts := []twi.RenderOption{twi.Backend(b, c), twi.ColorProfile(color.TrueColor)}
	view := counter.New
	switch *app {
	case "counter":
	case "surfaces":
		sheet, err := surfaces.Styles()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		b.width, b.height = surfaces.Columns, surfaces.Rows
		b.caps = terminal.Capabilities{Sync: true, Graphics: terminal.GraphicsSixel, CellPixels: image.Pt(10, 20)}
		opts = append(opts, twi.Styles(sheet), twi.Theme(surfaces.Themes()[0]))
		view = surfaces.App
	default:
		fmt.Fprintf(os.Stderr, "-app %q: want counter or surfaces\n", *app)
		os.Exit(2)
	}
	rt := twi.New(opts...)
	var ticks atomic.Int64
	if *period > 0 {
		origin := time.Now()
		var arm func()
		arm = func() {
			rt.After(time.Until(origin.Add(time.Duration(ticks.Load()+1)**period)), func() {
				ticks.Add(1)
				arm()
			})
		}
		rt.Dispatch(arm)
	}
	done := make(chan error, 1)
	go func() { done <- rt.Run(view(rt)) }()
	<-b.drawn
	settled := make(chan struct{})
	rt.Dispatch(func() { close(settled) })
	<-settled

	bytes, wakes, fired, cpu, spent, start := b.bytes.Load(), c.wakes.Load(), ticks.Load(), cpuTime(), cycles(), time.Now()
	time.Sleep(*window)
	cpu, spent, wall := cpuTime()-cpu, cycles()-spent, time.Since(start)
	bytes, wakes, fired = b.bytes.Load()-bytes, c.wakes.Load()-wakes, ticks.Load()-fired
	resident, private := memory()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	rt.Quit()
	if err := <-done; err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(cpu.Nanoseconds(), wall.Nanoseconds(), wakes, bytes, resident, private, stats.HeapInuse, fired, spent)
}
