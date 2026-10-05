//go:build unix && !(darwin || dragonfly || freebsd || netbsd || openbsd)

package terminal

import "golang.org/x/sys/unix"

const (
	ioctlGetTermios = unix.TCGETS
	ioctlSetTermios = unix.TCSETS
)
