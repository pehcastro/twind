package runtime_test

import (
	"image"
	"slices"
	"strings"
	"testing"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/runtime"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/runtime/testdata/frames"
	"github.com/twind-dev/twind/twi/text"
)

func driveFrames(t *testing.T, app drive.App, opts ...drive.Option) *drive.Driver {
	t.Helper()
	sheet, err := frames.Styles()
	if err != nil {
		t.Fatal(err)
	}
	d := drive.New(app, append([]drive.Option{drive.Styles(sheet)}, opts...)...)
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

func cells(f drive.Frame, x, y, n int) string {
	var b strings.Builder
	for i := range n {
		b.WriteString(f.Cells().At(x+i, y).Grapheme)
	}
	return b.String()
}

func TestTopLayerMenusPaintWholeInOpenOrder(t *testing.T) {
	l := &frames.Layers{}
	d := driveFrames(t, l.App, drive.Size(34, 8))
	d.Press("a")
	t.Logf("menu a open:\n%s", d.Frame().Text())
	for y, item := range []string{"a1", "a2", "a3", "a4"} {
		if got := cells(d.Frame(), 2, y+2, 2); got != item {
			t.Errorf("menu a row %d shows %q at column 2, want %q: the card clips its menu", y+2, got, item)
		}
	}
	d.Press("b")
	t.Logf("b opened after a:\n%s", d.Frame().Text())
	if got := cells(d.Frame(), 2, 3, 4); got != "a2b1" {
		t.Errorf("b opened last, row 3 shows %q, want a2b1: b over menu a", got)
	}
	d.Press("a")
	d.Press("a")
	t.Logf("a reopened after b:\n%s", d.Frame().Text())
	if got := cells(d.Frame(), 2, 3, 6); got != "a2    " {
		t.Errorf("a reopened last, row 3 shows %q, want menu a over b1", got)
	}
	if got := cells(d.Frame(), 4, 6, 2); got != "b4" {
		t.Errorf("row 6 shows %q at column 4, want b4 below menu a", got)
	}
}

func TestClicksPassThroughHiddenAndPointerNone(t *testing.T) {
	l := &frames.Layers{}
	d := driveFrames(t, l.App, drive.Size(34, 8))
	t.Logf("frame:\n%s", d.Frame().Text())
	for y, word := range []string{"press", "ghost"} {
		if got := cells(d.Frame(), 18, y, 5); got != word {
			t.Fatalf("row %d shows %q at column 18, want %q", y, got, word)
		}
		d.Click(19, y)
	}
	if want := []string{"press", "ghost"}; !slices.Equal(l.Clicks, want) {
		t.Errorf("clicks reached %v, want %v: a pointer-events-none or invisible box took them", l.Clicks, want)
	}
}

func TestWidthsAndCellPixelsFromTheBackend(t *testing.T) {
	black := color.RGBA{A: 255}
	for _, c := range []struct {
		name         string
		opts         []drive.Option
		border, rows int
	}{
		{"unicode defaults, nominal cell", nil, 5, 6},
		{"Widths{Flag: 1}, cell 12x16", []drive.Option{drive.Widths(text.Widths{text.Flag: 1}), drive.CellPixels(image.Pt(12, 16))}, 4, 8},
	} {
		f := driveFrames(t, frames.Flag, append([]drive.Option{drive.Size(24, 12)}, c.opts...)...).Frame()
		t.Logf("%s:\n%s", c.name, f.Text())
		if top, flag := cells(f, c.border-1, 0, 2), cells(f, c.border-1, 1, 3); top != "▁ " || flag != "k▏ " {
			t.Errorf("%s: top row ends %q and the flag row %q at column %d, want \"▁ \" and \"k▏ \": the right border at column %d", c.name, top, flag, c.border-1, c.border)
		}
		rows := 0
		for y := range f.Cells().Height() {
			if f.Cells().At(0, y).Bg.RGBA != black {
				rows++
			}
		}
		t.Logf("%s: aspect-video w-20 fills %d rows", c.name, rows)
		if rows != c.rows {
			t.Errorf("%s: aspect-video w-20 is %d rows, want %d", c.name, rows, c.rows)
		}
	}
}

func TestMotionWakesAtItsPaceAndStops(t *testing.T) {
	sheet, err := frames.Styles()
	if err != nil {
		t.Fatal(err)
	}
	r := launch(newBackend(20, 3), frames.Pulse, twi.Styles(sheet), twi.ColorProfile(color.TrueColor))
	r.next(t)
	wakes, window := r.clock.wakes.Load(), 500*time.Millisecond
	count := 0
	for deadline := time.After(window); deadline != nil; {
		select {
		case <-r.b.frames:
			count++
		case <-deadline:
			deadline = nil
		}
	}
	woke, most := r.clock.wakes.Load()-wakes, int(window/konst.FrameInterval)+2
	t.Logf("pulsing: %d frames and %d wakes in %v, at most %d frames", count, woke, window, most)
	if count < most/4 || count > most || woke > int64(3*most) {
		t.Errorf("pulsing: %d frames and %d wakes in %v, want between %d and %d frames and at most %d wakes", count, woke, window, most/4, most, 3*most)
	}
	r.b.events <- key('s')
	for settled := false; !settled; {
		select {
		case <-r.b.frames:
		case <-time.After(100 * time.Millisecond):
			settled = true
		}
	}
	wakes = r.clock.wakes.Load()
	r.quiet(t)
	time.Sleep(200 * time.Millisecond)
	if woke := r.clock.wakes.Load() - wakes; woke != 0 {
		t.Errorf("stopped: the runtime woke %d times in 300ms", woke)
	}
	if err := r.stop(t); err != nil {
		t.Error(err)
	}
}

func TestClosingFrameSurvivesAFocusSignal(t *testing.T) {
	r := launch(newBackend(24, 3), frames.Closing)
	r.next(t)
	r.b.events <- key('o')
	if f := r.next(t); !strings.Contains(f, "open") {
		t.Fatalf("after o: %q, want the dialog open", f)
	}
	r.b.events <- input.KeyEvent{Key: input.KeyEscape}
	if f := r.next(t); !strings.Contains(f, "closing") || !strings.Contains(f, "twice") {
		t.Errorf("first frame after escape %q, want the closing state painted with the focus signal set in the same frame", f)
	}
	if f := r.next(t); strings.Contains(f, "closing") {
		t.Errorf("second frame after escape %q, want the closing text cleared", f)
	}
	r.quiet(t)
	if err := r.stop(t); err != nil {
		t.Error(err)
	}
}

func TestAtPlacesANodeAtACell(t *testing.T) {
	d := driveFrames(t, func(*twi.Runtime) func() twi.Node {
		return func() twi.Node {
			return twi.Element(twi.Class("relative h-full"), twi.Text("page"), twi.Element(twi.At(6, 2), twi.Text("menu")), twi.Text("rest"))
		}
	}, drive.Size(16, 4))
	t.Logf("menu at 6,2:\n%s", d.Frame().Text())
	if got, rest := cells(d.Frame(), 6, 2, 4), cells(d.Frame(), 0, 1, 4); got != "menu" || rest != "rest" {
		t.Errorf("cells at 6,2 are %q and the next row starts %q, want menu placed out of flow and rest right under page", got, rest)
	}
}

func TestPointerDownCarriesButtonAndCell(t *testing.T) {
	var got []input.MouseEvent
	d := driveFrames(t, func(*twi.Runtime) func() twi.Node {
		return func() twi.Node {
			return twi.Element(twi.Class("h-full"), twi.OnPointerDown(func(e *twi.Event) { got = append(got, e.Mouse) }), twi.Text("area"))
		}
	}, drive.Size(10, 3))
	d.ClickWith(input.MouseRight, 3, 1)
	d.Click(1, 0)
	want := []input.MouseEvent{{X: 3, Y: 1, Button: input.MouseRight}, {X: 1, Y: 0, Button: input.MouseLeft}}
	if !slices.Equal(got, want) {
		t.Errorf("pointer down events %+v, want %+v", got, want)
	}
}
