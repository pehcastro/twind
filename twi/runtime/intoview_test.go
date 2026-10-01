package runtime_test

import (
	"fmt"
	"image"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

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
