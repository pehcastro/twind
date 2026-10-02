//go:build unix && !(darwin || dragonfly || freebsd || netbsd || openbsd)

package dev

import "golang.org/x/sys/unix"

const (
	ioctlGetTermios = unix.TCGETS
	ioctlSetTermios = unix.TCSETS
)
