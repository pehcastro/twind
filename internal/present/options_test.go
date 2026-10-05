package present

import (
	"bytes"
	"image"
	"runtime"
	"runtime/metrics"
	"testing"

	"github.com/pehcastro/twind/internal/present/demo"
	"github.com/pehcastro/twind/twi/buffer"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/terminal"
	"github.com/pehcastro/twind/twi/text"
)

func TestWidthsReachThePainter(t *testing.T) {
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for _, c := range []struct {
		widths       text.Widths
		second, want string
	}{
		{text.Widths{}, "", "the flag's continuation"},
		{text.Widths{text.Flag: 1}, "o", "o"},
	} {
		s, _ := screen(terminal.GraphicsNone)
		s.Widths = c.widths
		frame(t, s, flatPage(white, "🇧🇷ok"))
		if got := s.text.At(1, 0); got.Grapheme != c.second || c.second == "" && got.Width != buffer.Continuation {
			t.Errorf("widths %v: cell 1 is %+v, want %s", c.widths, got, c.want)
		}
	}
}

func TestSameCardInsideAClipRastersOnce(t *testing.T) {
	rastered := func(row string, cards int) (*Screen, int) {
		s, _ := screen(terminal.GraphicsSixel)
		frame(t, s, tree(t, demo.Cards(row, cards)))
		return s, s.rastered
	}
	_, one := rastered("", 1)
	s, two := rastered("", 2)
	if two != one {
		t.Errorf("two equal cards inside a clip rastered %d boxes, one card %d: the clip that cuts neither is part of the look", two, one)
	}
	if _, three := rastered("w-40", 2); three != one+1 {
		t.Errorf("a card cut by its clip beside a whole one rastered %d boxes, want %d: the cut card is another look", three, one+1)
	}
	img, left, right := s.image(), image.Pt(4*wt.X, 4*wt.Y), image.Pt(30*wt.X, 4*wt.Y)
	for y := range 20 * wt.Y {
		for x := range 24 * wt.X {
			if a, b := img.RGBAAt(left.X+x, left.Y+y), img.RGBAAt(right.X+x, right.Y+y); a != b {
				t.Fatalf("pixel %d,%d of the two cards: %v and %v, want equal", x, y, a, b)
			}
		}
	}
}

func TestOneWorkerStartsNoGoroutine(t *testing.T) {
	if runtime.GOMAXPROCS(0) < 2 {
		t.Skip("one processor: the parallel path starts no goroutine either")
	}
	root := tree(t, demo.Dialog())
	created := func() uint64 {
		sample := []metrics.Sample{{Name: "/sched/goroutines-created:goroutines"}}
		metrics.Read(sample)
		return sample[0].Value.Uint64()
	}
	var got [2][]byte
	for i, workers := range []int{0, 1} {
		s, out := screen(terminal.GraphicsSixel)
		s.Workers = workers
		runtime.GC()
		before := created()
		frame(t, s, root)
		started := created() - before
		got[i] = out.last()
		switch {
		case workers == 1 && started != 0:
			t.Errorf("one worker: the frame started %d goroutines, want 0", started)
		case workers == 0 && started == 0:
			t.Errorf("default workers: the frame started no goroutine, so this probe cannot see one")
		}
	}
	if !bytes.Equal(got[0], got[1]) {
		t.Errorf("one worker wrote %d bytes, the parallel path %d, want the same bytes", len(got[1]), len(got[0]))
	}
}
