//go:build windows

package terminal

import (
	"errors"
	"io"
	"maps"
	"os"
	"testing"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
)

const fakeIn, fakeOut windows.Handle = 10, 11

type fakeConsole struct {
	modes   map[windows.Handle]uint32
	refuse  windows.Handle
	records []inputRecord
	rows    int16
	class   string
}

func (f *fakeConsole) windowClass() string { return f.class }

func (f *fakeConsole) font(windows.Handle) Font { return Font{} }

func (f *fakeConsole) lacks(string, string) bool { return false }

func (f *fakeConsole) drawable() (window, error) { return nil, errNoWindow }

func (f *fakeConsole) overlay(*trace) (host, error) { return nil, errNoWindow }

func (f *fakeConsole) getMode(h windows.Handle, mode *uint32) error {
	*mode = f.modes[h]
	return nil
}

func (f *fakeConsole) setMode(h windows.Handle, mode uint32) error {
	if h == f.refuse {
		return errors.New("not a console")
	}
	f.modes[h] = mode
	return nil
}

func (f *fakeConsole) bufferInfo(_ windows.Handle, info *windows.ConsoleScreenBufferInfo) error {
	info.Size = windows.Coord{X: 80, Y: 9001}
	info.Window = windows.SmallRect{Left: 0, Top: 100, Right: 79, Bottom: 100 + f.rows - 1}
	return nil
}

func (f *fakeConsole) readInput(_, cancel windows.Handle, records []inputRecord, timeout uint32) (int, error) {
	if len(f.records) > 0 {
		records[0], f.records = f.records[0], f.records[1:]
		if records[0].kind == windows.WINDOW_BUFFER_SIZE_EVENT {
			f.rows = 30
		}
		return 1, nil
	}
	if event, _ := windows.WaitForSingleObject(cancel, timeout); event == uint32(windows.WAIT_TIMEOUT) {
		return 0, errQuiet
	}
	return 0, io.EOF
}

func conhostModes() map[windows.Handle]uint32 {
	return map[windows.Handle]uint32{fakeIn: 0x1f7, fakeOut: 0x3}
}

func TestConsoleModesRestored(t *testing.T) {
	for _, mouse := range []bool{false, true} {
		c := &fakeConsole{modes: conhostModes()}
		var raw map[windows.Handle]uint32
		func() {
			defer func() { _ = recover() }()
			con, err := openConsole(c, fakeIn, fakeOut, Options{NoMouse: !mouse})
			if err != nil {
				t.Fatal(err)
			}
			b, err := enter(io.Discard, con, Options{NoMouse: !mouse}, offer{})
			if err != nil {
				t.Fatal(err)
			}
			defer b.Exit() //nolint:errcheck
			raw = maps.Clone(c.modes)
			panic("between enter and exit")
		}()
		in := raw[fakeIn]
		if in&(windows.ENABLE_LINE_INPUT|windows.ENABLE_ECHO_INPUT|windows.ENABLE_PROCESSED_INPUT) != 0 || in&windows.ENABLE_VIRTUAL_TERMINAL_INPUT == 0 {
			t.Errorf("mouse %v: input mode %#x inside is not raw VT input", mouse, in)
		}
		if quickEdit := in&windows.ENABLE_QUICK_EDIT_MODE != 0; quickEdit == mouse {
			t.Errorf("mouse %v: quick edit %v inside", mouse, quickEdit)
		}
		if mouseInput := in&windows.ENABLE_MOUSE_INPUT != 0; mouseInput != mouse {
			t.Errorf("mouse %v: mouse input %v inside", mouse, mouseInput)
		}
		if raw[fakeOut]&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING == 0 {
			t.Errorf("mouse %v: output mode %#x inside has no VT processing", mouse, raw[fakeOut])
		}
		if !maps.Equal(c.modes, conhostModes()) {
			t.Errorf("mouse %v: modes after exit %#x, before enter %#x", mouse, c.modes, conhostModes())
		}
	}

	c := &fakeConsole{modes: conhostModes(), refuse: fakeIn}
	if _, err := openConsole(c, fakeIn, fakeOut, Options{}); err == nil {
		t.Error("input refused, want an error")
	}
	if !maps.Equal(c.modes, conhostModes()) {
		t.Errorf("modes after a refused enter %#x, before %#x", c.modes, conhostModes())
	}
}

