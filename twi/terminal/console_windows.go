//go:build windows

package terminal

import (
	"errors"
	"io"
	"os"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/sys/windows"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
)

func size(fd uintptr) (width, height int, err error) {
	return consoleSize(kernel32{}, windows.Handle(fd))
}

func consoleSize(c console, h windows.Handle) (width, height int, err error) {
	var info windows.ConsoleScreenBufferInfo
	if err := c.bufferInfo(h, &info); err != nil {
		return 0, 0, err
	}
	return int(info.Window.Right-info.Window.Left) + 1, int(info.Window.Bottom-info.Window.Top) + 1, nil
}

func EnableVirtualTerminal(f *os.File) (restore func() error, err error) {
	handle := windows.Handle(f.Fd())
	var original uint32
	if err := windows.GetConsoleMode(handle, &original); err != nil {
		return nil, err
	}
	if err := windows.SetConsoleMode(handle, original|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		return nil, err
	}
	return func() error { return windows.SetConsoleMode(handle, original) }, nil
}

type inputRecord struct {
	kind    uint16
	_       uint16
	keyDown int32
	repeat  uint16
	vk      uint16
	scan    uint16
	char    uint16
	control uint32
}

type console interface {
	getMode(h windows.Handle, mode *uint32) error
	setMode(h windows.Handle, mode uint32) error
	bufferInfo(h windows.Handle, info *windows.ConsoleScreenBufferInfo) error
	readInput(in, cancel windows.Handle, records []inputRecord, timeout uint32) (int, error)
}

type kernel32 struct{ readConsoleInput *windows.LazyProc }

func (kernel32) getMode(h windows.Handle, mode *uint32) error {
	return windows.GetConsoleMode(h, mode)
}

func (kernel32) setMode(h windows.Handle, mode uint32) error {
	return windows.SetConsoleMode(h, mode)
}

func (kernel32) bufferInfo(h windows.Handle, info *windows.ConsoleScreenBufferInfo) error {
	return windows.GetConsoleScreenBufferInfo(h, info)
}

func (k kernel32) readInput(in, cancel windows.Handle, records []inputRecord, timeout uint32) (int, error) {
	event, err := windows.WaitForMultipleObjects([]windows.Handle{cancel, in}, false, timeout)
	switch {
	case err != nil:
		return 0, err
	case event == uint32(windows.WAIT_TIMEOUT):
		return 0, errQuiet
	case event == windows.WAIT_OBJECT_0:
		return 0, io.EOF
	}
	var n uint32
	ok, _, err := k.readConsoleInput.Call(uintptr(in), uintptr(unsafe.Pointer(&records[0])), uintptr(len(records)), uintptr(unsafe.Pointer(&n)))
	if ok == 0 {
		return 0, err
	}
	return int(n), nil
}

type consoleTTY struct {
	console         console
	in, out         windows.Handle
	inMode, outMode uint32
	cancelled       windows.Handle
	records         []inputRecord
	high            rune
}

func openTTY(in, out *os.File, opt Options) (tty, error) {
	k := kernel32{windows.NewLazySystemDLL("kernel32.dll").NewProc("ReadConsoleInputW")}
	return openConsole(k, windows.Handle(in.Fd()), windows.Handle(out.Fd()), opt)
}

func openConsole(c console, in, out windows.Handle, opt Options) (tty, error) {
	t := &consoleTTY{console: c, in: in, out: out, records: make([]inputRecord, konst.ReadBuffer/utf8.UTFMax)}
	if err := errors.Join(c.getMode(in, &t.inMode), c.getMode(out, &t.outMode)); err != nil {
		return nil, err
	}
	cancelled, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return nil, err
	}
	t.cancelled = cancelled
	mode := uint32(windows.ENABLE_VIRTUAL_TERMINAL_INPUT | windows.ENABLE_WINDOW_INPUT | windows.ENABLE_EXTENDED_FLAGS)
	if !opt.Mouse {
		mode |= t.inMode & windows.ENABLE_QUICK_EDIT_MODE
	}
	err = errors.Join(c.setMode(out, t.outMode|windows.ENABLE_PROCESSED_OUTPUT|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING), c.setMode(in, mode))
	if err != nil {
		return nil, errors.Join(err, t.restore())
	}
	return t, nil
}

func (t *consoleTTY) read(p []byte, quiet bool) (int, bool, error) {
	timeout := uint32(windows.INFINITE)
	if quiet {
		timeout = uint32(konst.EscapeTimeout.Milliseconds())
	}
	n, err := t.console.readInput(t.in, t.cancelled, t.records, timeout)
	if err != nil {
		return 0, false, err
	}
	p, resized := p[:0], false
	for _, r := range t.records[:n] {
		switch r.kind {
		case windows.KEY_EVENT:
			c := rune(r.char)
			switch {
			case r.keyDown == 0 || c == 0:
			case utf16.IsSurrogate(c) && t.high == 0:
				t.high = c
			default:
				if t.high != 0 {
					c, t.high = utf16.DecodeRune(t.high, c), 0
				}
				p = utf8.AppendRune(p, c)
			}
		case windows.WINDOW_BUFFER_SIZE_EVENT:
			resized = true
		case windows.MOUSE_EVENT, windows.FOCUS_EVENT, windows.MENU_EVENT:
		}
	}
	return len(p), resized, nil
}

func (t *consoleTTY) size() (width, height int, err error) {
	return consoleSize(t.console, t.out)
}

func (t *consoleTTY) cancel() {
	_ = windows.SetEvent(t.cancelled)
}

func (t *consoleTTY) restore() error {
	return errors.Join(t.console.setMode(t.in, t.inMode), t.console.setMode(t.out, t.outMode), windows.CloseHandle(t.cancelled))
}
