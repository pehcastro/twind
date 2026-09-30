package main

import (
	"time"
	"unsafe"

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

func cycles() uint64 {
	var n uint64
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("QueryProcessCycleTime")
	if ok, _, err := proc.Call(uintptr(windows.CurrentProcess()), uintptr(unsafe.Pointer(&n))); ok == 0 {
		panic(err)
	}
	return n
}

type memoryCounters struct {
	cb, pageFaults                   uint32
	peakWorkingSet, workingSet       uintptr
	peakPaged, paged, peakNonPaged   uintptr
	nonPaged, pagefile, peakPagefile uintptr
	private                          uintptr
}

func memory() (resident, private uint64) {
	c := memoryCounters{}
	c.cb = uint32(unsafe.Sizeof(c))
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("K32GetProcessMemoryInfo")
	if ok, _, err := proc.Call(uintptr(windows.CurrentProcess()), uintptr(unsafe.Pointer(&c)), uintptr(c.cb)); ok == 0 {
		panic(err)
	}
	return uint64(c.workingSet), uint64(c.private)
}
