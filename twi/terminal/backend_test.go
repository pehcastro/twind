package terminal

import (
	"bytes"
	"errors"
	"image"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/input"
)

type fakeTTY struct {
	input     chan []byte
	resize    chan image.Point
	cancelled chan struct{}
	restored  int
	mu        sync.Mutex
	cells     image.Point
}

func (f *fakeTTY) read(p []byte, quiet bool) (int, bool, error) {
	var silence <-chan time.Time
	if quiet {
		silence = time.After(konst.EscapeTimeout)
	}
	select {
	case b := <-f.input:
		return copy(p, b), false, nil
	case c := <-f.resize:
		f.mu.Lock()
		f.cells = c
		f.mu.Unlock()
		return 0, true, nil
	case <-silence:
		return 0, false, errQuiet
	case <-f.cancelled:
		return 0, false, io.EOF
	}
}

func (f *fakeTTY) size() (width, height int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cells.X, f.cells.Y, nil
}
func (f *fakeTTY) cancel()        { close(f.cancelled) }
func (f *fakeTTY) restore() error { f.restored++; return nil }

type fakeTerminal struct {
	tty     *fakeTTY
	answers []string
	later   []string
	written bytes.Buffer
	broken  bool
}

func (t *fakeTerminal) Write(p []byte) (int, error) {
	if t.broken {
		return 0, errors.New("terminal gone")
	}
	t.written.Write(p)
	answers := t.later
	if bytes.Contains(p, []byte(konst.Queries)) {
		answers = t.answers
	} else if !bytes.Contains(p, []byte(konst.CellQuery)) {
		answers = nil
	}
	for _, a := range answers {
		t.tty.input <- []byte(a)
	}
	return len(p), nil
}

func newFake(answers ...string) *fakeTerminal {
	return &fakeTerminal{tty: &fakeTTY{
		input:     make(chan []byte, 16),
		resize:    make(chan image.Point),
		cancelled: make(chan struct{}),
		cells:     image.Pt(80, 24),
	}, answers: answers}
}

func TestStartupQueries(t *testing.T) {
	cases := []struct {
		name        string
		answers     []string
		sync, kitty bool
		fenced      bool
	}{
		{"sync set, kitty, fence", []string{"\x1b[?2026;1$y", "\x1b[?1u", "\x1b[?62;22c"}, true, true, true},
		{"sync reset is still supported", []string{"\x1b[?2026;2$y\x1b[?62c"}, true, false, true},
		{"sync unknown", []string{"\x1b[?2026;0$y\x1b[?62c"}, false, false, true},
		{"sync permanently reset", []string{"\x1b[?2026;4$y\x1b[?62c"}, false, false, true},
		{"another mode set", []string{"\x1b[?2004;1$y\x1b[?62c"}, false, false, true},
		{"fence alone, split over two reads", []string{"\x1b[?6", "2;22c"}, false, false, true},
		{"silent", nil, false, false, false},
	}
	const runs = 20
	for _, tc := range cases {
		var took time.Duration
		for range runs {
			term := newFake(tc.answers...)
			start := time.Now()
			b, err := enter(term, term.tty, Options{}, offer{})
			took += time.Since(start)
			if err != nil {
				t.Fatal(err)
			}
			if b.Capabilities != (Capabilities{Sync: tc.sync, KittyKeyboard: tc.kitty}) {
				t.Errorf("%s: %+v, want sync %v kitty %v", tc.name, b.Capabilities, tc.sync, tc.kitty)
			}
			if got := strings.Contains(term.written.String(), konst.KittyPush); got != tc.kitty {
				t.Errorf("%s: kitty push written %v", tc.name, got)
			}
			if err := b.Exit(); err != nil {
				t.Fatal(err)
			}
		}
		took /= runs
		t.Logf("%s: %v mean over %d runs", tc.name, took, runs)
		switch {
		case tc.fenced && took >= 5*time.Millisecond:
			t.Errorf("%s: took %v after the fence, want under 5ms", tc.name, took)
		case !tc.fenced && (took < konst.QueryTimeout || took > 2*konst.QueryTimeout):
			t.Errorf("%s: took %v, want the %v timeout", tc.name, took, konst.QueryTimeout)
		}
	}
}

var (
	windowsTerminal = []string{"\x1b[6;20;10t", "\x1b[4;480;800t\x1b[?2026;2$y\x1b[?61;4;6;7;14;21;22;23;24;28;32;42;52c"}
	kitty           = []string{"\x1b_Gi=31;OK\x1b\\", "\x1b[?2026;2$y\x1b[?0u\x1b[6;36;17t\x1b[4;864;1360t", "\x1b[?62;c"}
	conPTY          = []string{"\x1b[?1;0c"}
)

