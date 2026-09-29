//go:build windows

package terminal

import (
	"os"

	"golang.org/x/sys/windows"
)

func size(fd uintptr) (width, height int, err error) {
	var info windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(windows.Handle(fd), &info); err != nil {
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
