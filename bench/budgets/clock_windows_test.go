package budgets_test

import (
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func cycles() uint64 {
	var n uint64
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("QueryProcessCycleTime")
	_, _, _ = proc.Call(uintptr(windows.CurrentProcess()), uintptr(unsafe.Pointer(&n)))
	return n
}

func clock(b *testing.B) func() time.Duration {
	kernel := windows.NewLazySystemDLL("kernel32.dll")
	counter := kernel.NewProc("QueryPerformanceCounter").Addr()
	var perSecond int64
	if ok, _, err := syscall.SyscallN(kernel.NewProc("QueryPerformanceFrequency").Addr(), uintptr(unsafe.Pointer(&perSecond))); ok == 0 {
		b.Fatal(err)
	}
	return func() time.Duration {
		var n int64
		_, _, _ = syscall.SyscallN(counter, uintptr(unsafe.Pointer(&n)))
		return time.Duration(n/perSecond)*time.Second + time.Duration(n%perSecond)*time.Second/time.Duration(perSecond)
	}
}
