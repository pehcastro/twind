//go:build !windows

package main

import (
	"syscall"
	"time"
)

func cpuTime() time.Duration {
	var u syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &u); err != nil {
		panic(err)
	}
	return time.Duration(u.Utime.Nano() + u.Stime.Nano())
}
