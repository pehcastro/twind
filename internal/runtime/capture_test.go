package runtime_test

import (
	"fmt"
	"slices"
	"testing"

	termkonst "github.com/pehcastro/twind/internal/konst/terminal"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/input"
)

type pointerLog []string

func (l *pointerLog) listen(name string) []twi.NodeOption {
	record := func(verb string) func(*twi.Event) {
		return func(e *twi.Event) {
			at := e.Offset()
			*l = append(*l, fmt.Sprintf("%s %s %d,%d b%d m%d", name, verb, at.X, at.Y, e.Mouse.Button, e.Mouse.Modifiers))
		}
	}
	return []twi.NodeOption{
		twi.OnPointerDown(record("down")),
		twi.OnPointerMove(record("move")),
		twi.OnPointerUp(record("up")),
		twi.OnClick(record("click")),
	}
}

func (l *pointerLog) expect(t *testing.T, step string, want ...string) {
	t.Helper()
	if !slices.Equal(*l, want) {
		t.Errorf("%s:\n got %q\nwant %q", step, *l, want)
	}
	*l = nil
}

func pointerApp(log *pointerLog) drive.App {
	return func(rt *twi.Runtime) func() twi.Node {
		redraws := 0
		return func() twi.Node {
			box := append(log.listen("box"), twi.At(5, 2), twi.Element(twi.Key("inner"), twi.Text("0123456789")))
			other := append(log.listen("other"), twi.At(20, 6), twi.Text("other"))
			return twi.Element(
				twi.OnKey(func(input.KeyEvent) { redraws++; rt.Invalidate() }),
				twi.Element(box...),
				twi.Element(other...),
				twi.Text(fmt.Sprint("redraws ", redraws)),
			)
		}
	}
}

func startDriver(t *testing.T, app drive.App, opts ...drive.Option) *drive.Driver {
	d := drive.New(app, opts...)
	t.Cleanup(func() {
		if err := d.Err(); err != nil {
			t.Error(err)
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	return d
}

func TestPointerMoveUpOffsetInTheListenersCells(t *testing.T) {
	var log pointerLog
	d := startDriver(t, pointerApp(&log), drive.Size(30, 8))
	d.Move(8, 2)
	log.expect(t, "a move over the inner child", "box move 3,0 b0 m0")
	d.Press("x")
	log.expect(t, "a redraw with the pointer still")
	d.Down(9, 2)
	d.Up(9, 2)
	log.expect(t, "a click on the box", "box down 4,0 b1 m0", "box up 4,0 b1 m0", "box click 4,0 b1 m0")
}

func TestPointerCaptureOutsideTheElement(t *testing.T) {
	var log pointerLog
	d := startDriver(t, pointerApp(&log), drive.Size(30, 8))
	d.Down(6, 2)
	log.expect(t, "down on the box", "box down 1,0 b1 m0")
	d.Move(21, 6)
	log.expect(t, "a drag over the other element", "box move 16,4 b1 m0")
	d.Move(0, 0)
	log.expect(t, "a drag off every element", "box move -5,-2 b1 m0")
	d.Up(21, 6)
	log.expect(t, "the release over the other element", "box up 16,4 b1 m0")
	d.Move(22, 6)
	log.expect(t, "a move after the release", "other move 2,0 b0 m0")
	d.DownWith(input.MouseRight, 6, 2)
	d.Move(21, 6)
	d.UpWith(input.MouseRight, 21, 6)
	d.Move(22, 6)
	log.expect(t, "a right drag", "box down 1,0 b3 m0", "box move 16,4 b3 m0", "box up 16,4 b3 m0", "other move 2,0 b0 m0")
}

func TestPointerCaptureOfARemovedElement(t *testing.T) {
	var log pointerLog
	d := startDriver(t, func(rt *twi.Runtime) func() twi.Node {
		shown := true
		return func() twi.Node {
			root := append(log.listen("root"), twi.Text("root"))
			if shown {
				box := append(log.listen("box"), twi.At(5, 2), twi.OnPointerDown(func(*twi.Event) { shown = false; rt.Invalidate() }), twi.Text("box"))
				root = append(root, twi.Element(box...))
			}
			return twi.Element(root...)
		}
	}, drive.Size(30, 8))
	d.Down(6, 2)
	d.Move(2, 0)
	d.Up(2, 0)
	log.expect(t, "the pressed element is removed by its own down", "box down 1,0 b1 m0", "root down 6,2 b1 m0", "root move 2,0 b1 m0", "root up 2,0 b1 m0")
}

func TestPointerMoveInsideAScrolledArea(t *testing.T) {
	var log pointerLog
	d := startDriver(t, func(*twi.Runtime) func() twi.Node {
		return func() twi.Node {
			rows := log.listen("content")
			for i := range listRows {
				rows = append(rows, twi.Text(fmt.Sprint("row ", i)))
			}
			return twi.Element(twi.Class("col h-5 w-12 shrink-0 overflow-y-auto"), twi.Element(append(rows, twi.Class("col shrink-0"))...))
		}
	}, drive.Size(20, 8), drive.Styles(scrollSheet(t)))
	d.Wheel(2, 1, 1)
	log = nil
	d.Click(3, 1)
	y := 1 + termkonst.WheelLines
	log.expect(t, "a click on the second visible row after one wheel notch",
		fmt.Sprintf("content down 3,%d b1 m0", y), fmt.Sprintf("content up 3,%d b1 m0", y), fmt.Sprintf("content click 3,%d b1 m0", y))
}

func TestPointerModifierShiftClick(t *testing.T) {
	var log pointerLog
	d := startDriver(t, pointerApp(&log), drive.Size(30, 8))
	shift, ctrlAlt := input.ModShift, input.ModCtrl|input.ModAlt
	d.Hold(shift)
	d.Click(7, 2)
	d.Hold(ctrlAlt)
	d.Click(7, 2)
	d.Hold(0)
	log.expect(t, "Shift+click, then Ctrl+Alt+click",
		fmt.Sprintf("box down 2,0 b1 m%d", shift), fmt.Sprintf("box up 2,0 b1 m%d", shift), fmt.Sprintf("box click 2,0 b1 m%d", shift),
		fmt.Sprintf("box down 2,0 b1 m%d", ctrlAlt), fmt.Sprintf("box up 2,0 b1 m%d", ctrlAlt), fmt.Sprintf("box click 2,0 b1 m%d", ctrlAlt))
}
