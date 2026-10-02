package runtime_test

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

func TestViewRunsOnlyWhenItCanChange(t *testing.T) {
	var views atomic.Int64
	b := newBackend(20, 9)
	r := launch(b, func(rt *twi.Runtime) func() twi.Node {
		plain := 0
		return func() twi.Node {
			views.Add(1)
			list := []twi.NodeOption{twi.Class("col h-5 w-12 shrink-0 overflow-y-auto")}
			for i := range listRows {
				list = append(list, twi.Element(twi.Text(fmt.Sprint("row ", i))))
			}
			return twi.Element(twi.Class("col"),
				twi.Element(list...),
				twi.Text(fmt.Sprint("n=", plain)),
				twi.Element(twi.OnClick(func(*twi.Event) { plain++ }), twi.Text("click")),
				twi.Element(twi.OnPointerEnter(func() { plain += 10 }), twi.Text("enter")),
			)
		}
	}, twi.Styles(scrollSheet(t)))
	t.Cleanup(func() {
		if err := r.stop(t); err != nil {
			t.Error(err)
		}
	})
	g := blank(b)
	frame := func() { g.apply(t, r.next(t)) }
	frame()
	r.quiet(t)
	step := func(name string, rebuilt bool, want string, events ...input.Event) {
		t.Helper()
		before := views.Load()
		for _, ev := range events {
			b.events <- ev
		}
		frame()
		for len(b.frames) > 0 {
			frame()
		}
		if got := views.Load() > before; got != rebuilt {
			t.Errorf("%s: the view ran %d times, want it to run %v", name, views.Load()-before, rebuilt)
		}
		if want != "" && !strings.Contains(g.text(), want) {
			t.Errorf("%s: the screen does not show %q:\n%s", name, want, g.text())
		}
	}
	move := func(x, y int) input.Event {
		return input.MouseEvent{X: x, Y: y, Button: input.MouseNone, Action: input.MouseMove}
	}
	step("a wheel notch over a list with no listener", false, "row 3", wheel(2, 1, input.MouseWheelDown))
	step("a move inside the list, then a notch back", false, "row 0", move(3, 2), wheel(3, 2, input.MouseWheelUp))
	step("a click whose handler changes a plain field, then a notch", true, "n=1",
		input.MouseEvent{X: 1, Y: 6, Button: input.MouseLeft, Action: input.MousePress},
		input.MouseEvent{X: 1, Y: 6, Button: input.MouseLeft, Action: input.MouseRelease},
		wheel(2, 1, input.MouseWheelDown))
	step("an enter handler, then a notch", true, "n=11", move(1, 7), wheel(2, 1, input.MouseWheelDown))
	step("a resize", true, "", input.ResizeEvent{Width: 19, Height: 9})
}
