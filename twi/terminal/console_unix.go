//go:build unix

package terminal

import (
	"os"

	"golang.org/x/sys/unix"
)

func size(fd uintptr) (width, height int, err error) {
	ws, err := unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, err
	}
	return int(ws.Col), int(ws.Row), nil
}

func EnableVirtualTerminal(*os.File) (restore func() error, err error) {
	return func() error { return nil }, nil
}
