package ui

import (
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/theme"
)

type field struct {
	*drive.Driver
	state func() (string, int, int)
}

func fieldDriver(t *testing.T, value string, multi bool) *field {
	t.Helper()
	f := &field{}
	f.Driver = overlayDriver(t, 40, 9, func(rt *twi.Runtime) func() twi.Node {
		in, area := NewInput(rt), NewTextarea(rt)
		in.Insert(value)
		area.Insert(value)
		f.state = func() (string, int, int) {
			if multi {
				s, e := area.Selection()
				return area.Value(), s, e
			}
			s, e := in.Selection()
			return in.Value(), s, e
		}
		return func() twi.Node {
			field := in.Node(twi.Class("w-16"))
			if multi {
				field = area.Node(twi.Class("w-24"))
			}
			return twi.Element(twi.Class("flex flex-col p-1 h-full bg-background text-foreground"), field)
		}
	})
	return f
}

func (f *field) want(t *testing.T, value string, start, end int) {
	t.Helper()
	f.Advance(settleTime)
	if v, s, e := f.state(); v != value || s != start || e != end {
		t.Fatalf("field holds %q [%d,%d], want %q [%d,%d]:\n%s", v, s, e, value, start, end, f.Frame().Text())
	}
}

func (f *field) spot(t *testing.T, s string) (int, int) {
	t.Helper()
	x, y, ok := at(f.Frame(), s)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", s, f.Frame().Text())
	}
	return x, y
}

func TestClickPeedroBackspace(t *testing.T) {
	f := fieldDriver(t, "Peedro", false)
	x, y := f.spot(t, "Peedro")
	settledClick(f.Driver, x+2, y)
	f.want(t, "Peedro", 2, 2)
	settledPress(f.Driver, "backspace")
	f.want(t, "Pedro", 1, 1)
	if !has(f.Driver, "Pedro") || has(f.Driver, "Peedro") {
		t.Errorf("the frame does not show Pedro:\n%s", f.Frame().Text())
	}
}

func TestClickBorderAndPadding(t *testing.T) {
	f := fieldDriver(t, "Peedro", false)
	x, y := f.spot(t, "Peedro")
	for _, c := range []struct {
		name       string
		x, y, want int
	}{
		{"top border row", x + 2, y - 1, 2},
		{"bottom border row", x + 3, y + 1, 3},
		{"left padding", x - 1, y, 0},
		{"left border", x - 2, y, 0},
		{"past the end", x + 8, y, 6},
	} {
		settledClick(f.Driver, c.x, c.y)
		if _, s, e := f.state(); s != c.want || e != c.want {
			t.Errorf("%s: caret [%d,%d], want %d", c.name, s, e, c.want)
		}
	}
}

func TestClickPastTheEndShowsTheCaret(t *testing.T) {
	f := fieldDriver(t, "Peedro", false)
	x, y := f.spot(t, "Peedro")
	settledClick(f.Driver, x+9, y)
	f.want(t, "Peedro", 6, 6)
	caret := zinc(t, theme.Light).Tokens[theme.Foreground].RGBA
	if bg := f.Frame().Cells().At(x+6, y).Bg.RGBA; bg != caret {
		t.Errorf("the cell after Peedro has background %v, want the caret %v", bg, caret)
	}
}

func TestClickWideRightHalf(t *testing.T) {
	f := fieldDriver(t, "a中b", false)
	x, y := f.spot(t, "a中")
	settledClick(f.Driver, x+2, y)
	f.want(t, "a中b", 4, 4)
	settledClick(f.Driver, x+1, y)
	f.want(t, "a中b", 1, 1)
}

func TestClickScrolledField(t *testing.T) {
	f := fieldDriver(t, "0123456789abcdefghij", false)
	x, y := f.spot(t, "0123")
	settledClick(f.Driver, x, y)
	settledPress(f.Driver, "end")
	t.Logf("the caret at the end of a long value:\n%s", f.Frame().Text())
	x, y = f.spot(t, "ghij")
	settledClick(f.Driver, x-1, y)
	settledPress(f.Driver, "delete")
	f.want(t, "0123456789abcdeghij", 15, 15)
}

func TestClickFocusesField(t *testing.T) {
	f := fieldDriver(t, "hello", false)
	x, y := f.spot(t, "hello")
	settledClick(f.Driver, x+1, y)
	settledPress(f.Driver, "X")
	f.want(t, "hXello", 2, 2)
}

