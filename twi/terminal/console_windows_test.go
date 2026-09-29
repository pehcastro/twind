//go:build windows

package terminal

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

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