func TestConsoleRecords(t *testing.T) {
	high, low := utf16.EncodeRune('😀')
	key := func(c rune, down int32) inputRecord {
		return inputRecord{kind: windows.KEY_EVENT, keyDown: down, char: uint16(c), repeat: 1}
	}
	size := inputRecord{kind: windows.WINDOW_BUFFER_SIZE_EVENT}
	c := &fakeConsole{rows: 24, modes: conhostModes(), records: []inputRecord{
		key('a', 1), key('a', 0), key(0, 1), key(high, 1), key(low, 1), size, size, {kind: windows.FOCUS_EVENT}, key('b', 1),
	}}
	con, err := openConsole(c, fakeIn, fakeOut, Options{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := enter(io.Discard, con, Options{}, offer{})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Exit() //nolint:errcheck
	for _, want := range []input.Event{input.KeyEvent{Rune: 'a'}, input.KeyEvent{Rune: '😀'}, input.ResizeEvent{Width: 80, Height: 30}, input.KeyEvent{Rune: 'b'}} {
		select {
		case ev := <-b.Events:
			if ev != want {
				t.Errorf("event %#v, want %#v", ev, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("no event, want %#v", want)
		}
	}
}

func TestConsoleMouse(t *testing.T) {
	click := func(x, y int16, buttons, control, flags uint32) inputRecord {
		m := mouseRecord{kind: windows.MOUSE_EVENT, x: x, y: y, buttons: buttons, control: control, flags: flags}
		return *(*inputRecord)(unsafe.Pointer(&m))
	}
	wheel := func(delta int16) uint32 { return uint32(uint16(delta)) << 16 }
	records := []inputRecord{
		click(5, 103, windows.FROM_LEFT_1ST_BUTTON_PRESSED, windows.SHIFT_PRESSED, 0),
		click(7, 104, windows.FROM_LEFT_1ST_BUTTON_PRESSED, 0, windows.MOUSE_MOVED),
		click(7, 104, 0, windows.LEFT_CTRL_PRESSED, 0),
		click(0, 100, 0, 0, windows.MOUSE_MOVED),
		click(79, 123, wheel(120), windows.RIGHT_CTRL_PRESSED|windows.SHIFT_PRESSED, windows.MOUSE_WHEELED),
		click(2, 101, wheel(-120), 0, windows.MOUSE_WHEELED),
		click(3, 102, windows.RIGHTMOST_BUTTON_PRESSED, windows.LEFT_ALT_PRESSED, 0),
		click(3, 102, windows.RIGHTMOST_BUTTON_PRESSED|windows.FROM_LEFT_2ND_BUTTON_PRESSED, 0, 0),
		click(3, 102, windows.FROM_LEFT_2ND_BUTTON_PRESSED, 0, 0),
		click(3, 102, 0, 0, 0),
		click(4, 102, windows.FROM_LEFT_1ST_BUTTON_PRESSED, 0, windows.DOUBLE_CLICK),
		click(4, 102, 0, 0, 0),
	}
	for _, c := range "\x1b[<0;2;2M" {
		records = append(records, inputRecord{kind: windows.KEY_EVENT, keyDown: 1, char: uint16(c), repeat: 1})
	}
	records = append(records,
		click(1, 101, windows.FROM_LEFT_1ST_BUTTON_PRESSED, 0, 0),
		inputRecord{kind: windows.KEY_EVENT, keyDown: 1, char: 'z', repeat: 1},
	)
	c := &fakeConsole{rows: 24, modes: conhostModes(), records: records}
	con, err := openConsole(c, fakeIn, fakeOut, Options{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := enter(io.Discard, con, Options{}, offer{})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Exit() //nolint:errcheck
	for _, want := range []input.Event{
		input.MouseEvent{X: 5, Y: 3, Button: input.MouseLeft, Action: input.MousePress, Modifiers: input.ModShift},
		input.MouseEvent{X: 7, Y: 4, Button: input.MouseLeft, Action: input.MouseMove},
		input.MouseEvent{X: 7, Y: 4, Button: input.MouseLeft, Action: input.MouseRelease, Modifiers: input.ModCtrl},
		input.MouseEvent{Button: input.MouseNone, Action: input.MouseMove},
		input.MouseEvent{X: 79, Y: 23, Button: input.MouseWheelUp, Action: input.MouseScroll, Modifiers: input.ModShift | input.ModCtrl},
		input.MouseEvent{X: 2, Y: 1, Button: input.MouseWheelDown, Action: input.MouseScroll},
		input.MouseEvent{X: 3, Y: 2, Button: input.MouseRight, Action: input.MousePress, Modifiers: input.ModAlt},
		input.MouseEvent{X: 3, Y: 2, Button: input.MouseMiddle, Action: input.MousePress},
		input.MouseEvent{X: 3, Y: 2, Button: input.MouseRight, Action: input.MouseRelease},
		input.MouseEvent{X: 3, Y: 2, Button: input.MouseMiddle, Action: input.MouseRelease},
		input.MouseEvent{X: 4, Y: 2, Button: input.MouseLeft, Action: input.MousePress},
		input.MouseEvent{X: 4, Y: 2, Button: input.MouseLeft, Action: input.MouseRelease},
		input.MouseEvent{X: 1, Y: 1, Button: input.MouseLeft, Action: input.MousePress},
		input.KeyEvent{Rune: 'z'},
	} {
		select {
		case ev := <-b.Events:
			if ev != want {
				t.Errorf("event %#v, want %#v", ev, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("no event, want %#v", want)
		}
	}
}

func TestEnableVirtualTerminal(t *testing.T) {
	r, pipe, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, err = EnableVirtualTerminal(pipe)
	_, _ = r.Close(), pipe.Close()
	if err == nil {
		t.Error("a pipe is not a console, want an error")
	}

	console, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("no console attached: %v", err)
	}
	defer console.Close() //nolint:errcheck
	handle := windows.Handle(console.Fd())
	var original, enabled, restored uint32
	if err := windows.GetConsoleMode(handle, &original); err != nil {
		t.Fatal(err)
	}
	if err := windows.SetConsoleMode(handle, original&^windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		t.Fatal(err)
	}
	defer windows.SetConsoleMode(handle, original) //nolint:errcheck
	restore, err := EnableVirtualTerminal(console)
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.GetConsoleMode(handle, &enabled); err != nil {
		t.Fatal(err)
	}
	if err := restore(); err != nil {
		t.Fatal(err)
	}
	if err := windows.GetConsoleMode(handle, &restored); err != nil {
		t.Fatal(err)
	}
	if enabled&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING == 0 {
		t.Errorf("mode %#x after enable has no VT processing", enabled)
	}
	if restored != original&^windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING {
		t.Errorf("mode %#x after restore, want %#x", restored, original&^windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	}
	width, height, err := Size(console)
	if err != nil || width <= 0 || height <= 0 {
		t.Errorf("console size %dx%d, %v", width, height, err)
	}
	t.Logf("console mode %#x, VT enabled %#x, restored %#x, size %dx%d", original, enabled, restored, width, height)
}

func TestProfileAsksTheConsole(t *testing.T) {
	c := &fakeConsole{modes: conhostModes(), refuse: fakeOut}
	if got := consoleProfile(c, fakeOut); got != color.None {
		t.Errorf("VT output refused: promise %d, want none", got)
	}
	c.refuse = 0
	if got := consoleProfile(c, fakeOut); got != color.TrueColor {
		t.Errorf("VT output accepted: promise %d, want truecolor", got)
	}
	if !maps.Equal(c.modes, conhostModes()) {
		t.Errorf("modes after asking %#x, before %#x", c.modes, conhostModes())
	}

	r, pipe, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()    //nolint:errcheck
	defer pipe.Close() //nolint:errcheck
	if got := promised(pipe.Fd()); got != color.None {
		t.Errorf("a pipe promises %d, want none", got)
	}

	console, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("no console attached: %v", err)
	}
	defer console.Close() //nolint:errcheck
	handle := windows.Handle(console.Fd())
	var original, after uint32
	if err := windows.GetConsoleMode(handle, &original); err != nil {
		t.Fatal(err)
	}
	got := Profile(console, func(string) string { return "" })
	if err := windows.GetConsoleMode(handle, &after); err != nil {
		t.Fatal(err)
	}
	if got != color.TrueColor || after != original {
		t.Errorf("console: profile %d mode %#x after, want truecolor and %#x", got, after, original)
	}
}

func TestIdentityConsoleWindow(t *testing.T) {
	for class, want := range map[string]bool{konst.ConhostWindowClass: true, "PseudoConsoleWindow": false, "": false} {
		con, err := openConsole(&fakeConsole{modes: conhostModes(), class: class}, fakeIn, fakeOut, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if got := con.conhost(); got != want {
			t.Errorf("window class %q: conhost %v, want %v", class, got, want)
		}
		if err := con.restore(); err != nil {
			t.Fatal(err)
		}
	}
}
