package present

import (
	"slices"
	"testing"
	"time"

	"github.com/twind-dev/twind/internal/present/demo"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/terminal"
)

func p95(b *testing.B, samples []time.Duration) {
	slices.Sort(samples)
	b.ReportMetric(float64(samples[len(samples)*95/100].Nanoseconds()), "p95-ns/frame")
}

func BenchmarkFirstFrameDialog(b *testing.B) {
	root := tree(b, demo.Dialog())
	var samples []time.Duration
	bytes := 0
	for b.Loop() {
		s, out := screen(terminal.GraphicsSixel)
		start := time.Now()
		frame(b, s, root)
		samples = append(samples, time.Since(start))
		bytes = len(out.last())
	}
	p95(b, samples)
	b.ReportMetric(float64(bytes), "bytes/frame")
}

func BenchmarkRowHover(b *testing.B) {
	const framesPerSample = 8
	trees := [2]scene.Node{tree(b, demo.List(1)), tree(b, demo.List(2))}
	s, out := screen(terminal.GraphicsSixel)
	frame(b, s, trees[0])
	frame(b, s, trees[1])
	var samples []time.Duration
	written := 0
	start := time.Now()
	for i := 0; b.Loop(); i++ {
		frame(b, s, trees[i%2])
		written += len(out.last())
		if i%framesPerSample == framesPerSample-1 {
			samples = append(samples, time.Since(start)/framesPerSample)
			start = time.Now()
		}
	}
	if len(samples) > 0 {
		p95(b, samples)
	}
	b.ReportMetric(float64(written)/float64(b.N), "bytes/frame")
}
