package runtime_test

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/motion"
	"github.com/pehcastro/twind/twi/runtime"
	"github.com/pehcastro/twind/twi/terminal"
)

func anchored(t *testing.T, out runtime.Backend, b *backend, profile color.Profile, view render.Node, key string, path []int) grid {
	t.Helper()
	rt := runtime.New(runtime.Config{Clock: &clock{}, Sheet: scrollSheet(t), Profile: profile})
	tree := runtime.Tree{Root: render.Node{Classes: []string{"col"}, Children: []render.Node{view}}, Events: runtime.Node{Children: []runtime.Node{{Key: key, At: path}}}}
	rt.ScrollIntoView(key)
	r := run{b: b, done: make(chan error, 1)}
	go func() { r.done <- rt.Run(out, func() runtime.Tree { return tree }) }()
	g := blank(b)
	g.apply(t, r.next(t))
	g.settle(t, b)
	rt.Quit()
	if err := r.result(t); err != nil {
		t.Fatal(err)
	}
	return g
}

func (g grid) settle(t *testing.T, b *backend) {
	t.Helper()
	for {
		select {
		case f := <-b.frames:
			g.apply(t, f)
		case <-time.After(300 * time.Millisecond):
			return
		}
	}
}

func numberedRows() []render.Node {
	rows := make([]render.Node, listRows)
	for i := range rows {
		rows[i] = render.Node{Text: fmt.Sprint("row ", i)}
	}
	return rows
}

func TestScrollIntoViewLandsWhereTheSlideSettles(t *testing.T) {
	slide := &motion.Presence{Offset: motion.Offset{Opacity: 1, Scale: 1, Y: 3}, Duration: 200 * time.Millisecond}
	page := render.Node{Classes: []string{"col", "shrink-0"}, Enter: slide, Children: numberedRows()}
	view := render.Node{Classes: []string{"col", "h-5", "w-12", "shrink-0", "overflow-y-auto"}, Children: []render.Node{page}}
	b := newBackend(20, 6)
	g := anchored(t, b, b, color.None, view, "row 12", []int{0, 0, 12})
	if got := strings.TrimSpace(string(g[0][:listWidth-1])); got != "row 12" {
		t.Errorf("after the slide settled the view starts at %q, want row 12:\n%s", got, g.text())
	}
}

func TestScrollIntoViewOnHalfRows(t *testing.T) {
	for _, c := range []struct {
		row, screen int
		why         string
	}{{listRows - 1, listView - 1, "the last row above the view's half-clipped last row"}, {7, 1, "the anchor below the view's half-clipped first row"}} {
		b := gdiBackend{newBackend(20, 6), make(chan terminal.Pixels, 1<<10)}
		b.caps = terminal.Capabilities{Graphics: terminal.GraphicsGDI, CellPixels: image.Pt(10, 20)}
		view := render.Node{Classes: []string{"col", "h-5", "w-12", "shrink-0", "overflow-y-auto", "pt-h"}, Children: numberedRows()}
		key := fmt.Sprint("row ", c.row)
		g := anchored(t, b, b.backend, color.TrueColor, view, key, []int{0, c.row})
		if got := strings.TrimSpace(string(g[c.screen][:listWidth-1])); got != key {
			t.Errorf("screen row %d shows %q, want %s:\n%s", c.screen, got, c.why, g.text())
		}
	}
}

func TestScrollIntoViewDuringMeasure(t *testing.T) {
	list := func(opts ...twi.NodeOption) twi.Node {
		opts = append(opts, twi.Focusable(), twi.Class("col h-5 w-12 shrink-0 overflow-y-auto"))
		for i := range listRows {
			opts = append(opts, twi.Element(twi.Key(fmt.Sprint("row-", i)), twi.Text(fmt.Sprint("row ", i))))
		}
		return twi.Element(twi.Class("col"), twi.Element(opts...))
	}
	check := func(t *testing.T, r run, g grid, want int) {
		t.Helper()
		g.apply(t, r.next(t))
		select {
		case f := <-r.b.frames:
			g.apply(t, f)
		case <-time.After(100 * time.Millisecond):
		}
		if got := g.first(); got != want {
			t.Errorf("the frame shows row %d first, want row %d:\n%s", got, want, g.text())
		}
		r.quiet(t)
	}
	t.Run("ref handler", func(t *testing.T) {
		r := launch(newBackend(20, 8), func(rt *twi.Runtime) func() twi.Node {
			box, asked := twi.NewRef(rt), false
			return func() twi.Node {
				if !asked && box.Bounds() != (image.Rectangle{}) {
					asked = true
					rt.ScrollIntoView("row-12")
				}
				return list(twi.Measure(box))
			}
		}, twi.Styles(scrollSheet(t)))
		t.Cleanup(func() {
			if err := r.stop(t); err != nil {
				t.Error(err)
			}
		})
		check(t, r, blank(r.b), 12)
	})
	t.Run("OnScroll handler", func(t *testing.T) {
		r := launch(newBackend(20, 8), func(rt *twi.Runtime) func() twi.Node {
			asked := false
			scrolled := twi.OnScroll(func(image.Point) {
				if !asked {
					asked = true
					rt.ScrollIntoView("row-15")
				}
			})
			return func() twi.Node { return list(scrolled) }
		}, twi.Styles(scrollSheet(t)))
		t.Cleanup(func() {
			if err := r.stop(t); err != nil {
				t.Error(err)
			}
		})
		g := blank(r.b)
		check(t, r, g, 0)
		r.b.events <- wheel(2, 1, input.MouseWheelDown)
		check(t, r, g, 15)
	})
}
