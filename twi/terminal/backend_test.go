package terminal

import (
	"bytes"
	"errors"
	"image"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/text"
)

type fakeTTY struct {
	input     chan []byte
	resize    chan image.Point
	cancelled chan struct{}
	restored  int
	mu        sync.Mutex
	cells     image.Point
	reads     int
}

func (f *fakeTTY) read(p []byte, wait time.Duration) (int, bool, error) {
	f.mu.Lock()
	f.reads++
	f.mu.Unlock()
	var silence <-chan time.Time
	if wait > 0 {
		silence = time.After(wait)
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
	reply   func(p []byte) []string
	delay   time.Duration
	mu      sync.Mutex
	written bytes.Buffer
	broken  bool
}

func (t *fakeTerminal) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.broken {
		return 0, errors.New("terminal gone")
	}
	t.written.Write(p)
	answers := t.answers
	switch {
	case t.reply != nil:
		answers = t.reply(p)
	case bytes.Contains(p, []byte(konst.Queries)) || bytes.Contains(p, []byte(konst.InlineQueries)):
	case bytes.Contains(p, []byte(konst.CellQuery)):
		answers = t.later
	case !bytes.HasSuffix(p, []byte(konst.Fence)):
		answers = nil
	}
	send := func() {
		for _, a := range answers {
			t.tty.input <- []byte(a)
		}
	}
	if t.delay > 0 && len(answers) > 0 {
		go func() {
			time.Sleep(t.delay)
			send()
		}()
		return len(p), nil
	}
	send()
	return len(p), nil
}

func (t *fakeTerminal) out() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.written.String()
}

