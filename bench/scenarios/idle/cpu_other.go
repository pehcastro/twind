//go:build !windows

package main

import (
	"syscall"
	"time"
)

func usage() syscall.Rusage {
	var u syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &u); err != nil {
		panic(err)
	}
	return u
}

func cpuTime() time.Duration {
	u := usage()
	return time.Duration(u.Utime.Nano() + u.Stime.Nano())
}

func cycles() uint64 { return 0 }

func memory() (resident, private uint64) {
	peak := uint64(usage().Maxrss) << 10
	return peak, peak
}