func TestGraphics(t *testing.T) {
	cases := []struct {
		name     string
		answers  []string
		offer    offer
		graphics Graphics
		cell     image.Point
		fenced   bool
	}{
		{"windows terminal 1.24", windowsTerminal, offer{}, GraphicsSixel, image.Pt(10, 20), true},
		{"kitty", kitty, offer{}, GraphicsKitty, image.Pt(17, 36), true},
		{"tui-test conpty", conPTY, offer{}, GraphicsNone, image.Point{}, true},
		{"silent", nil, offer{}, GraphicsNone, image.Point{}, false},
		{"kitty outranks sixel", []string{"\x1b_Gi=31;OK\x1b\\\x1b[6;20;10t\x1b[?62;4c"}, offer{}, GraphicsKitty, image.Pt(10, 20), true},
		{"window pixels only", []string{"\x1b[4;480;800t\x1b[?61;4c"}, offer{}, GraphicsSixel, image.Pt(10, 20), true},
		{"cell report split over reads", []string{"\x1b[6;2", "0;10t\x1b[?61;4c"}, offer{}, GraphicsSixel, image.Pt(10, 20), true},
		{"zero sizes are unknown", []string{"\x1b[6;0;0t\x1b[4;0;0t\x1b[?61;4c"}, offer{}, GraphicsNone, image.Point{}, true},
		{"window smaller than the grid", []string{"\x1b[4;480;40t\x1b[?61;4c"}, offer{}, GraphicsNone, image.Point{}, true},
		{"sixel without a size", []string{"\x1b[?61;4c"}, offer{}, GraphicsNone, image.Point{}, true},
		{"kitty error status", []string{"\x1b_Gi=31;ENOTSUPPORTED:no\x1b\\\x1b[6;20;10t\x1b[?62;c"}, offer{}, GraphicsNone, image.Pt(10, 20), true},
		{"kitty reply for another id", []string{"\x1b_Gi=7;OK\x1b\\\x1b[6;20;10t\x1b[?62;c"}, offer{}, GraphicsNone, image.Pt(10, 20), true},
		{"4 as the class is not sixel", []string{"\x1b[6;20;10t\x1b[?4c"}, offer{}, GraphicsNone, image.Pt(10, 20), true},
		{"iterm2 offered outranks sixel", windowsTerminal, offer{graphics: GraphicsITerm2}, GraphicsITerm2, image.Pt(10, 20), true},
		{"kitty outranks iterm2 offered", kitty, offer{graphics: GraphicsITerm2}, GraphicsKitty, image.Pt(17, 36), true},
		{"forced none", windowsTerminal, offer{GraphicsNone, true}, GraphicsNone, image.Pt(10, 20), true},
		{"forced kitty", windowsTerminal, offer{GraphicsKitty, true}, GraphicsKitty, image.Pt(10, 20), true},
		{"forced sixel, silent", nil, offer{GraphicsSixel, true}, GraphicsNone, image.Point{}, false},
	}
	for _, tc := range cases {
		term := newFake(tc.answers...)
		start := time.Now()
		b, err := enter(term, term.tty, Options{}, tc.offer)
		took := time.Since(start)
		if err != nil {
			t.Fatal(err)
		}
		if b.Capabilities.Graphics != tc.graphics || b.Capabilities.CellPixels != tc.cell {
			t.Errorf("%s: graphics %d cell %v, want %d %v", tc.name, b.Capabilities.Graphics, b.Capabilities.CellPixels, tc.graphics, tc.cell)
		}
		if tc.fenced != (took < konst.QueryTimeout) {
			t.Errorf("%s: took %v, fenced %v", tc.name, took, tc.fenced)
		}
		term.tty.input <- []byte("x")
		select {
		case ev := <-b.Events:
			if ev != (input.KeyEvent{Rune: 'x'}) {
				t.Errorf("%s: first event %#v, want the key typed after the queries", tc.name, ev)
			}
		case <-time.After(time.Second):
			t.Errorf("%s: no key event", tc.name)
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOffered(t *testing.T) {
	cases := []struct {
		env   map[string]string
		offer offer
		err   bool
	}{
		{map[string]string{}, offer{}, false},
		{map[string]string{"TERM_PROGRAM": "zed", "WT_SESSION": "x"}, offer{}, false},
		{map[string]string{"TERM_PROGRAM": "iTerm.app"}, offer{graphics: GraphicsITerm2}, false},
		{map[string]string{"TERM_PROGRAM": "WezTerm"}, offer{graphics: GraphicsITerm2}, false},
		{map[string]string{"LC_TERMINAL": "iTerm2"}, offer{graphics: GraphicsITerm2}, false},
		{map[string]string{"TWIND_GRAPHICS": "none", "TERM_PROGRAM": "iTerm.app"}, offer{GraphicsNone, true}, false},
		{map[string]string{"TWIND_GRAPHICS": "sixel"}, offer{GraphicsSixel, true}, false},
		{map[string]string{"TWIND_GRAPHICS": "kitty"}, offer{GraphicsKitty, true}, false},
		{map[string]string{"TWIND_GRAPHICS": "iterm2"}, offer{GraphicsITerm2, true}, false},
		{map[string]string{"TWIND_GRAPHICS": "Sixel"}, offer{}, true},
		{map[string]string{"TWIND_GRAPHICS": "regis"}, offer{}, true},
	}
	for _, tc := range cases {
		got, err := offered(func(k string) string { return tc.env[k] })
		if got != tc.offer || (err != nil) != tc.err {
			t.Errorf("%v: %+v, %v; want %+v, error %v", tc.env, got, err, tc.offer, tc.err)
		}
	}
}

func TestCellPixelsAfterResize(t *testing.T) {
	cases := []struct {
		name    string
		answers []string
		later   []string
		cell    image.Point
		asked   bool
	}{
		{"font zoom in windows terminal", windowsTerminal, []string{"\x1b[6;24;12t\x1b[4;720;1200t\x1b[?61;4c"}, image.Pt(12, 24), true},
		{"window pixels only after resize", windowsTerminal, []string{"\x1b[4;720;1200t\x1b[?61;4c"}, image.Pt(12, 24), true},
		{"no answer keeps the last size", windowsTerminal, []string{"\x1b[?61;4c"}, image.Pt(10, 20), true},
		{"no graphics, nothing asked", conPTY, []string{"\x1b[6;24;12t\x1b[?1;0c"}, image.Point{}, false},
	}
	for _, tc := range cases {
		term := newFake(tc.answers...)
		term.later = tc.later
		b, err := enter(term, term.tty, Options{}, offer{})
		if err != nil {
			t.Fatal(err)
		}
		term.tty.resize <- image.Pt(100, 30)
		select {
		case ev := <-b.Events:
			if ev != (input.ResizeEvent{Width: 100, Height: 30}) {
				t.Fatalf("%s: event %#v, want the resize", tc.name, ev)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s: no resize event", tc.name)
		}
		before := term.written.Len()
		start := time.Now()
		if _, err := b.Write([]byte("frame")); err != nil {
			t.Fatal(err)
		}
		took := time.Since(start)
		out := term.written.String()[before:]
		if b.Capabilities.CellPixels != tc.cell {
			t.Errorf("%s: cell %v after resize, want %v", tc.name, b.Capabilities.CellPixels, tc.cell)
		}
		if asked := strings.HasPrefix(out, konst.CellQuery); asked != tc.asked || !strings.HasSuffix(out, "frame") {
			t.Errorf("%s: wrote %q, want the cell query %v before the frame", tc.name, out, tc.asked)
		}
		if took >= konst.QueryTimeout {
			t.Errorf("%s: frame write took %v, want the fence", tc.name, took)
		}
		before = term.written.Len()
		if _, err := b.Write([]byte("frame")); err != nil {
			t.Fatal(err)
		}
		if out := term.written.String()[before:]; out != "frame" {
			t.Errorf("%s: second frame wrote %q, want the frame alone", tc.name, out)
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEvents(t *testing.T) {
	term := newFake("\x1b[?2026;1$y", "x", "\x1b[?62c")
	b, err := enter(term, term.tty, Options{Mouse: true}, offer{})
	if err != nil {
		t.Fatal(err)
	}
	if !b.Capabilities.Sync {
		t.Fatal("sync not detected")
	}
	term.tty.input <- []byte("\x1b[?2026;1$y")
	term.tty.input <- []byte("\x1b")
	want := []input.Event{input.KeyEvent{Rune: 'x'}, input.KeyEvent{Key: input.KeyEscape}}
	for _, w := range want {
		select {
		case ev := <-b.Events:
			if ev != w {
				t.Errorf("event %#v, want %#v", ev, w)
			}
		case <-time.After(time.Second):
			t.Fatalf("no event, want %#v", w)
		}
	}
	if err := b.Exit(); err != nil {
		t.Fatal(err)
	}
	for ev := range b.Events {
		t.Errorf("event %#v after exit", ev)
	}
	written := term.written.Len()
	if err := b.Exit(); err != nil {
		t.Fatal(err)
	}
	out := term.written.String()
	if term.written.Len() != written || term.tty.restored != 1 {
		t.Errorf("second exit wrote %q, restored %d times", out[written:], term.tty.restored)
	}
	if !strings.HasSuffix(out, konst.MouseOff+konst.LeaveScreen) || !strings.Contains(out, konst.MouseOn) {
		t.Errorf("mouse on and off missing from %q", out)
	}
}

func TestWriteFailureRestores(t *testing.T) {
	term := newFake("\x1b[?62c")
	b, err := enter(term, term.tty, Options{}, offer{})
	if err != nil {
		t.Fatal(err)
	}
	term.broken = true
	if _, err := b.Write([]byte("frame")); err == nil {
		t.Error("write to a broken terminal returned no error")
	}
	if term.tty.restored != 1 {
		t.Errorf("restored %d times after a failed write, want 1", term.tty.restored)
	}
}
