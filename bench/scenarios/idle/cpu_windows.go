package main

import (
	"time"

	"golang.org/x/sys/windows"
)

func cpuTime() time.Duration {
	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(windows.CurrentProcess(), &creation, &exit, &kernel, &user); err != nil {
		panic(err)
	}
	ticks := func(f windows.Filetime) time.Duration {
		return time.Duration(uint64(f.HighDateTime)<<32|uint64(f.LowDateTime)) * 100
	}
	return ticks(kernel) + ticks(user)
}
