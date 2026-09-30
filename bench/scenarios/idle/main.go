package main

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/testdata/counter"
)

const (
	window  = 10 * time.Second
	columns = 80
	rows    = 24
)

type backend struct {
	events chan input.Event
	bytes  atomic.Int64
	drawn  chan struct{}
}

func (b *backend) Write(p []byte) (int, error) {
	if b.bytes.Add(int64(len(p))) == int64(len(p)) {
		close(b.drawn)
	}
	return len(p), nil
}

func (b *backend) Events() <-chan input.Event           { return b.events }
func (b *backend) Size() (width, height int, err error) { return columns, rows, nil }
func (b *backend) Sync() bool                           { return true }
func (b *backend) Exit() error                          { return nil }

type clock struct{ wakes atomic.Int64 }

func (c *clock) Now() time.Time {
	c.wakes.Add(1)
	return time.Now()
}

func (c *clock) After(d time.Duration) <-chan time.Time { return time.After(d) }

func main() {
	b := &backend{events: make(chan input.Event), drawn: make(chan struct{})}
	c := &clock{}
	rt := twi.New(twi.Backend(b, c), twi.ColorProfile(color.TrueColor))
	done := make(chan error, 1)
	go func() { done <- rt.Run(counter.New(rt)) }()
	<-b.drawn

	bytes, wakes, cpu, start := b.bytes.Load(), c.wakes.Load(), cpuTime(), time.Now()
	time.Sleep(window)
	cpu, wall := cpuTime()-cpu, time.Since(start)
	bytes, wakes = b.bytes.Load()-bytes, c.wakes.Load()-wakes

	rt.Quit()
	if err := <-done; err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(cpu.Nanoseconds(), wall.Nanoseconds(), wakes, bytes)
}
