package render_test

import (
	"strings"
	"testing"
	"time"

	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/internal/render/testdata/sheet"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/motion"
	"github.com/twind-dev/twind/twi/scene"
)

type motionRun struct {
	t     *testing.T
	tree  render.Tree
	frame render.Frame
}

func newMotionRun(t *testing.T) *motionRun {
	t.Helper()
	styles, err := sheet.Styles()
	if err != nil {
		t.Fatal(err)
	}
	return &motionRun{t: t, frame: render.Frame{Sheet: styles, Width: 80, Height: layout.Length{Unit: layout.Cells, Value: 24}}}
}

func (r *motionRun) at(now time.Duration, root render.Node) scene.Node {
	r.t.Helper()
	r.frame.Now = now
	n, err := r.tree.Scene(root, r.frame)
	if err != nil {
		r.t.Fatal(err)
	}
	return n
}

func (r *motionRun) wake(want bool) {
	r.t.Helper()
	if at, moving := r.tree.Wake(); moving != want {
		r.t.Errorf("at %v wake %v moving %v, want moving %v", r.frame.Now, at, moving, want)
	}
}

func screen(children ...render.Node) render.Node {
	return render.Node{Classes: strings.Fields(sheet.Page), Children: children}
}

func classes(s ...string) []string { return strings.Fields(strings.Join(s, " ")) }

func TestTransitionColoursOKLab(t *testing.T) {
	r := newMotionRun(t)
	light := screen(render.Node{Classes: classes(sheet.Fading, sheet.Light)})
	dark := screen(render.Node{Classes: classes(sheet.Fading, sheet.Dark)})
	for _, now := range []time.Duration{0, time.Second} {
		if got := r.at(now, light).Children[0].Background.RGBA; got != (color.RGBA{R: 244, G: 244, B: 245, A: 255}) {
			t.Fatalf("at rest %v background %v, want zinc-100", now, got)
		}
		r.wake(false)
	}
	var line motion.Timeline
	id := line.Start(0, motion.Color(color.RGBA{R: 244, G: 244, B: 245, A: 255}), motion.Color(color.RGBA{R: 24, G: 24, B: 27, A: 255}), motion.Tween(200*time.Millisecond, motion.Linear()))
	line.Step(100 * time.Millisecond)
	for _, step := range []struct {
		after  time.Duration
		want   color.RGBA
		moving bool
	}{
		{0, color.RGBA{R: 244, G: 244, B: 245, A: 255}, true},
		{100 * time.Millisecond, line.Value(id).Color(), true},
		{200 * time.Millisecond, color.RGBA{R: 24, G: 24, B: 27, A: 255}, false},
		{300 * time.Millisecond, color.RGBA{R: 24, G: 24, B: 27, A: 255}, false},
	} {
		got := r.at(time.Second+step.after, dark).Children[0].Background.RGBA
		t.Logf("%3d ms after the class change: background %v", step.after.Milliseconds(), got)
		if got != step.want {
			t.Errorf("%v after the class change background %v, want %v", step.after, got, step.want)
		}
		r.wake(step.moving)
	}
}

func TestKeyframePulse(t *testing.T) {
	r := newMotionRun(t)
	root := screen(render.Node{Classes: classes(sheet.Pulsing)})
	for _, step := range []struct {
		now     time.Duration
		opacity float64
	}{
		{0, 1},
		{500 * time.Millisecond, -1},
		{time.Second, 0.5},
		{1500 * time.Millisecond, -1},
		{2 * time.Second, 1},
		{3 * time.Second, 0.5},
	} {
		got := r.at(step.now, root).Children[0].Opacity
		t.Logf("pulse at %v: opacity %.3f", step.now, got)
		if step.opacity >= 0 && got != step.opacity || step.opacity < 0 && (got <= 0.5 || got >= 1) {
			t.Errorf("pulse at %v opacity %v, want %v (-1: strictly between 0.5 and 1)", step.now, got, step.opacity)
		}
		r.wake(true)
	}
	r.at(4*time.Second, screen())
	r.wake(false)
}

func zoomIn() *motion.Presence {
	return &motion.Presence{Offset: motion.Offset{Opacity: 0, Scale: 0.95}, Duration: 150 * time.Millisecond, Easing: motion.CubicBezier(0.25, 0.1, 0.25, 1)}
}

func dialog() render.Node {
	return render.Node{Classes: classes(sheet.DialogBox), Enter: zoomIn(), Exit: zoomIn(), Children: []render.Node{
		{Classes: classes(sheet.DialogHead), Children: []render.Node{{Text: "Edit profile"}}},
		{Classes: classes(sheet.Muted), Children: []render.Node{{Text: "Make changes to your profile here."}}},
	}}
}

