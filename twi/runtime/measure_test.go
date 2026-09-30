package runtime_test

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	rkonst "github.com/twind-dev/twind/internal/konst/runtime"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

type tipApp struct {
	boxX   int
	hidden bool
}

func (a *tipApp) build(rt *twi.Runtime) func() twi.Node {
	box := twi.NewRef(rt)
	return func() twi.Node {
		at, view := box.Bounds(), rt.Viewport()
		x := at.Max.X + 1
		if x+len("tip!") > view.Max.X {
			x = at.Min.X - 1 - len("tip!")
		}
		keys := twi.OnKey(func(k input.KeyEvent) {
			switch k.Rune {
			case 'm':
				a.boxX = 35
			case 'h':
				a.hidden = true
			}
			rt.Invalidate()
		})
		if a.hidden {
			return twi.Element(keys, twi.Text(fmt.Sprint(at)))
		}
		return twi.Element(keys, twi.Text(fmt.Sprint(at)), twi.Element(
			twi.Element(twi.At(x, at.Min.Y), twi.Text("tip!")),
			twi.Element(twi.Measure(box), twi.At(a.boxX, 5), twi.Text("[box]")),
		))
	}
}

func (a *tipApp) start(t *testing.T) (run, grid) {
	t.Helper()
	r := launch(newBackend(40, 12), a.build)
	t.Cleanup(func() {
		if err := r.stop(t); err != nil {
			t.Error(err)
		}
	})
	return r, blank(r.b)
}

func (g grid) row(y int) string { return strings.TrimRight(string(g[y]), " ") }

func TestMeasureReportsTheBoxAfterTheFirstFrame(t *testing.T) {
	a := &tipApp{boxX: 35}
	r, g := a.start(t)
	g.apply(t, r.next(t))
	if got, want := g.row(0), "(35,5)-(40,6)"; got != want {
		t.Fatalf("the first frame reads the ref as %q, want %q:\n%s", got, want, g.text())
	}
}

func TestMeasureNeverDrawsTheUnplacedFrame(t *testing.T) {
	a := &tipApp{boxX: 35}
	r, g := a.start(t)
	g.apply(t, r.next(t))
	if got, want := g.row(5), strings.Repeat(" ", 30)+"tip! [box]"; got != want {
		t.Fatalf("the first frame drew row 5 as %q, want the tip left of the box %q:\n%s", got, want, g.text())
	}
	if strings.Contains(g.row(1), "tip!") || strings.Contains(g.row(0), "tip!") {
		t.Fatalf("the first frame drew the tip where an unmeasured box puts it:\n%s", g.text())
	}
	r.quiet(t)
}

func TestMeasureFollowsABoxThatMovesInTheSameFrame(t *testing.T) {
	a := &tipApp{boxX: 10}
	r, g := a.start(t)
	g.apply(t, r.next(t))
	if got, want := g.row(5), strings.Repeat(" ", 10)+"[box] tip!"; got != want {
		t.Fatalf("with room on the right row 5 is %q, want %q:\n%s", got, want, g.text())
	}
	r.b.events <- key('m')
	g.apply(t, r.next(t))
	if got, want := g.row(5), strings.Repeat(" ", 30)+"tip! [box]"; got != want {
		t.Fatalf("the frame the box moved in drew row 5 as %q, want %q:\n%s", got, want, g.text())
	}
	if got, want := g.row(0), "(35,5)-(40,6)"; got != want {
		t.Fatalf("the frame the box moved in reads the ref as %q, want %q", got, want)
	}
	r.quiet(t)
	r.b.events <- key('h')
	g.apply(t, r.next(t))
	if got, want := g.row(0), "(0,0)-(0,0)"; got != want {
		t.Fatalf("a ref whose node is gone reads %q, want %q:\n%s", got, want, g.text())
	}
}

func TestMeasurePassLimitStopsALoopThatNeverSettles(t *testing.T) {
	var builds atomic.Int32
	r := launch(newBackend(40, 12), func(rt *twi.Runtime) func() twi.Node {
		self := twi.NewRef(rt)
		return func() twi.Node {
			builds.Add(1)
			x := 10
			if self.Bounds().Min.X == 10 {
				x = 0
			}
			return twi.Element(twi.Element(twi.Measure(self), twi.At(x, 0), twi.Text("flip")))
		}
	})
	t.Cleanup(func() {
		if err := r.stop(t); err != nil {
			t.Error(err)
		}
	})
	r.next(t)
	r.quiet(t)
	if got := builds.Load(); got != rkonst.MeasurePasses {
		t.Fatalf("a box that moves every pass built the app %d times in one frame, want %d", got, rkonst.MeasurePasses)
	}
}
