package budgets_test

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	runkonst "github.com/pehcastro/twind/internal/konst/runtime"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/motion"
)

const (
	tileCount  = 1000
	tilePitch  = 4
	tilesInRow = 30
	tileCols   = tilesInRow * tilePitch
	tileRows   = (tileCount + tilesInRow - 1) / tilesInRow
)

type tiles struct {
	rt      *twi.Runtime
	line    motion.Timeline
	now     time.Duration
	ids     []motion.ID
	flipped bool
	moving  atomic.Bool
}

func tileSpot(slot int) motion.Value {
	return motion.Bounds(float64(slot%tilesInRow*tilePitch), float64(slot/tilesInRow), 0, 0)
}

func (t *tiles) slot(i int) int {
	if t.flipped {
		return tileCount - 1 - i
	}
	return i
}

func (t *tiles) flip() {
	t.flipped = !t.flipped
	t.ids = t.ids[:0]
	for i := range tileCount {
		to := t.slot(i)
		t.ids = append(t.ids, t.line.Start(t.now, tileSpot(tileCount-1-to), tileSpot(to), motion.SpringDefault()))
	}
	t.moving.Store(true)
	t.rt.After(runkonst.MotionInterval, t.tick)
}

func (t *tiles) tick() {
	t.now += runkonst.MotionInterval
	_, running := t.line.Step(t.now)
	t.rt.Invalidate()
	if !running {
		t.ids = t.ids[:0]
		t.moving.Store(false)
		return
	}
	t.rt.After(runkonst.MotionInterval, t.tick)
}

func (t *tiles) app(rt *twi.Runtime) func() twi.Node {
	t.rt = rt
	labels := make([]string, tileCount)
	for i := range labels {
		labels[i] = fmt.Sprintf("%03d", i)
	}
	return func() twi.Node {
		opts := []twi.NodeOption{twi.Class("bg-zinc-950 text-zinc-100"), twi.OnKey(func(*twi.Event) { t.flip() })}
		for i := range tileCount {
			at := tileSpot(t.slot(i))
			if len(t.ids) > 0 {
				at = t.line.Value(t.ids[i])
			}
			x, y, _, _ := at.Bounds()
			opts = append(opts, twi.Element(twi.At(int(math.Round(x)), int(math.Round(y))), twi.Text(labels[i])))
		}
		return twi.Element(opts...)
	}
}

func BenchmarkMotion1000(b *testing.B) {
	sheet, err := Styles()
	if err != nil {
		b.Fatal(err)
	}
	t := &tiles{}
	d := drive.New(t.app, drive.Size(tileCols, tileRows), drive.With(twi.Styles(sheet)))
	now := clock(b)
	rest := strings.Fields(d.Frame().Text())
	var samples []time.Duration
	b.ResetTimer()
	for range b.N {
		d.Press("x")
		for t.moving.Load() {
			start := now()
			d.Advance(runkonst.MotionInterval)
			samples = append(samples, now()-start)
		}
	}
	b.StopTimer()
	got := strings.Fields(d.Frame().Text())
	want := slices.Clone(rest)
	if b.N%2 == 1 {
		slices.Reverse(want)
	}
	if len(rest) != tileCount || !slices.Equal(got, want) || len(samples) < b.N {
		b.Fatalf("after %d flips over %d frames the tiles read %d labels in another order than the %d they should", b.N, len(samples), len(got), len(want))
	}
	if err := d.Close(); err != nil {
		b.Fatal(err)
	}
	slices.Sort(samples)
	over := func(budget time.Duration) float64 {
		i, _ := slices.BinarySearch(samples, budget+1)
		return float64(len(samples)-i) / float64(b.N)
	}
	b.ReportMetric(float64(len(samples))/float64(b.N), "frames/op")
	b.ReportMetric(float64(samples[len(samples)/2])/float64(time.Millisecond), "frame-p50-ms")
	b.ReportMetric(float64(samples[len(samples)*95/100])/float64(time.Millisecond), "frame-p95-ms")
	b.ReportMetric(over(runkonst.FrameInterval), "over-60fps/op")
	b.ReportMetric(over(runkonst.MotionInterval), "over-30fps/op")
}
