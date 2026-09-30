package runtime_test

import (
	"strings"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/runtime/testdata/selection"
)

type selecting struct {
	t *testing.T
	d *drive.Driver
}

var cardBg = color.RGBA{R: 24, G: 24, B: 27, A: 255}

func startSelecting(t *testing.T) *selecting {
	sheet, err := selection.Styles()
	if err != nil {
		t.Fatal(err)
	}
	s := &selecting{t: t, d: drive.New(selection.App, drive.Size(60, 12), drive.Styles(sheet))}
	t.Cleanup(func() {
		if err := s.d.Err(); err != nil {
			t.Error(err)
		}
		if err := s.d.Close(); err != nil {
			t.Error(err)
		}
	})
	return s
}

func (s *selecting) at(word string, nth int) (int, int) {
	s.t.Helper()
	for y, line := range strings.Split(s.d.Frame().Text(), "\n") {
		if x := strings.Index(line, word); x >= 0 {
			return len([]rune(line[:x])) + nth, y
		}
	}
	s.t.Fatalf("no %q in the frame:\n%s", word, s.d.Frame().Text())
	return 0, 0
}

func (s *selecting) lit(word string) []bool {
	s.t.Helper()
	x, y := s.at(word, 0)
	cells := s.d.Frame().Cells()
	lit := make([]bool, len(word))
	for i := range word {
		c := cells.At(x+i, y)
		if c.Grapheme != word[i:i+1] || c.Fg.RGBA != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) && word != "nosel" {
			s.t.Errorf("%q cell %d is %q fg %v: the highlight must keep the glyph and its colour", word, i, c.Grapheme, c.Fg.RGBA)
		}
		lit[i] = c.Bg.RGBA != cardBg
	}
	return lit
}

func (s *selecting) highlighted(want map[string]string) {
	s.t.Helper()
	for word, marks := range want {
		got := ""
		for _, on := range s.lit(word) {
			got += map[bool]string{true: "#", false: "."}[on]
		}
		if got != marks {
			s.t.Errorf("%q highlighted %s, want %s:\n%s", word, got, marks, s.d.Frame().ANSI())
		}
	}
}

func (s *selecting) copied(want string) {
	s.t.Helper()
	s.d.Press("ctrl+c")
	if got := s.d.Clipboard(); got != want {
		s.t.Errorf("ctrl+c copied %q, want %q", got, want)
	}
}

func TestSelectionStaysInItsCard(t *testing.T) {
	s := startSelecting(t)
	x, y := s.at("bravo", 2)
	s.d.Down(x, y)
	mx, my := s.at("mike", 2)
	s.d.Move(mx, my)
	s.d.Up(mx, my)
	s.highlighted(map[string]string{
		"alpha": ".....", "bravo": "..###", "charlie": "#######", "delta": "#####", "foxtrot": "#######",
		"golf": "....", "nosel": ".....", "whole": ".....", "india": ".....", "lima": "....", "mike": "....",
	})
	s.copied("avo charlie delta echo foxtrot")
	s.d.Press("escape")
	if err := s.d.Err(); err != nil {
		t.Errorf("ctrl+c with a selection quit the app: %v", err)
	}
}

func TestSelectionBackwardsKeepsTheAnchor(t *testing.T) {
	s := startSelecting(t)
	x, y := s.at("echo", 1)
	s.d.Down(x, y)
	bx, by := s.at("bravo", 3)
	s.d.Move(bx, by)
	s.d.Up(bx, by)
	s.highlighted(map[string]string{"bravo": "...##", "charlie": "#######", "delta": "#####", "echo": "##..", "foxtrot": "......."})
	s.copied("vo charlie delta ec")
}

func TestSelectionClickSelectsNothingAndClears(t *testing.T) {
	s := startSelecting(t)
	x, y := s.at("bravo", 0)
	s.d.Down(x, y)
	s.d.Move(x+3, y)
	s.d.Up(x+3, y)
	s.highlighted(map[string]string{"bravo": "####."})
	lx, ly := s.at("lima", 1)
	s.d.Click(lx, ly)
	s.highlighted(map[string]string{"bravo": ".....", "lima": "...."})
	s.d.Down(x, y)
	s.d.Move(x+1, y)
	s.d.Up(x+1, y)
	s.highlighted(map[string]string{"bravo": "##..."})
	s.d.Press("escape")
	s.highlighted(map[string]string{"bravo": "....."})
}

