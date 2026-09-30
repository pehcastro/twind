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

func pulsing(t *testing.T, b *backend) run {
	sheet, err := frames.Styles()
	if err != nil {
		t.Fatal(err)
	}
	r := launch(b, frames.Pulse, twi.Styles(sheet), twi.ColorProfile(color.TrueColor))
	r.next(t)
	return r
}

func (r run) count(window time.Duration) int {
	count := 0
	for deadline := time.After(window); deadline != nil; {
		select {
		case <-r.b.frames:
			count++
		case <-deadline:
			deadline = nil
		}
	}
	return count
}

func TestMotionWakesAtItsPaceAndStops(t *testing.T) {
	r := pulsing(t, newBackend(20, 3))
	wakes, window := r.clock.wakes.Load(), 500*time.Millisecond
	count := r.count(window)
	woke, most := r.clock.wakes.Load()-wakes, int(window/konst.MotionInterval)+2
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

func TestSlowTerminalKeepsTheMotionPace(t *testing.T) {
	b := newBackend(20, 3)
	b.blocked = 20 * time.Millisecond
	r := pulsing(t, b)
	window := 600 * time.Millisecond
	count, paced := r.count(window), int(window/konst.MotionInterval)
	t.Logf("pulsing through a terminal that blocks each write %v: %d frames in %v, %d at the motion pace", b.blocked, count, window, paced)
	if count < paced-3 || count > paced+2 {
		t.Errorf("%d frames in %v with each write blocked %v, want %d to %d: the pace counts from the frame's start, not its end", count, window, b.blocked, paced-3, paced+2)
	}
	if err := r.stop(t); err != nil {
		t.Error(err)
	}
}

func TestFadingPanelCoversThePageText(t *testing.T) {
	d := driveFrames(t, func(*twi.Runtime) func() twi.Node { return frames.Fading }, drive.Size(24, 6))
	inside := func(f drive.Frame) (page, panel string) {
		for y := 1; y < 4; y++ {
			for x := 2; x < 14; x++ {
				if g := f.Cells().At(x, y).Grapheme; y == 1 && x < 7 {
					panel += g
				} else {
					page += strings.TrimSpace(g)
				}
			}
		}
		return page, panel
	}
	t.Logf("fade starts:\n%s", d.Frame().Text())
	if page, _ := inside(d.Frame()); page == "" {
		t.Errorf("the panel at opacity 0 hides the page text under it")
	}
	d.Advance(100 * time.Millisecond)
	t.Logf("mid-fade:\n%s", d.Frame().Text())
	if page, panel := inside(d.Frame()); page != "" || panel != "panel" {
		t.Errorf("mid-fade the panel's box shows %q from the page and %q where its own text is, want no page glyph and panel", page, panel)
	}
}

func TestPlacesReachHoverAndSelection(t *testing.T) {
	d := driveFrames(t, func(*twi.Runtime) func() twi.Node { return frames.Placed }, drive.Size(30, 4))
	at := func(word string) (int, int) {
		for y, line := range strings.Split(d.Frame().Text(), "\n") {
			if x := strings.Index(line, word); x >= 0 {
				return x, y
			}
		}
		t.Fatalf("no %q in the frame:\n%s", word, d.Frame().Text())
		return 0, 0
	}
	bg := func(word string) color.RGBA { return d.Frame().Cells().At(at(word)).Bg.RGBA }
	rest := bg("three")
	d.Move(at("one"))
	if got := bg("one"); got != rest {
		t.Errorf("hovering the first item turned it %v, want %v: last:hover: holds only on the last", got, rest)
	}
	d.Move(at("three"))
	if got := bg("three"); got == rest {
		t.Errorf("hovering the last item left it %v: last:hover: never restyled it", got)
	}
	d.Down(0, 1)
	d.Move(18, 1)
	d.Up(18, 1)
	d.Press("ctrl+c")
	if got := d.Clipboard(); got != "alpha bravo" {
		t.Errorf("a drag over alpha bravo charlie copied %q, want %q: charlie is last:select-none", got, "alpha bravo")
	}
}

func TestCopyWritesTheClipboard(t *testing.T) {
	app := func(rt *twi.Runtime) func() twi.Node {
		return func() twi.Node {
			return twi.Element(twi.OnKey(func(k input.KeyEvent) {
				if k.Rune == 'y' {
					if err := rt.Copy("from the app"); err != nil {
						t.Error(err)
					}
				}
			}), twi.Text("copy"))
		}
	}
	d := driveFrames(t, app, drive.Size(10, 1))
	d.Press("y")
	if got := d.Clipboard(); got != "from the app" {
		t.Errorf("rt.Copy wrote %q to the clipboard, want %q", got, "from the app")
	}
	r := launch(newBackend(10, 1), app, twi.NoClipboard())
	r.next(t)
	r.b.events <- key('y')
	r.quiet(t)
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
