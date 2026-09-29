package terminal

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/input"
)

type fakeTTY struct {
	input     chan []byte
	cancelled chan struct{}
	restored  int
}

func (f *fakeTTY) read(p []byte, quiet bool) (int, bool, error) {
	var silence <-chan time.Time
	if quiet {
		silence = time.After(konst.EscapeTimeout)
	}
	select {
	case b := <-f.input:
		return copy(p, b), false, nil
	case <-silence:
		return 0, false, errQuiet
	case <-f.cancelled:
		return 0, false, io.EOF
	}
}

func (f *fakeTTY) size() (width, height int, err error) { return 80, 24, nil }
func (f *fakeTTY) cancel()                              { close(f.cancelled) }
func (f *fakeTTY) restore() error                       { f.restored++; return nil }

type fakeTerminal struct {
	tty     *fakeTTY
	answers []string
	written bytes.Buffer
	broken  bool
}

func (t *fakeTerminal) Write(p []byte) (int, error) {
	if t.broken {
		return 0, errors.New("terminal gone")
	}
	t.written.Write(p)
	if bytes.Contains(p, []byte(konst.Queries)) {
		for _, a := range t.answers {
			t.tty.input <- []byte(a)
		}
	}
	return len(p), nil
}

func newFake(answers ...string) *fakeTerminal {
	return &fakeTerminal{tty: &fakeTTY{input: make(chan []byte, 16), cancelled: make(chan struct{})}, answers: answers}
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
			b, err := enter(term, term.tty, Options{})
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

func TestEvents(t *testing.T) {
	term := newFake("\x1b[?2026;1$y", "x", "\x1b[?62c")
	b, err := enter(term, term.tty, Options{Mouse: true})
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
	b, err := enter(term, term.tty, Options{})
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
