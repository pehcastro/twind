package budgets_test

import (
	"runtime"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/terminal"
)

const (
	columns    = 80
	rows       = 24
	mebi       = 1 << 20
	keyNodes   = 1000
	keysPerOp  = 1000
	focusPills = 40
	noThrottle = time.Second
)

type counter struct{ n int }

func (c *counter) Write(p []byte) (int, error) {
	c.n += len(p)
	return len(p), nil
}

func measure(b *testing.B, frame func()) {
	now := clock(b)
	samples := make([]time.Duration, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		start := now()
		frame()
		samples[i] = now() - start
	}
	b.StopTimer()
	percentiles(b, "", samples)
}

func percentiles(b *testing.B, prefix string, samples []time.Duration) {
	slices.Sort(samples)
	b.ReportMetric(float64(samples[len(samples)/2]), prefix+"p50-ns")
	b.ReportMetric(float64(samples[len(samples)*95/100]), prefix+"p95-ns")
}

func render(b *testing.B, build func() twi.Node) {
	var out counter
	frame := func() {
		sheet, err := Styles()
		if err == nil {
			err = twi.Render(&out, build(), twi.Styles(sheet), twi.Width(columns), twi.ColorProfile(color.TrueColor))
		}
		if err != nil {
			b.Fatal(err)
		}
	}

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	frame()
	runtime.ReadMemStats(&after)
	bytes := out.n

	measure(b, frame)
	b.ReportMetric(float64(int64(after.HeapInuse)-int64(before.HeapInuse))/mebi, "heap-MB")
	b.ReportMetric(float64(bytes), "bytes/frame")
}

func BenchmarkColdStart343(b *testing.B) { render(b, hello) }

func BenchmarkRender(b *testing.B) {
	for _, n := range []int{1000, 10000} {
		b.Run("nodes="+strconv.Itoa(n), func(b *testing.B) {
			render(b, func() twi.Node { return tree(n) })
		})
	}
}

func BenchmarkOneCell(b *testing.B) {
	fill := buffer.Cell{
		Grapheme: " ",
		Fg:       color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 244, G: 244, B: 245, A: 255}},
		Bg:       color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 9, G: 9, B: 11, A: 255}},
	}
	for _, sync := range []bool{true, false} {
		b.Run("sync="+strconv.FormatBool(sync), func(b *testing.B) {
			frames := [2]*buffer.Buffer{buffer.New(columns, rows), buffer.New(columns, rows)}
			for _, f := range frames {
				f.Fill(buffer.Rect{W: columns, H: rows}, fill)
			}
			changed := fill
			changed.Grapheme = "x"
			frames[1].Set(columns/2, rows/2, changed)

			var out counter
			w := terminal.Writer{Out: &out, Profile: color.TrueColor, Sync: sync}
			if err := w.Diff(frames[0], frames[1]); err != nil {
				b.Fatal(err)
			}
			first := out.n
			out.n = 0

			i := 1
			measure(b, func() {
				if err := w.Diff(frames[i%2], frames[(i+1)%2]); err != nil {
					b.Fatal(err)
				}
				i++
			})
			b.ReportMetric(float64(first), "first-bytes/frame")
			b.ReportMetric(float64(out.n)/float64(b.N), "bytes/frame")
		})
	}
}
