package render_test

import (
	"math"
	"testing"
	"time"

	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/internal/render/testdata/sheet"
	"github.com/twind-dev/twind/twi/style"
)

func stated(value string) *style.NodeState {
	return &style.NodeState{Attrs: []style.Attr{{Name: "data-state", Value: value}}}
}

func opening() render.Node {
	return render.Node{Classes: classes(sheet.Opening), State: stated("open"), Children: []render.Node{{Text: "Edit profile"}}}
}

func TestClassPresenceEnter(t *testing.T) {
	r := newMotionRun(t)
	closed := screen(render.Node{Text: "behind"})
	open := screen(render.Node{Text: "behind"}, opening())
	r.at(0, closed)
	r.at(10*time.Second, open)
	final := r.at(20*time.Second, open).Children[1]
	r.tree = render.Tree{}
	r.at(0, closed)
	var opacities []float64
	for _, after := range []time.Duration{0, 50 * time.Millisecond, 100 * time.Millisecond, 150 * time.Millisecond, 199 * time.Millisecond, 200 * time.Millisecond, 250 * time.Millisecond} {
		d := r.at(time.Second+after, open).Children[1]
		t.Logf("%3d ms after mount: opacity %.3f bounds %+v text %+v", after.Milliseconds(), d.Opacity, d.Bounds, d.Children[0].Bounds)
		opacities = append(opacities, d.Opacity)
		r.wake(after < 200*time.Millisecond)
		switch {
		case after >= 200*time.Millisecond && (d.Opacity != 1 || d.Bounds != final.Bounds):
			t.Errorf("%v after mount opacity %v bounds %+v, want 1 %+v", after, d.Opacity, d.Bounds, final.Bounds)
		case after == 0 && (d.Opacity != 0 || d.Bounds.W != 57):
			t.Errorf("at mount opacity %v width %d, want 0 and 57 (zoom-in-95 of 60)", d.Opacity, d.Bounds.W)
		case after > 0 && after < 200*time.Millisecond && (d.Bounds.W > final.Bounds.W || d.Bounds.W < 57):
			t.Errorf("%v after mount width %d, want between 57 and 60", after, d.Bounds.W)
		}
	}
	for i := 1; i < 5; i++ {
		if !(opacities[i] > opacities[i-1] && opacities[i] < 1) {
			t.Errorf("enter opacities %v, want 0 rising to 1 at 200 ms", opacities)
			break
		}
	}
	hovered := screen(render.Node{Text: "behind"}, opening())
	hovered.Children[1].Classes = append(hovered.Children[1].Classes, sheet.Nowrap)
	if d := r.at(1300*time.Millisecond, hovered).Children[1]; d.Opacity != 1 || d.Bounds != final.Bounds {
		t.Errorf("a restyle after the enter restarted it: opacity %v bounds %+v", d.Opacity, d.Bounds)
	}
	r.wake(false)
}

func TestClassPresenceExit(t *testing.T) {
	r := newMotionRun(t)
	gone := screen(render.Node{Text: "behind"})
	open := screen(render.Node{Text: "behind"}, opening())
	closing := screen(render.Node{Text: "behind"}, render.Node{Classes: classes(sheet.Closing), State: stated("closed"), Children: []render.Node{{Text: "Edit profile"}}})
	r.at(0, open)
	r.at(time.Second, open)
	r.wake(false)
	if d := r.at(2*time.Second, closing).Children[1]; d.Opacity != 1 {
		t.Errorf("the closed frame starts its exit at opacity %v, want 1", d.Opacity)
	}
	r.wake(true)
	var last float64 = 1
	for _, step := range []struct {
		after time.Duration
		kids  int
	}{
		{16 * time.Millisecond, 2},
		{75 * time.Millisecond, 2},
		{149 * time.Millisecond, 2},
		{150 * time.Millisecond, 1},
		{time.Second, 1},
	} {
		root := r.at(2*time.Second+step.after, gone)
		if len(root.Children) != step.kids {
			t.Fatalf("%v after the closed frame %d children, want %d", step.after, len(root.Children), step.kids)
		}
		if step.kids == 2 {
			d := root.Children[1]
			t.Logf("%3d ms after data-state=closed, node removed: kept, opacity %.3f bounds %+v", step.after.Milliseconds(), d.Opacity, d.Bounds)
			if d.Opacity >= last || d.Opacity <= 0 || d.Bounds.W != 60 {
				t.Errorf("%v: held node opacity %v (before %v) width %d, want falling and 60 wide", step.after, d.Opacity, last, d.Bounds.W)
			}
			last = d.Opacity
			if r.tree.Reaches([]int{1}) {
				t.Error("a held node is reachable by its old path")
			}
		} else {
			t.Logf("%3d ms after data-state=closed: gone", step.after.Milliseconds())
		}
		r.wake(step.kids == 2)
	}
	r.at(5*time.Second, open)
	if n := len(r.at(6*time.Second, gone).Children); n != 1 {
		t.Errorf("a node removed with no closed frame is kept: %d children", n)
	}
	r.wake(false)
}

