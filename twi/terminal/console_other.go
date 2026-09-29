//go:build !unix && !windows

package terminal

import "os"

func size(uintptr) (width, height int, err error) {
	return 0, 0, errNotConsole
}

func EnableVirtualTerminal(*os.File) (restore func() error, err error) {
	return func() error { return nil }, nil
}