func TestDragSelects(t *testing.T) {
	f := fieldDriver(t, "hello world", false)
	x, y := f.spot(t, "hello")
	f.Down(x+1, y)
	f.Move(x+2, y)
	f.Move(x+4, y)
	f.Up(x+4, y)
	f.want(t, "hello world", 1, 4)
	primary := zinc(t, theme.Light).Tokens[theme.Primary].RGBA
	for i := 1; i < 4; i++ {
		if bg := f.Frame().Cells().At(x+i, y).Bg.RGBA; bg != primary {
			t.Errorf("cell %d of the selection has background %v, want the primary %v", i, bg, primary)
		}
	}
	settledPress(f.Driver, "X")
	f.want(t, "hXo world", 2, 2)
}

func TestDragEndsOnRelease(t *testing.T) {
	f := fieldDriver(t, "hello world", false)
	x, y := f.spot(t, "hello")
	f.Down(x+1, y)
	f.Move(x+3, y)
	f.Up(x+3, y)
	settledMove(f.Driver, x+6, y)
	settledMove(f.Driver, x+8, y)
	f.want(t, "hello world", 1, 3)
	settledClick(f.Driver, x+2, y)
	settledMove(f.Driver, x+5, y)
	f.want(t, "hello world", 2, 2)
}

func TestClickShiftExtends(t *testing.T) {
	f := fieldDriver(t, "hello world", false)
	x, y := f.spot(t, "hello")
	settledClick(f.Driver, x+2, y)
	f.Hold(input.ModShift)
	settledClick(f.Driver, x+8, y)
	f.Hold(0)
	f.want(t, "hello world", 2, 8)
	settledPress(f.Driver, "X")
	f.want(t, "heXrld", 3, 3)
}

func TestDragOutsideTheField(t *testing.T) {
	f := fieldDriver(t, "hello world", false)
	x, y := f.spot(t, "hello")
	f.Down(x+3, y)
	f.Move(x+20, y+3)
	f.Up(x+20, y+3)
	f.want(t, "hello world", 3, 11)
	settledMove(f.Driver, x, y)
	f.want(t, "hello world", 3, 11)
}

func TestDragAcrossRows(t *testing.T) {
	f := fieldDriver(t, "one two\nthree four\nfive", true)
	x, y := f.spot(t, "one")
	f.Down(x+4, y)
	f.Move(x+2, y+2)
	f.Up(x+2, y+2)
	f.want(t, "one two\nthree four\nfive", 4, 21)
}

func TestDragBackwardOverWide(t *testing.T) {
	f := fieldDriver(t, "ab中cd", false)
	x, y := f.spot(t, "ab中")
	f.Down(x+5, y)
	f.Move(x+3, y)
	f.Up(x+3, y)
	f.want(t, "ab中cd", 5, 6)
}

func TestClickDoubleSelectsWord(t *testing.T) {
	f := fieldDriver(t, "hello big world", false)
	x, y := f.spot(t, "big")
	f.Click(x+1, y)
	f.Click(x+1, y)
	f.want(t, "hello big world", 6, 9)
	settledPress(f.Driver, "X")
	f.want(t, "hello X world", 7, 7)
}

func TestClickTripleSelectsLine(t *testing.T) {
	f := fieldDriver(t, "one two\nthree four\nfive", true)
	x, y := f.spot(t, "three")
	f.Click(x+7, y)
	f.Click(x+7, y)
	f.Click(x+7, y)
	f.want(t, "one two\nthree four\nfive", 8, 18)
}

func TestClickSlowIsNotDouble(t *testing.T) {
	f := fieldDriver(t, "hello big world", false)
	x, y := f.spot(t, "big")
	settledClick(f.Driver, x+1, y)
	settledClick(f.Driver, x+1, y)
	f.want(t, "hello big world", 7, 7)
}

func TestClickTextareaRowAndLineEnd(t *testing.T) {
	f := fieldDriver(t, "ab\ncdef\ng", true)
	x, y := f.spot(t, "cdef")
	settledClick(f.Driver, x+2, y)
	f.want(t, "ab\ncdef\ng", 5, 5)
	settledClick(f.Driver, x+12, y-1)
	f.want(t, "ab\ncdef\ng", 2, 2)
}

func TestClickComboboxField(t *testing.T) {
	var box *Combobox
	d := overlayDriver(t, 40, 12, func(rt *twi.Runtime) func() twi.Node {
		box = NewCombobox(rt)
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col p-1 h-full w-30 bg-background text-foreground"), box.Input())
		}
	})
	x, y, _ := at(d.Frame(), "│")
	settledClick(d, x+2, y)
	d.Type("Peedro")
	d.Advance(settleTime)
	x, y, ok := at(d.Frame(), "Peedro")
	if !ok {
		t.Fatalf("no Peedro in the combobox:\n%s", d.Frame().Text())
	}
	settledClick(d, x+2, y)
	settledPress(d, "backspace")
	if got := box.field.Value(); got != "Pedro" {
		t.Errorf("combobox field holds %q, want Pedro:\n%s", got, d.Frame().Text())
	}
}