func TestSelectionSkipsSelectNone(t *testing.T) {
	s := startSelecting(t)
	x, y := s.at("nosel", 1)
	s.d.Down(x, y)
	hx, hy := s.at("hotel", 4)
	s.d.Move(hx, hy)
	s.d.Up(hx, hy)
	s.highlighted(map[string]string{"nosel": ".....", "hotel": "....."})
	gx, gy := s.at("golf", 1)
	s.d.Down(gx, gy)
	s.d.Move(hx, hy)
	s.d.Up(hx, hy)
	s.highlighted(map[string]string{"golf": ".###", "nosel": ".....", "hotel": "#####"})
	s.copied("olf hotel")
}

func TestSelectionWideGlyphsAndTruncation(t *testing.T) {
	s := startSelecting(t)
	x, y := s.at("中文", 0)
	s.d.Down(x+1, y)
	s.d.Move(x+25, y)
	s.d.Up(x+25, y)
	cells := s.d.Frame().Cells()
	for i := range 20 {
		c := cells.At(x+i, y)
		if lit := c.Bg.RGBA != cardBg; lit != (i < 19) {
			t.Errorf("cell %d %q lit %v, want the 19 cells of the visible text lit and the ellipsis not:\n%s", i, c.Grapheme, lit, s.d.Frame().ANSI())
		}
	}
	s.copied("中文 wide and trunc")
}

func TestSelectionIdleAndNoClipboard(t *testing.T) {
	sheet, err := selection.Styles()
	if err != nil {
		t.Fatal(err)
	}
	r := launch(newBackend(60, 12), selection.App, twi.Styles(sheet), twi.NoClipboard())
	r.next(t)
	for _, ev := range []input.MouseEvent{
		{X: 3, Y: 2, Button: input.MouseLeft, Action: input.MousePress},
		{X: 7, Y: 2, Button: input.MouseLeft, Action: input.MouseMove},
		{X: 7, Y: 2, Button: input.MouseLeft, Action: input.MouseRelease},
	} {
		r.b.events <- ev
	}
	for settled := false; !settled; {
		select {
		case <-r.b.frames:
		case <-time.After(100 * time.Millisecond):
			settled = true
		}
	}
	wakes := r.clock.wakes.Load()
	time.Sleep(300 * time.Millisecond)
	if woke := r.clock.wakes.Load() - wakes; woke != 0 {
		t.Errorf("idle with a selection shown, the runtime woke %d times", woke)
	}
	r.b.events <- input.KeyEvent{Key: input.KeyRune, Rune: 'c', Modifiers: input.ModCtrl}
	r.quiet(t)
	select {
	case err := <-r.done:
		t.Fatalf("ctrl+c with a selection and the clipboard off quit the app: %v", err)
	default:
	}
	if err := r.stop(t); err != nil {
		t.Error(err)
	}
}

func TestSelectionWordLineAndAll(t *testing.T) {
	s := startSelecting(t)
	x, y := s.at("juliet", 2)
	s.d.Click(x, y)
	s.d.Click(x, y)
	s.highlighted(map[string]string{"india": ".....", "juliet": "######", "kilo": "...."})
	s.copied("juliet")
	s.d.Click(x, y)
	s.d.Click(x, y)
	s.d.Click(x, y)
	s.highlighted(map[string]string{"india": "#####", "juliet": "######", "kilo": "####", "lima": "...."})
	s.copied("india juliet kilo")
	wx, wy := s.at("whole", 2)
	s.d.Click(wx+20, wy+3)
	s.d.Click(wx, wy)
	s.highlighted(map[string]string{"whole": "#####", "thing": "#####", "golf": "...."})
	s.copied("whole thing")
}
