package runtime_test

import (
	"fmt"
	"image"
	"slices"
	"testing"

	termkonst "github.com/pehcastro/twind/internal/konst/terminal"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
)

func TestOnScrollFiresOnceWithTheNewOffset(t *testing.T) {
	fired := make(chan image.Point, 64)
	r := launch(newBackend(20, 8), func(rt *twi.Runtime) func() twi.Node {
		presses := 0
		return func() twi.Node {
			list := []twi.NodeOption{twi.Focusable(), twi.Class("col h-5 w-12 shrink-0 overflow-y-auto"), twi.OnScroll(func(at image.Point) {
				fired <- at
				rt.Invalidate()
			})}
			for i := range listRows {
				list = append(list, twi.Element(twi.Key(fmt.Sprint("row-", i)), twi.Text(fmt.Sprint("row ", i))))
			}
			return twi.Element(twi.Class("col"), twi.OnKey(func(e *twi.Event) {
				switch e.Key.Rune {
				case 'x':
					presses++
					rt.Invalidate()
				case 'j':
					rt.ScrollIntoView("row-12")
				}
			}), twi.Element(list...), twi.Text(fmt.Sprint("x pressed ", presses)))
		}
	}, twi.Styles(scrollSheet(t)))
	t.Cleanup(func() {
		if err := r.stop(t); err != nil {
			t.Error(err)
		}
	})
	step := func(name string, ev input.Event, want ...image.Point) {
		t.Helper()
		if ev != nil {
			r.b.events <- ev
		}
		r.next(t)
		r.quiet(t)
		var got []image.Point
		for len(fired) > 0 {
			got = append(got, <-fired)
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: fired %v, want %v", name, got, want)
		}
	}
	step("the first frame", nil)
	step("a wheel notch over the list", wheel(2, 1, input.MouseWheelDown), image.Pt(0, termkonst.WheelLines))
	step("a key that only invalidates", key('x'))
	r.b.events <- press(input.KeyTab)
	step("page down", press(input.KeyPageDown), image.Pt(0, termkonst.WheelLines+listView))
	step("ScrollIntoView row 12", key('j'), image.Pt(0, 12))
}
