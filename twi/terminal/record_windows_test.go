//go:build windows

package terminal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/twind-dev/twind/twi/input"
)

type recordingConsole struct {
	console
	log io.Writer
}

func (r recordingConsole) readInput(in, cancel windows.Handle, records []inputRecord, timeout uint32) (int, error) {
	n, err := r.console.readInput(in, cancel, records, timeout)
	for _, rec := range records[:n] {
		line := fmt.Sprintf("other %d\n", rec.kind)
		switch rec.kind {
		case windows.KEY_EVENT:
			line = fmt.Sprintf("key %d %d %d %d %d %d\n", rec.keyDown, rec.repeat, rec.vk, rec.scan, rec.char, rec.control)
		case windows.MOUSE_EVENT:
			m := (*mouseRecord)(unsafe.Pointer(&rec))
			line = fmt.Sprintf("mouse %d %d %d %d %d\n", m.x, m.y, m.buttons, m.control, m.flags)
		}
		if _, werr := io.WriteString(r.log, line); werr != nil {
			return n, werr
		}
	}
	return n, err
}

func TestRecordConsoleInput(t *testing.T) {
	path := os.Getenv("TWIND_RECORD")
	if path == "" {
		t.Skip("TWIND_RECORD names the file to record into; run inside the terminal under test")
	}
	in, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	out, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	log, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close() //nolint:errcheck
	note := func(format string, a ...any) {
		if _, err := fmt.Fprintf(log, format, a...); err != nil {
			t.Fatal(err)
		}
	}
	dll := windows.NewLazySystemDLL("kernel32.dll")
	k := kernel32{dll.NewProc("ReadConsoleInputW"), dll.NewProc("GetConsoleWindow")}
	var info windows.ConsoleScreenBufferInfo
	if err := k.bufferInfo(windows.Handle(out.Fd()), &info); err != nil {
		t.Fatal(err)
	}
	note("window %d %d %d %d\n", info.Window.Left, info.Window.Top, info.Window.Right, info.Window.Bottom)
	con, err := openConsole(recordingConsole{k, log}, windows.Handle(in.Fd()), windows.Handle(out.Fd()), Options{})
	if err != nil {
		t.Fatal(err)
	}
	o, err := offered(os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	b, err := enter(out, con, Options{}, o)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Exit() //nolint:errcheck
	if _, err := b.Write([]byte("\x1b[2J\x1b[5;5H[ left click here three times, right click once, then press q ]")); err != nil {
		t.Fatal(err)
	}
	note("entered %+v\n", b.Capabilities)
	deadline := time.After(time.Minute)
	for {
		select {
		case ev := <-b.Events:
			note("event %#v\n", ev)
			if k, ok := ev.(input.KeyEvent); ok && k.Rune == 'q' {
				return
			}
		case <-deadline:
			t.Fatal("no q within a minute")
		}
	}
}
