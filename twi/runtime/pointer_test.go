package runtime_test

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	rkonst "github.com/twind-dev/twind/internal/konst/runtime"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/runtime/testdata/hover"
	"github.com/twind-dev/twind/twi/style"
)

type pointed struct {
	t     *testing.T
	trace *hover.Trace
	d     *drive.Driver
	sheet style.Sheet
}

func startPointer(t *testing.T, open bool) *pointed {
	sheet, err := hover.Styles()
	if err != nil {
		t.Fatal(err)
	}
	p := &pointed{t: t, trace: &hover.Trace{Open: open}, sheet: sheet}
	p.d = drive.New(p.trace.App, drive.Size(30, 8), drive.Styles(sheet))
	t.Cleanup(func() {
		if err := p.d.Err(); err != nil {
			t.Error(err)
		}
		if err := p.d.Close(); err != nil {
			t.Error(err)
		}
	})
	return p
}

func (p *pointed) at(word string) (int, int) {
	p.t.Helper()
	for y, line := range strings.Split(p.d.Frame().Text(), "\n") {
		if x := strings.Index(line, word); x >= 0 {
			return len([]rune(line[:x])), y
		}
	}
	p.t.Fatalf("no %q in the frame:\n%s", word, p.d.Frame().Text())
	return 0, 0
}

func (p *pointed) expect(step, want string) {
	p.t.Helper()
	if got := strings.Join(p.trace.Events, ", "); got != want {
		p.t.Errorf("%s: events %q, want %q", step, got, want)
	}
	p.trace.Events = nil
}

func (p *pointed) background(word string, want style.State) {
	p.t.Helper()
	x, y := p.at(word)
	cell := p.d.Frame().Cells().At(x, y)
	bg := p.sheet.ComputeState(style.ComputedStyle{}, strings.Fields(hover.Pill), style.NodeState{States: want}).Background.RGBA
	if cell.Bg.RGBA != bg {
		p.t.Errorf("%s background %v, want %v for states %b:\n%s", word, cell.Bg.RGBA, bg, want, p.d.Frame().ANSI())
	}
}

func (p *pointed) underlined(word string) bool {
	x, y := p.at(word)
	return p.d.Frame().Cells().At(x, y).Attr&buffer.Underline != 0
}

func TestPointerHoverEnterLeave(t *testing.T) {
	p := startPointer(t, false)
	p.background("one", 0)
	p.d.Move(p.at("one"))
	p.expect("move onto one", "enter row, enter one")
	p.background("one", style.StateHover)
	p.d.Move(p.at("two"))
	p.expect("move from one to its sibling", "leave one, enter two")
	p.background("one", 0)
	p.background("two", style.StateHover)
	p.d.Move(p.at("plain"))
	p.expect("move off the row", "leave two, leave row")
	p.background("two", 0)
}

func TestPointerClickActiveAndFocus(t *testing.T) {
	p := startPointer(t, false)
	x, y := p.at("one")
	p.d.Down(x, y)
	p.expect("down on one", "enter row, enter one")
	p.background("one", style.StateActive|style.StateHover)
	p.d.Up(x, y)
	p.expect("up on one", "click one, click row")
	p.background("one", style.StateHover)
	if p.underlined("one") {
		t.Errorf("a click focused one with focus-visible:\n%s", p.d.Frame().ANSI())
	}
	p.d.Press("tab")
	if !p.underlined("two") {
		t.Errorf("tab after a click does not show focus-visible on two:\n%s", p.d.Frame().ANSI())
	}
	p.d.Down(x, y)
	p.d.Move(p.at("two"))
	p.background("one", style.StateActive)
	p.d.Up(p.at("two"))
	p.expect("down on one, up on two", "leave one, enter two")
	p.background("one", 0)
}

func TestPointerHitsByPaintOrder(t *testing.T) {
	p := startPointer(t, true)
	p.d.Click(1, 1)
	p.expect("click where the fixed z-50 overlay covers one", "enter overlay, click overlay")
	p.d.Click(p.at("two"))
	p.expect("click outside the overlay", "leave overlay, enter row, enter two, outside, click two, click row")
	if strings.Contains(p.d.Frame().Text(), "overlay") {
		t.Errorf("the overlay is still drawn after an outside click:\n%s", p.d.Frame().Text())
	}
}

func TestPointerMoveDrawsOnlyWhenAStyleChanges(t *testing.T) {
	sheet, err := hover.Styles()
	if err != nil {
		t.Fatal(err)
	}
	trace := &hover.Trace{}
	var views atomic.Int32
	r := launch(newBackend(30, 8), func(rt *twi.Runtime) func() twi.Node {
		view := trace.App(rt)
		return func() twi.Node {
			views.Add(1)
			return view()
		}
	}, twi.Styles(sheet), twi.ColorProfile(color.TrueColor))
	r.next(t)
	drawn := views.Load()
	for _, at := range [][2]int{{20, 6}, {3, 3}, {25, 5}} {
		r.b.events <- input.MouseEvent{X: at[0], Y: at[1], Action: input.MouseMove, Button: input.MouseNone}
		time.Sleep(2 * rkonst.FrameInterval)
		settled := make(chan struct{})
		r.rt.Dispatch(func() { r.rt.Dispatch(func() { close(settled) }) })
		<-settled
		if n := views.Load(); n != drawn {
			t.Fatalf("a move to %v over nothing with a hover style built the view %d times, want %d", at, n, drawn)
		}
		r.quiet(t)
	}
	r.b.events <- input.MouseEvent{X: 2, Y: 1, Action: input.MouseMove, Button: input.MouseNone}
	if f := r.next(t); !strings.Contains(f, "48;2;0;166;244") {
		t.Errorf("the move onto one wrote %q, want the sky-500 background", f)
	}
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
}

func TestPointerCoalescesMoves(t *testing.T) {
	sheet, err := hover.Styles()
	if err != nil {
		t.Fatal(err)
	}
	trace := &hover.Trace{}
	r := launch(newBackend(30, 8), trace.App, twi.Styles(sheet), twi.ColorProfile(color.TrueColor))
	r.next(t)
	const moves = 200
	start := time.Now()
	for i := range moves {
		r.b.events <- input.MouseEvent{X: 2 + 7*(i%2), Y: 1, Action: input.MouseMove, Button: input.MouseNone}
	}
	elapsed := time.Since(start)
	time.Sleep(50 * time.Millisecond)
	frames := len(r.b.frames)
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
	limit := int(elapsed/rkonst.FrameInterval) + 3
	if frames > limit {
		t.Errorf("%d moves in %v drew %d frames, want at most %d", moves, elapsed, frames, limit)
	}
	if hits := len(trace.Events); hits > 3*limit {
		t.Errorf("%d moves in %v were hit tested one by one: %d enter and leave events, want at most %d", moves, elapsed, hits, 3*limit)
	}
}
