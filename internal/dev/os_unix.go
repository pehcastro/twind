//go:build unix

package dev

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func ownGroup() *syscall.SysProcAttr {
	return nil
}

func askToExit(p *os.Process) error {
	return p.Signal(unix.SIGTERM)
}

func SaveConsole() (restore func() error, err error) {
	fd := int(os.Stdin.Fd())
	saved, err := unix.IoctlGetTermios(fd, ioctlGetTermios)
	if err != nil {
		return nil, err
	}
	return func() error { return unix.IoctlSetTermios(fd, ioctlSetTermios, saved) }, nil
}