func events(t *testing.T, b *Backend, name string, want ...input.Event) {
	t.Helper()
	for _, w := range want {
		select {
		case ev := <-b.Events:
			if ev != w {
				t.Errorf("%s: event %#v, want %#v", name, ev, w)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s: no event, want %#v", name, w)
		}
	}
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
	for _, tc := range cases {
		var took time.Duration
		runs := 20
		if !tc.fenced {
			runs = 1
		}
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
		took /= time.Duration(runs)
		t.Logf("%s: %v mean over %d runs", tc.name, took, runs)
		switch {
		case tc.fenced && took >= 5*time.Millisecond:
			t.Errorf("%s: took %v after the fence, want under 5ms", tc.name, took)
		case !tc.fenced && (took < konst.StartupTimeout || took > konst.StartupTimeout+konst.QueryTimeout):
			t.Errorf("%s: took %v, want the %v startup wait", tc.name, took, konst.StartupTimeout)
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

func TestCellSizeAfterResize(t *testing.T) {
	resized := input.ResizeEvent{Width: 100, Height: 30}
	zoomed := input.ResizeEvent{Width: 100, Height: 30, Cell: image.Pt(12, 24)}
	key := input.KeyEvent{Rune: 'x'}
	cases := []struct {
		name    string
		answers []string
		later   []string
		cell    image.Point
		want    []input.Event
	}{
		{"font zoom", windowsTerminal, []string{"\x1b[6;24;12t\x1b[4;720;1200t\x1b[?61;4c", "x"}, image.Pt(12, 24), []input.Event{resized, zoomed, key}},
		{"window pixels only", windowsTerminal, []string{"\x1b[4;720;1200t\x1b[?61;4c", "x"}, image.Pt(12, 24), []input.Event{resized, zoomed, key}},
		{"reply split over reads", windowsTerminal, []string{"\x1b[6;2", "4;12t\x1b[?6", "1;4c", "x"}, image.Pt(12, 24), []input.Event{resized, zoomed, key}},
		{"same cell sends one event", windowsTerminal, []string{"\x1b[6;20;10t\x1b[4;600;1000t\x1b[?61;4c", "x"}, image.Pt(10, 20), []input.Event{resized, key}},
		{"no answer keeps the last size", windowsTerminal, []string{"\x1b[?61;4c", "x"}, image.Pt(10, 20), []input.Event{resized, key}},
		{"zero sizes keep the last size", windowsTerminal, []string{"\x1b[6;0;0t\x1b[?61;4c", "x"}, image.Pt(10, 20), []input.Event{resized, key}},
		{"no graphics, nothing asked", conPTY, []string{"\x1b[6;24;12t\x1b[?1;0c"}, image.Point{}, []input.Event{resized, key}},
	}
	for _, tc := range cases {
		term := newFake(tc.answers...)
		term.later = tc.later
		b, err := enter(term, term.tty, Options{}, offer{})
		if err != nil {
			t.Fatal(err)
		}
		before := len(term.out())
		term.tty.resize <- image.Pt(100, 30)
		query := konst.CellQuery
		if tc.cell == (image.Point{}) {
			query = ""
			term.tty.input <- []byte("x")
		}
		events(t, b, tc.name, tc.want...)
		if out := term.out()[before:]; out != query {
			t.Errorf("%s: the reader wrote %q, want %q", tc.name, out, query)
		}
		before = len(term.out())
		if _, err := b.Write([]byte("frame")); err != nil {
			t.Fatal(err)
		}
		if out := term.out()[before:]; out != "frame" {
			t.Errorf("%s: wrote %q, want the frame alone", tc.name, out)
		}
		if b.Capabilities.CellPixels != tc.cell {
			t.Errorf("%s: cell %v after the frame, want %v", tc.name, b.Capabilities.CellPixels, tc.cell)
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCellSizeOnInput(t *testing.T) {
	zoomed := input.ResizeEvent{Width: 80, Height: 24, Cell: image.Pt(12, 24)}
	key := input.KeyEvent{Rune: 'x'}
	for _, tc := range []struct {
		name  string
		pause time.Duration
		in    string
		want  []input.Event
		asked bool
	}{
		{"key after a pause", konst.CellPoll, "x", []input.Event{key, zoomed}, true},
		{"key right after the startup queries", 0, "x", []input.Event{key}, false},
		{"focus in", 0, "\x1b[I", []input.Event{input.FocusEvent{Focused: true}, zoomed}, true},
		{"focus out after a pause", konst.CellPoll, "\x1b[O", []input.Event{input.FocusEvent{}}, false},
	} {
		term := newFake(windowsTerminal...)
		term.later = []string{"\x1b[6;24;12t\x1b[?61;4c"}
		b, err := enter(term, term.tty, Options{}, offer{})
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(tc.pause)
		before := len(term.out())
		term.tty.input <- []byte(tc.in)
		events(t, b, tc.name, tc.want...)
		query := ""
		if tc.asked {
			query = konst.CellQuery
		}
		if out := term.out()[before:]; out != query {
			t.Errorf("%s: wrote %q, want %q", tc.name, out, query)
		}
		if tc.asked {
			term.tty.input <- []byte("y")
			events(t, b, tc.name, input.KeyEvent{Rune: 'y'})
			if out := term.out()[before:]; out != query {
				t.Errorf("%s: a key within %v of the query wrote %q", tc.name, konst.CellPoll, out[len(query):])
			}
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCellSizeIdle(t *testing.T) {
	term := newFake(windowsTerminal...)
	term.later = []string{"\x1b[6;24;12t\x1b[?61;4c"}
	b, err := enter(term, term.tty, Options{}, offer{})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(konst.EscapeTimeout * 2)
	term.tty.mu.Lock()
	reads := term.tty.reads
	term.tty.mu.Unlock()
	written := len(term.out())
	const idle = 10 * time.Second
	time.Sleep(idle)
	term.tty.mu.Lock()
	woke := term.tty.reads - reads
	term.tty.mu.Unlock()
	t.Logf("idle %v: %d wakes, %d bytes written", idle, woke, len(term.out())-written)
	if woke != 0 || len(term.out()) != written {
		t.Errorf("idle %v: %d wakes, wrote %q", idle, woke, term.out()[written:])
	}
	if err := b.Exit(); err != nil {
		t.Fatal(err)
	}
}

func TestCellSizeExitWaitsForTheReply(t *testing.T) {
	term := newFake(windowsTerminal...)
	term.later = []string{"\x1b[6;24;12t\x1b[?61;4c"}
	term.delay = konst.QueryTimeout / 4
	b, err := enter(term, term.tty, Options{}, offer{})
	if err != nil {
		t.Fatal(err)
	}
	term.tty.resize <- image.Pt(100, 30)
	events(t, b, "resize", input.ResizeEvent{Width: 100, Height: 30})
	start := time.Now()
	if err := b.Exit(); err != nil {
		t.Fatal(err)
	}
	took := time.Since(start)
	time.Sleep(2 * term.delay)
	if n := len(term.tty.input); n != 0 || took > konst.QueryTimeout {
		t.Errorf("exit took %v and left %d replies for the shell", took, n)
	}
	if out := term.out(); !strings.HasSuffix(out, konst.LeaveScreen) {
		t.Errorf("wrote %q, want the leave sequence last", out)
	}
}

func TestInBandResize(t *testing.T) {
	report := "\x1b[48;34;120;816;1440t"
	for _, tc := range []struct {
		name  string
		state string
		on    bool
	}{
		{"reset", "2", true},
		{"set", "1", true},
		{"not recognised", "0", false},
		{"permanently reset", "4", false},
	} {
		answers := slices.Clone(windowsTerminal)
		answers[1] = "\x1b[?2048;" + tc.state + "$y" + answers[1]
		term := newFake(answers...)
		term.later = []string{"\x1b[6;24;12t\x1b[?61;4c"}
		b, err := enter(term, term.tty, Options{}, offer{})
		if err != nil {
			t.Fatal(err)
		}
		if b.Capabilities.InBandResize != tc.on || strings.Contains(term.out(), konst.InBandOn) != tc.on {
			t.Errorf("%s: in-band %v, wrote %q", tc.name, b.Capabilities.InBandResize, term.out())
		}
		term.tty.input <- []byte(report)
		events(t, b, tc.name, input.ResizeEvent{Width: 120, Height: 34, Cell: image.Pt(12, 24)})
		if _, err := b.Write(nil); err != nil {
			t.Fatal(err)
		}
		if b.Capabilities.CellPixels != image.Pt(12, 24) {
			t.Errorf("%s: cell %v after the report", tc.name, b.Capabilities.CellPixels)
		}
		if tc.on {
			before := len(term.out())
			time.Sleep(konst.CellPoll + konst.QueryTimeout)
			if out := term.out()[before:]; out != "" {
				t.Errorf("%s: wrote %q with in-band reports on, want no poll", tc.name, out)
			}
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(term.out(), konst.InBandOff) != tc.on {
			t.Errorf("%s: exit wrote %q", tc.name, term.out())
		}
	}
}

func TestEvents(t *testing.T) {
	term := newFake("\x1b[?2026;1$y", "x", "\x1b[?62c")
	b, err := enter(term, term.tty, Options{}, offer{})
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

func TestMouseCaptureIsTheDefault(t *testing.T) {
	for _, tc := range []struct {
		name  string
		opt   Options
		mouse bool
	}{{"zero options", Options{}, true}, {"no mouse", Options{NoMouse: true}, false}} {
		term := newFake("\x1b[?62c")
		b, err := enter(term, term.tty, tc.opt, offer{})
		if err != nil {
			t.Fatal(err)
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
		out := term.written.String()
		if on, off := strings.Contains(out, konst.MouseOn), strings.HasSuffix(out, konst.MouseOff+konst.LeaveScreen); on != tc.mouse || off != tc.mouse {
			t.Errorf("%s: mouse on %v, off before leaving %v, want %v", tc.name, on, off, tc.mouse)
		}
	}
	term := newFake("\x1b[1;1R\x1b[?62c")
	if _, _, err := query(term, term.tty, offer{}); err != nil {
		t.Fatal(err)
	}
	if out := term.written.String(); strings.Contains(out, konst.MouseOn) || strings.Contains(out, konst.MouseOff) {
		t.Errorf("the inline query touched mouse reporting: %q", out)
	}
}

func TestMargins(t *testing.T) {
	for _, tc := range []struct {
		name    string
		answers []string
		margins bool
	}{
		{"windows terminal answers reset", []string{"\x1b[?69;2$y\x1b[?61;4c"}, true},
		{"set", []string{"\x1b[?69;1$y\x1b[?62c"}, true},
		{"unknown", []string{"\x1b[?69;0$y\x1b[?62c"}, false},
		{"permanently reset", []string{"\x1b[?69;4$y\x1b[?62c"}, false},
		{"another mode set", []string{"\x1b[?2026;1$y\x1b[?62c"}, false},
		{"split over reads", []string{"\x1b[?6", "9;2$y\x1b[?62c"}, true},
		{"no answer", []string{"\x1b[?62c"}, false},
	} {
		term := newFake(tc.answers...)
		b, err := enter(term, term.tty, Options{}, offer{})
		if err != nil {
			t.Fatal(err)
		}
		if b.Capabilities.Margins != tc.margins {
			t.Errorf("%s: margins %v, want %v", tc.name, b.Capabilities.Margins, tc.margins)
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
		if out := term.written.String(); !strings.Contains(out, konst.MarginsQuery) || strings.Index(out, konst.MarginsQuery) > strings.LastIndex(out, "\x1b[c") {
			t.Errorf("%s: the ?69 query is missing or after the fence in %q", tc.name, out)
		}
	}
}

func TestInlineQuery(t *testing.T) {
	cursorAfterWT := []string{windowsTerminal[0], "\x1b[4;480;800t\x1b[?2026;2$y\x1b[7;3R\x1b[?61;4;6;7;14;21;22;23;24;28;32;42;52c"}
	cases := []struct {
		name    string
		answers []string
		offer   offer
		caps    Capabilities
		cursor  image.Point
		fenced  bool
	}{
		{"windows terminal 1.24", cursorAfterWT, offer{}, Capabilities{Sync: true, Graphics: GraphicsSixel, CellPixels: image.Pt(10, 20)}, image.Pt(2, 6), true},
		{"windows terminal probes after text", []string{"\x1b[1;23R\x1b[1;25R\x1b[1;25R\x1b[1;25R\x1b[1;25R\x1b[1;25R\x1b[?61;4;6;7;14;21;22;23;24;28;32;42;52c"}, offer{}, Capabilities{Widths: text.Widths{2, 2, 2, 2, 2}}, image.Pt(22, 0), true},
		{"forced kitty", cursorAfterWT, offer{GraphicsKitty, true}, Capabilities{Sync: true, Graphics: GraphicsKitty, CellPixels: image.Pt(10, 20)}, image.Pt(2, 6), true},
		{"no cursor report", windowsTerminal, offer{}, Capabilities{}, image.Point{}, true},
		{"cursor split over reads", []string{"\x1b[6;20;10t\x1b[1", "2;1R\x1b[?61;4c"}, offer{}, Capabilities{Graphics: GraphicsSixel, CellPixels: image.Pt(10, 20)}, image.Pt(0, 11), true},
		{"silent", nil, offer{}, Capabilities{}, image.Point{}, false},
	}
	for _, tc := range cases {
		term := newFake(tc.answers...)
		start := time.Now()
		caps, cursor, err := query(term, term.tty, tc.offer)
		took := time.Since(start)
		if err != nil {
			t.Fatal(err)
		}
		if caps != tc.caps || cursor != tc.cursor {
			t.Errorf("%s: %+v cursor %v, want %+v %v", tc.name, caps, cursor, tc.caps, tc.cursor)
		}
		switch {
		case tc.fenced && took >= konst.QueryTimeout:
			t.Errorf("%s: took %v, want the fence", tc.name, took)
		case !tc.fenced && (took < konst.QueryTimeout || took > 2*konst.QueryTimeout):
			t.Errorf("%s: took %v, want the %v timeout", tc.name, took, konst.QueryTimeout)
		}
		if out := strings.TrimSuffix(term.written.String(), konst.KittyFenced); out != konst.Probes+konst.InlineQueries {
			t.Errorf("%s: wrote %q, want the probes and the inline queries alone, the fence last", tc.name, out)
		}
		if term.tty.restored != 1 {
			t.Errorf("%s: restored %d times, want 1", tc.name, term.tty.restored)
		}
		if n := len(term.tty.input); n != 0 {
			t.Errorf("%s: %d answers left unread", tc.name, n)
		}
	}
}

func TestWidthProbes(t *testing.T) {
	const wt = "\x1b[?2027;3$y\x1b[1;1R\x1b[1;3R\x1b[1;3R\x1b[1;3R\x1b[1;3R\x1b[1;3R\x1b[?2027;3$y\x1b[?2026;2$y\x1b[6;20;10t\x1b[4;480;800t\x1b[?61;4;6;7;14;21;22;23;24;28;32;42;52c"
	cases := []struct {
		name      string
		answers   []string
		widths    text.Widths
		graphemes bool
		fenced    bool
	}{
		{"windows terminal 1.24", []string{wt}, text.Widths{2, 2, 2, 2, 2}, true, true},
		{"silent", nil, text.Widths{}, false, false},
		{"no cluster support", []string{"\x1b[?2027;0$y\x1b[3;5R\x1b[3;9R\x1b[3;11R\x1b[3;6R\x1b[3;9R\x1b[3;8R\x1b[?62c"}, text.Widths{4, 6, 1, 4, 3}, false, true},
		{"reports split over reads", []string{"\x1b[?2027;1$y\x1b[1;1R\x1b[1;2R\x1b[1", ";3R\x1b[1;3R\x1b[1;3R", "\x1b[1;3R\x1b[?62c"}, text.Widths{1, 2, 2, 2, 2}, true, true},
		{"fewer reports than probes", []string{"\x1b[1;1R\x1b[1;2R\x1b[1;3R\x1b[1;3R\x1b[1;3R\x1b[?62c"}, text.Widths{}, false, true},
		{"right margin clamps", []string{"\x1b[1;76R\x1b[1;78R\x1b[1;80R\x1b[1;78R\x1b[1;80R\x1b[1;79R\x1b[?62c"}, text.Widths{2, 0, 2, 0, 3}, false, true},
		{"another row or no advance", []string{"\x1b[1;1R\x1b[2;1R\x1b[1;1R\x1b[1;3R\x1b[1;3R\x1b[1;3R\x1b[?62c"}, text.Widths{0, 0, 2, 2, 2}, false, true},
		{"2027 permanently reset", []string{"\x1b[?2027;4$y\x1b[?62c"}, text.Widths{}, false, true},
		{"2027 reset", []string{"\x1b[?2027;2$y\x1b[?62c"}, text.Widths{}, true, true},
	}
	for _, tc := range cases {
		term := newFake(tc.answers...)
		start := time.Now()
		b, err := enter(term, term.tty, Options{}, offer{})
		took := time.Since(start)
		if err != nil {
			t.Fatal(err)
		}
		if b.Capabilities.Widths != tc.widths || b.Capabilities.Graphemes != tc.graphemes {
			t.Errorf("%s: widths %v graphemes %v, want %v %v", tc.name, b.Capabilities.Widths, b.Capabilities.Graphemes, tc.widths, tc.graphemes)
		}
		if tc.fenced != (took < konst.QueryTimeout) || took > 2*konst.StartupTimeout {
			t.Errorf("%s: took %v, fenced %v", tc.name, took, tc.fenced)
		}
		term.tty.input <- []byte("x")
		select {
		case ev := <-b.Events:
			if ev != (input.KeyEvent{Rune: 'x'}) {
				t.Errorf("%s: first event %#v, want the key typed after the probes", tc.name, ev)
			}
		case <-time.After(time.Second):
			t.Errorf("%s: no key event", tc.name)
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
		out := term.written.String()
		if !strings.Contains(out, konst.GraphemesOn+konst.Probes) {
			t.Errorf("%s: wrote %q, want the probes after mode 2027 is set", tc.name, out)
		}
		if strings.Contains(out, konst.GraphemesOff) != tc.graphemes {
			t.Errorf("%s: mode 2027 reset on exit %v, want %v", tc.name, !tc.graphemes, tc.graphemes)
		}
	}
}

func TestProbeOrder(t *testing.T) {
	body := strings.TrimSuffix(strings.TrimPrefix(konst.Probes, konst.ProbeBegin), konst.ProbeEnd)
	clusters := strings.Split(body, konst.ProbeStep)
	if len(clusters) != int(text.Classes)+1 || clusters[text.Classes] != "" {
		t.Fatalf("probes %q, want one cluster per class", clusters)
	}
	for c := range text.Classes {
		var w text.Widths
		w[c] = 7
		if got := w.Width(clusters[c]); got != 7 {
			t.Errorf("probe %d %+q measures %d under an override of 7 for its class", c, clusters[c], got)
		}
	}
}

func TestProbeReturnsEveryAnswer(t *testing.T) {
	version := "\x1bP>|WezTerm 20260929\x1b\\"
	pointer := "\x1b]22;default\x07"
	cases := []struct {
		name    string
		answers []string
		fenced  bool
	}{
		{"wezterm", []string{version + "\x1b[>1;277;0c\x1b[?1003;2$y", pointer + "\x1bP1$r0;38:2::1:2:3m\x1b\\\x1b[?65;4;6;18;22;52c"}, true},
		{"split inside a control string", []string{"\x1bP>|Wez", "Term 20260929\x1b\\\x1b[?62c"}, true},
		{"silent", nil, false},
	}
	for _, tc := range cases {
		term := newFake(tc.answers...)
		start := time.Now()
		b, raw, _, err := probe(term, term.tty, konst.DoctorQueries+konst.Fence, konst.StartupTimeout)
		took := time.Since(start)
		if err := errors.Join(err, b.Exit()); err != nil {
			t.Fatal(err)
		}
		if want := strings.Join(tc.answers, ""); string(raw) != want {
			t.Errorf("%s: raw %q, want %q", tc.name, raw, want)
		}
		if tc.fenced != (took < konst.QueryTimeout) || took > 2*konst.StartupTimeout {
			t.Errorf("%s: took %v, fenced %v", tc.name, took, tc.fenced)
		}
		if out := term.written.String(); out != konst.DoctorQueries+konst.Fence {
			t.Errorf("%s: wrote %q, want the queries and the fence alone", tc.name, out)
		}
		if term.tty.restored != 1 {
			t.Errorf("%s: restored %d times, want 1", tc.name, term.tty.restored)
		}
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

func TestInlineGraphemesAndFocus(t *testing.T) {
	for _, tc := range []struct {
		name             string
		answers          []string
		graphemes, focus bool
	}{
		{"wezterm nightly", []string{"\x1b[?2026;2$y\x1b[?2027;3$y\x1b[?1004;2$y\x1b[?69;2$y\x1b[1;1R\x1b[?65;4;6;18;22;52c"}, true, true},
		{"focus set", []string{"\x1b[?1004;1$y\x1b[1;1R\x1b[?62c"}, false, true},
		{"focus kept set", []string{"\x1b[?1004;3$y\x1b[1;1R\x1b[?62c"}, false, true},
		{"not recognised", []string{"\x1b[?2027;0$y\x1b[?1004;0$y\x1b[1;1R\x1b[?62c"}, false, false},
		{"permanently reset", []string{"\x1b[?2027;4$y\x1b[?1004;4$y\x1b[1;1R\x1b[?62c"}, false, false},
		{"split over reads", []string{"\x1b[?2027;3$y\x1b[?10", "04;2$y\x1b[1;1R\x1b[?62c"}, true, true},
		{"another mode", []string{"\x1b[?2026;1$y\x1b[?1049;1$y\x1b[1;1R\x1b[?62c"}, false, false},
	} {
		term := newFake(tc.answers...)
		caps, _, err := query(term, term.tty, offer{})
		if err != nil {
			t.Fatal(err)
		}
		if caps.Graphemes != tc.graphemes || caps.Focus != tc.focus {
			t.Errorf("%s: graphemes %v focus %v, want %v %v", tc.name, caps.Graphemes, caps.Focus, tc.graphemes, tc.focus)
		}
		out := term.written.String()
		fence := strings.LastIndex(out, "\x1b[c")
		for _, ask := range []string{"\x1b[?2027$p", "\x1b[?1004$p"} {
			if i := strings.Index(out, ask); i < 0 || i > fence {
				t.Errorf("%s: %q missing or after the fence in %q", tc.name, ask, out)
			}
		}
		for _, set := range []string{konst.GraphemesOn, konst.GraphemesOff, konst.FocusOn, konst.FocusOff} {
			if strings.Contains(out, set) {
				t.Errorf("%s: the inline query wrote %q", tc.name, set)
			}
		}
	}
}

func TestFocusReports(t *testing.T) {
	term := newFake("\x1b[?1004;1$y\x1b[I\x1b[?62c")
	b, err := enter(term, term.tty, Options{}, offer{})
	if err != nil {
		t.Fatal(err)
	}
	if !b.Capabilities.Focus {
		t.Errorf("focus not detected from ?1004;1: %+v", b.Capabilities)
	}
	for _, in := range []string{"\x1b[O", "\x1b[", "I", "\x1b[Ox"} {
		term.tty.input <- []byte(in)
	}
	for _, w := range []input.Event{input.FocusEvent{Focused: true}, input.FocusEvent{}, input.FocusEvent{Focused: true}, input.FocusEvent{}, input.KeyEvent{Rune: 'x'}} {
		select {
		case ev := <-b.Events:
			if ev != w {
				t.Errorf("event %#v, want %#v", ev, w)
			}
		case <-time.After(time.Second):
			t.Fatalf("no event, want %#v", w)
		}
	}
	if err := errors.Join(b.Exit(), b.Exit()); err != nil {
		t.Fatal(err)
	}
	out := term.written.String()
	if on, ask := strings.Index(out, konst.FocusOn), strings.Index(out, "\x1b[?1004$p"); on < 0 || ask < on {
		t.Errorf("wrote %q, want ?1004h before the ?1004 question", out)
	}
	if strings.Count(out, konst.FocusOff) != 1 || !strings.HasSuffix(out, konst.LeaveScreen) || !strings.Contains(konst.LeaveScreen, konst.FocusOff) {
		t.Errorf("wrote %q, want ?1004l once, on exit", out)
	}
}

func TestSixelFromATerminalStillStarting(t *testing.T) {
	term := newFake(windowsTerminal...)
	term.delay = 4 * konst.QueryTimeout
	start := time.Now()
	b, err := enter(term, term.tty, Options{}, offer{})
	took := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if b.Capabilities.Graphics != GraphicsSixel || b.Capabilities.CellPixels != image.Pt(10, 20) || !b.Capabilities.Sync {
		t.Errorf("answers %v after the start: %+v, want sixel, 10x20 cells and sync", term.delay, b.Capabilities)
	}
	if took > 2*term.delay+konst.QueryTimeout {
		t.Errorf("took %v, want the second fence at %v", took, 2*term.delay)
	}
	if err := b.Exit(); err != nil {
		t.Fatal(err)
	}
}

func TestSixelWhenKittyRefusesCompression(t *testing.T) {
	for _, tc := range []struct {
		name     string
		zlib     bool
		graphics Graphics
	}{
		{"contour 0.7.0 refuses o=z", false, GraphicsSixel},
		{"kitty takes o=z", true, GraphicsKitty},
	} {
		term := newFake()
		term.reply = func(p []byte) []string {
			const da1 = "\x1b[?65;1;3;4;7;9;18;21;22;29;52;314c"
			start := bytes.Index(p, []byte("\x1b_G"))
			switch {
			case !bytes.HasSuffix(p, []byte(konst.Fence)):
				return nil
			case start < 0:
				return []string{"\x1b[6;19;9t" + da1}
			}
			control, _, _ := bytes.Cut(p[start:], []byte(";"))
			status := "OK"
			if bytes.Contains(control, []byte("o=z")) && !tc.zlib {
				status = "ENOTSUP:compressed payloads are not supported"
			}
			return []string{"\x1b_Gi=31;" + status + "\x1b\\" + da1}
		}
		b, err := enter(term, term.tty, Options{}, offer{})
		if err != nil {
			t.Fatal(err)
		}
		if b.Capabilities.Graphics != tc.graphics || b.Capabilities.CellPixels != image.Pt(9, 19) {
			t.Errorf("%s: graphics %d cell %v, want %d 9x19", tc.name, b.Capabilities.Graphics, b.Capabilities.CellPixels, tc.graphics)
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSizeFromTheTerminal(t *testing.T) {
	for _, tc := range []struct {
		name    string
		answers []string
		size    image.Point
		cell    image.Point
	}{
		{"tabby: the pty was told a stale width", []string{"\x1b[8;36;120t\x1b[6;17;7t\x1b[4;612;840t\x1b[?62;4;9;22c"}, image.Pt(120, 36), image.Pt(7, 17)},
		{"window pixels over the terminal's grid", []string{"\x1b[4;720;1200t\x1b[8;36;120t\x1b[?62;4c"}, image.Pt(120, 36), image.Pt(10, 20)},
		{"no grid answer", []string{"\x1b[6;17;7t\x1b[?62;4c"}, image.Pt(109, 36), image.Pt(7, 17)},
		{"zero grid", []string{"\x1b[8;0;0t\x1b[6;17;7t\x1b[?62;4c"}, image.Pt(109, 36), image.Pt(7, 17)},
		{"short grid report", []string{"\x1b[8;36t\x1b[6;17;7t\x1b[?62;4c"}, image.Pt(109, 36), image.Pt(7, 17)},
	} {
		term := newFake(tc.answers...)
		term.tty.cells = image.Pt(109, 36)
		b, err := enter(term, term.tty, Options{}, offer{})
		if err != nil {
			t.Fatal(err)
		}
		if w, h, err := b.Size(); image.Pt(w, h) != tc.size || err != nil || b.Capabilities.CellPixels != tc.cell {
			t.Errorf("%s: size %dx%d %v cell %v, want %v cell %v", tc.name, w, h, err, b.Capabilities.CellPixels, tc.size, tc.cell)
		}
		if !strings.Contains(term.out(), konst.GridQuery) {
			t.Errorf("%s: wrote %q, want the grid asked", tc.name, term.out())
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSizeAfterAConsoleResize(t *testing.T) {
	const tabby = "\x1b[8;36;120t\x1b[6;17;7t\x1b[4;612;840t\x1b[?62;4;9;22c"
	key := input.KeyEvent{Rune: 'x'}
	for _, tc := range []struct {
		name    string
		answers []string
		later   []string
		console image.Point
		want    []input.Event
		size    image.Point
	}{
		{"the pty shrinks, the grid stays", []string{tabby}, []string{"\x1b[8;36;120t\x1b[?62;4c"}, image.Pt(109, 36), []input.Event{key}, image.Pt(120, 36)},
		{"the grid follows the pty", []string{tabby}, []string{"\x1b[8;30;100t\x1b[?62;4c"}, image.Pt(100, 30), []input.Event{input.ResizeEvent{Width: 100, Height: 30}, key}, image.Pt(100, 30)},
		{"the grid moves, the pty does not", []string{tabby}, []string{"\x1b[8;40;130t\x1b[?62;4c"}, image.Pt(120, 36), []input.Event{input.ResizeEvent{Width: 130, Height: 40}, key}, image.Pt(130, 40)},
		{"no grid answer after the resize", []string{tabby}, []string{"\x1b[?62;4c"}, image.Pt(100, 30), []input.Event{input.ResizeEvent{Width: 100, Height: 30}, key}, image.Pt(100, 30)},
		{"a grid first answered after a resize", []string{"\x1b[6;17;7t\x1b[?62;4c"}, []string{"\x1b[8;36;120t\x1b[?62;4c"}, image.Pt(100, 30), []input.Event{input.ResizeEvent{Width: 100, Height: 30}, input.ResizeEvent{Width: 120, Height: 36}, key}, image.Pt(120, 36)},
	} {
		term := newFake(tc.answers...)
		term.later = tc.later
		term.tty.cells = image.Pt(120, 36)
		b, err := enter(term, term.tty, Options{}, offer{})
		if err != nil {
			t.Fatal(err)
		}
		before := len(term.out())
		term.tty.resize <- tc.console
		for deadline := time.Now().Add(time.Second); !strings.Contains(term.out()[before:], konst.CellQuery) && time.Now().Before(deadline); {
			time.Sleep(time.Millisecond)
		}
		term.tty.input <- []byte("x")
		events(t, b, tc.name, tc.want...)
		if w, h, _ := b.Size(); image.Pt(w, h) != tc.size {
			t.Errorf("%s: size %dx%d, want %v", tc.name, w, h, tc.size)
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSizeWhileAQueryIsOut(t *testing.T) {
	var mu sync.Mutex
	grid := "\x1b[8;36;120t"
	term := newFake()
	term.delay = konst.QueryTimeout / 2
	term.reply = func(p []byte) []string {
		if !bytes.HasSuffix(p, []byte(konst.Fence)) {
			return nil
		}
		mu.Lock()
		defer mu.Unlock()
		return []string{grid + "\x1b[6;17;7t\x1b[?62;4c"}
	}
	term.tty.cells = image.Pt(120, 36)
	b, err := enter(term, term.tty, Options{}, offer{})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	grid = "\x1b[8;30;100t"
	mu.Unlock()
	term.tty.resize <- image.Pt(100, 30)
	for deadline := time.Now().Add(time.Second); strings.Count(term.out(), konst.CellQuery) < 2 && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	mu.Lock()
	grid = "\x1b[8;25;90t"
	mu.Unlock()
	term.tty.resize <- image.Pt(90, 25)
	events(t, b, "two resizes, one answer out", input.ResizeEvent{Width: 100, Height: 30}, input.ResizeEvent{Width: 90, Height: 25})
	if n := strings.Count(term.out(), konst.CellQuery); n != 3 {
		t.Errorf("asked %d times, want startup and one per resize", n)
	}
	if err := b.Exit(); err != nil {
		t.Fatal(err)
	}
}

func TestStartupLeaksNothingToConhost(t *testing.T) {
	answer := func(da1, cell string) func(p []byte) []string {
		return func(p []byte) []string {
			if !bytes.HasSuffix(p, []byte(konst.Fence)) {
				return nil
			}
			if bytes.Contains(p, []byte("\x1b_G")) {
				return []string{konst.KittyOK + da1}
			}
			return []string{cell + da1}
		}
	}
	for _, tc := range []struct {
		name     string
		reply    func(p []byte) []string
		offer    offer
		asked    bool
		graphics Graphics
		took     time.Duration
	}{
		{"conhost", answer("\x1b[?1;0c", ""), offer{}, false, GraphicsNone, konst.QueryTimeout},
		{"conhost with a cell size", answer("\x1b[?1;0c", "\x1b[6;16;8t"), offer{}, false, GraphicsNone, konst.QueryTimeout},
		{"wezterm", answer("\x1b[?65;4;6;18;22;52c", "\x1b[6;22;10t"), offer{}, true, GraphicsKitty, konst.QueryTimeout},
		{"kitty", answer("\x1b[?62;c", "\x1b[6;36;17t"), offer{}, true, GraphicsKitty, konst.QueryTimeout},
		{"no cell size", answer("\x1b[?65;4c", ""), offer{}, false, GraphicsNone, konst.QueryTimeout},
		{"forced sixel", answer("\x1b[?65;4c", "\x1b[6;22;10t"), offer{GraphicsSixel, true}, false, GraphicsSixel, konst.QueryTimeout},
		{"silent", func([]byte) []string { return nil }, offer{}, false, GraphicsNone, konst.StartupTimeout + konst.QueryTimeout},
	} {
		for _, inline := range []bool{false, true} {
			term := newFake()
			term.reply = tc.reply
			start := time.Now()
			var caps Capabilities
			if inline {
				var err error
				if caps, _, err = query(term, term.tty, tc.offer); err != nil {
					t.Fatal(err)
				}
			} else {
				b, err := enter(term, term.tty, Options{}, tc.offer)
				if err != nil {
					t.Fatal(err)
				}
				caps = b.Capabilities
				term.tty.input <- []byte("x")
				events(t, b, tc.name, input.KeyEvent{Rune: 'x'})
				if err := b.Exit(); err != nil {
					t.Fatal(err)
				}
			}
			if took := time.Since(start); took > tc.took+konst.QueryTimeout/2 {
				t.Errorf("%s inline %v: took %v, want under %v", tc.name, inline, took, tc.took)
			}
			if asked := strings.Contains(term.out(), "\x1b_"); asked != tc.asked {
				t.Errorf("%s inline %v: APC written %v, want %v in %q", tc.name, inline, asked, tc.asked, term.out())
			}
			if !inline && caps.Graphics != tc.graphics {
				t.Errorf("%s: graphics %d, want %d", tc.name, caps.Graphics, tc.graphics)
			}
		}
	}
}
