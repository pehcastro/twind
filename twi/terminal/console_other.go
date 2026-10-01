//go:build !unix && !windows

package terminal

import (
	"os"

	"github.com/twind-dev/twind/twi/color"
)

func size(uintptr) (width, height int, err error) {
	return 0, 0, errNotConsole
}

func promised(uintptr) color.Profile {
	return color.None
}

func EnableVirtualTerminal(*os.File) (restore func() error, err error) {
	return func() error { return nil }, nil
}

func openTTY(*os.File, *os.File, Options) (tty, error) {
	return nil, errNotConsole
}
