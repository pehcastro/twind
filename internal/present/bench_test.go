package present

import (
	"fmt"
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

func BenchmarkScrollOneRow(b *testing.B) {
	for _, margins := range []bool{true, false} {
		b.Run(fmt.Sprintf("margins=%v", margins), func(b *testing.B) {
			offsets := make([]int, 0, 2*(listRows-rows))
			for o := range listRows - rows {
				offsets = append(offsets, o)
			}
			for o := listRows - rows; o > 0; o-- {
				offsets = append(offsets, o)
			}
			trees := scrolled(b, offsets...)
			s, out := screen(terminal.GraphicsSixel)
			s.Margins = margins
			frame(b, s, trees[0])
			const framesPerSample = 8
			var samples []time.Duration
			written, images, tiles := 0, 0, 0
			start := time.Now()
			for i := 1; b.Loop(); i++ {
				frame(b, s, trees[i%len(trees)])
				if i%framesPerSample == 0 {
					samples = append(samples, time.Since(start)/framesPerSample)
					start = time.Now()
				}
				written, images = written+len(out.last()), images+s.imageBytes
				for _, sent := range s.send {
					if sent {
						tiles++
					}
				}
			}
			if len(samples) > 0 {
				p95(b, samples)
			}
			b.ReportMetric(float64(written)/float64(b.N), "bytes/frame")
			b.ReportMetric(float64(images)/float64(b.N), "image-bytes/frame")
			b.ReportMetric(float64(tiles)/float64(b.N), "tiles/frame")
		})
	}
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