func TestClassPresenceSubtreeHeldForItsLongestExit(t *testing.T) {
	r := newMotionRun(t)
	dialog := func(state string) render.Node {
		return screen(render.Node{Text: "behind"}, render.Node{Classes: classes(sheet.Backdrop), State: stated(state), Children: []render.Node{
			{Classes: classes(sheet.Panel), State: stated(state), Children: []render.Node{{Text: "Edit profile"}}},
		}})
	}
	r.at(0, dialog("open"))
	r.at(time.Second, dialog("open"))
	r.at(2*time.Second, dialog("closed"))
	panel := 1.0
	for _, step := range []struct {
		after time.Duration
		kids  int
	}{{100 * time.Millisecond, 2}, {150 * time.Millisecond, 2}, {199 * time.Millisecond, 2}, {200 * time.Millisecond, 1}} {
		root := r.at(2*time.Second+step.after, screen(render.Node{Text: "behind"}))
		if len(root.Children) != step.kids {
			t.Fatalf("%v after close %d children, want %d (the panel's duration-200 outlasts the backdrop's 150 ms)", step.after, len(root.Children), step.kids)
		}
		if step.kids == 2 {
			b := root.Children[1]
			p := b.Children[0]
			t.Logf("%3d ms after close: backdrop opacity %.3f panel opacity %.3f width %d", step.after.Milliseconds(), b.Opacity, p.Opacity, p.Bounds.W)
			if p.Opacity >= panel || step.after < 150*time.Millisecond && b.Opacity >= 1 {
				t.Errorf("%v after close: backdrop %v panel %v (before %v), want both fading", step.after, b.Opacity, p.Opacity, panel)
			}
			panel = p.Opacity
		}
		r.wake(step.kids == 2)
	}
}

func TestClassPresenceTypedWins(t *testing.T) {
	typed := opening()
	typed.Enter = zoomIn()
	for _, after := range []time.Duration{75 * time.Millisecond, 150 * time.Millisecond} {
		r := newMotionRun(t)
		r.at(0, screen())
		r.at(time.Second, screen(typed))
		d := r.at(time.Second+after, screen(typed)).Children[0]
		want := zoomIn().Enter(after).Opacity
		t.Logf("%3d ms: opacity %.3f, typed presence alone %.3f", after.Milliseconds(), d.Opacity, want)
		if math.Abs(d.Opacity-want) > 1e-9 {
			t.Errorf("%v after mount opacity %v, want the typed presence's %v alone", after, d.Opacity, want)
		}
	}
}

func TestClassPresenceReducedMotion(t *testing.T) {
	r := newMotionRun(t)
	r.frame.ReducedMotion = true
	r.at(0, screen())
	if d := r.at(time.Second, screen(opening())).Children[0]; d.Opacity != 1 || d.Bounds.W != 60 {
		t.Errorf("reduced motion class enter: opacity %v width %d, want 1 and 60", d.Opacity, d.Bounds.W)
	}
	r.wake(false)
	r.at(2*time.Second, screen(render.Node{Classes: classes(sheet.Closing), State: stated("closed")}))
	if n := len(r.at(2*time.Second+time.Millisecond, screen()).Children); n != 0 {
		t.Errorf("reduced motion held a class exit: %d children", n)
	}
	r.wake(false)
}

func TestClassPresenceSlideAndScale(t *testing.T) {
	r := newMotionRun(t)
	root := func(extra string) render.Node {
		return screen(render.Node{Classes: classes(sheet.Sliding)}, render.Node{Classes: classes(sheet.Zoomed)}, render.Node{Classes: classes(sheet.Growing, extra)})
	}
	n := r.at(0, root(""))
	slide, zoomed := n.Children[0], n.Children[1]
	t.Logf("slide at mount %+v, scale-50 %+v", slide.Bounds, zoomed.Bounds)
	if slide.Bounds.X != 6 || slide.Bounds.Y != -2 {
		t.Errorf("slide-in-from-left-4 slide-in-from-top over -translate-y-1/2 at mount %+v, want x 6 y -2", slide.Bounds)
	}
	if zoomed.Bounds.W != 10 || zoomed.Bounds.H != 2 || zoomed.Bounds.X != 5 || zoomed.Bounds.Y != 1 {
		t.Errorf("scale-50 on a 20x4 box at (0,0) drawn %+v, want 10x2 at (5,1)", zoomed.Bounds)
	}
	r.at(time.Second, root(""))
	grown := r.at(time.Second, root("scale-50")).Children[2]
	half := r.at(time.Second+100*time.Millisecond, root("scale-50")).Children[2]
	end := r.at(time.Second+200*time.Millisecond, root("scale-50")).Children[2]
	t.Logf("transition-transform to scale-50: 0 ms %+v, 100 ms %+v, 200 ms %+v", grown.Bounds, half.Bounds, end.Bounds)
	if grown.Bounds.W != 20 || half.Bounds.W != 15 || end.Bounds.W != 10 {
		t.Errorf("transition-transform widths %d %d %d, want 20 15 10", grown.Bounds.W, half.Bounds.W, end.Bounds.W)
	}
	if s := r.at(2*time.Second, root("")).Children[0]; s.Bounds.X != 10 || s.Bounds.Y != 2 {
		t.Errorf("slide at rest %+v, want x 10 y 2", s.Bounds)
	}
}
