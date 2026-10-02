//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package dev

import "golang.org/x/sys/unix"

const (
	ioctlGetTermios = unix.TIOCGETA
	ioctlSetTermios = unix.TIOCSETA
)