func TestPresenceDialogZoomIn(t *testing.T) {
	r := newMotionRun(t)
	closed := screen(render.Node{Text: "behind"})
	open := screen(render.Node{Text: "behind"}, dialog())
	r.at(0, closed)
	r.at(10*time.Second, open)
	rest := r.at(20*time.Second, open)
	r.tree = render.Tree{}
	r.at(0, closed)
	r.wake(false)
	final := rest.Children[1]
	var opacities []float64
	for _, after := range []time.Duration{0, 50 * time.Millisecond, 100 * time.Millisecond, 150 * time.Millisecond, 200 * time.Millisecond} {
		d := r.at(time.Second+after, open).Children[1]
		t.Logf("%3d ms after mount: opacity %.3f bounds %+v title %+v", after.Milliseconds(), d.Opacity, d.Bounds, d.Children[0].Bounds)
		opacities = append(opacities, d.Opacity)
		r.wake(after < 150*time.Millisecond)
		if after >= 150*time.Millisecond {
			if d.Opacity != 1 || d.Bounds != final.Bounds || d.Children[0].Bounds != final.Children[0].Bounds {
				t.Errorf("%v after mount opacity %v bounds %+v title %+v, want 1 %+v %+v", after, d.Opacity, d.Bounds, d.Children[0].Bounds, final.Bounds, final.Children[0].Bounds)
			}
			continue
		}
		b := final.Bounds
		if off := func(a, b int) bool { return a-b > 1 || b-a > 1 }; d.Bounds.W > b.W || d.Bounds.H > b.H || off(2*d.Bounds.X+d.Bounds.W, 2*b.X+b.W) || off(2*d.Bounds.Y+d.Bounds.H, 2*b.Y+b.H) {
			t.Errorf("%v after mount bounds %+v, want at most %+v about the same centre, within half a cell", after, d.Bounds, b)
		}
		if title := d.Children[0].Bounds; title.W != final.Children[0].Bounds.W || title.X < d.Bounds.X || title.Y < d.Bounds.Y {
			t.Errorf("%v after mount title %+v escapes the zoomed dialog %+v or changed width", after, title, d.Bounds)
		}
	}
	if opacities[0] != 0 || !(opacities[1] > 0 && opacities[1] < opacities[2] && opacities[2] < 1) {
		t.Errorf("enter opacities %v, want 0 rising to 1", opacities)
	}
	restyled := screen(render.Node{Text: "behind"}, dialog())
	restyled.Children[1].Classes = append(restyled.Children[1].Classes, "shadow-sm")
	if d := r.at(1300*time.Millisecond, restyled).Children[1]; d.Opacity != 1 || d.Bounds != final.Bounds {
		t.Errorf("a restyle after the enter restarted it: opacity %v bounds %+v", d.Opacity, d.Bounds)
	}
	r.wake(false)
}

func TestPresenceExitKeepsNode(t *testing.T) {
	r := newMotionRun(t)
	closed := screen(render.Node{Text: "behind"})
	open := screen(render.Node{Text: "behind"}, dialog())
	r.at(0, open)
	r.at(time.Second, open)
	r.wake(false)
	if !r.tree.Reaches([]int{1}) {
		t.Fatal("the open dialog is not reachable at path 1")
	}
	for _, step := range []struct {
		after time.Duration
		kids  int
	}{
		{0, 2},
		{75 * time.Millisecond, 2},
		{149 * time.Millisecond, 2},
		{150 * time.Millisecond, 1},
		{time.Second, 1},
	} {
		root := r.at(2*time.Second+step.after, closed)
		t.Logf("%3d ms after removal: %d children", step.after.Milliseconds(), len(root.Children))
		if len(root.Children) != step.kids {
			t.Fatalf("%v after removal %d children, want %d", step.after, len(root.Children), step.kids)
		}
		if step.kids == 2 {
			d := root.Children[1]
			t.Logf("  exiting dialog opacity %.3f bounds %+v", d.Opacity, d.Bounds)
			if step.after > 0 && (d.Opacity <= 0 || d.Opacity >= 1 || d.Bounds.W >= 60) {
				t.Errorf("%v after removal dialog opacity %v width %d, want fading and shrinking", step.after, d.Opacity, d.Bounds.W)
			}
			if r.tree.Reaches([]int{1}) {
				t.Errorf("an exiting node is reachable by its old path")
			}
		}
		r.wake(step.kids == 2)
	}
	reopened := r.at(4*time.Second, open)
	if d := reopened.Children[1]; d.Opacity != 0 {
		t.Errorf("a dialog opened again after its exit starts at opacity %v, want 0", d.Opacity)
	}
	r.wake(true)
	r.at(5*time.Second, screen(render.Node{Text: "behind"}, render.Node{Classes: classes(sheet.DialogBox)}))
	r.wake(false)
}

func TestPresenceReducedMotion(t *testing.T) {
	r := newMotionRun(t)
	r.frame.ReducedMotion = true
	closed := screen(render.Node{Text: "behind"})
	open := screen(render.Node{Text: "behind"}, dialog())
	r.at(0, closed)
	if d := r.at(time.Second, open).Children[1]; d.Opacity != 1 || d.Bounds.W != 60 {
		t.Errorf("reduced motion enter: opacity %v width %d, want 1 and 60", d.Opacity, d.Bounds.W)
	}
	r.wake(false)
	if n := len(r.at(2*time.Second, closed).Children); n != 1 {
		t.Errorf("reduced motion exit kept the dialog: %d children", n)
	}
	r.wake(false)
}

func TestTransitionTranslateDrawn(t *testing.T) {
	r := newMotionRun(t)
	for i, now := range []time.Duration{0, time.Second, 2 * time.Second, 3 * time.Second} {
		shade := []string{sheet.Light, sheet.Dark}[i%2]
		n := r.at(now, screen(
			render.Node{Classes: classes(sheet.Shifted, "h-3", "w-20", shade), Children: []render.Node{{Classes: classes("w-10"), Children: []render.Node{{Text: "moved"}}}}},
			render.Node{Classes: classes(sheet.Half)},
		))
		shifted, half := n.Children[0], n.Children[1]
		if inner := shifted.Children[0].Children[0].Bounds; shifted.Bounds.X != 2 || shifted.Bounds.Y != 1 || inner.X != 2 || inner.Y != 1 {
			t.Errorf("frame %v: translate-x-2 translate-y-1 box %+v text %+v, want both at (2,1)", now, shifted.Bounds, inner)
		}
		if half.Bounds.X != -5 || half.Bounds.Y != 3 {
			t.Errorf("frame %v: -translate-x-1/2 on w-10 at %+v, want x -5 y 3", now, half.Bounds)
		}
		r.wake(false)
	}
}
