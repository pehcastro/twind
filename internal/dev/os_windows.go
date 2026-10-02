package dev

import (
	"errors"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func ownGroup() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}
}

func askToExit(p *os.Process) error {
	return windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(p.Pid))
}

func SaveConsole() (restore func() error, err error) {
	in, out := windows.Handle(os.Stdin.Fd()), windows.Handle(os.Stdout.Fd())
	var inMode, outMode uint32
	if err := errors.Join(windows.GetConsoleMode(in, &inMode), windows.GetConsoleMode(out, &outMode)); err != nil {
		return nil, err
	}
	outMode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	if err := windows.SetConsoleMode(out, outMode); err != nil {
		return nil, err
	}
	return func() error {
		return errors.Join(windows.SetConsoleMode(in, inMode), windows.SetConsoleMode(out, outMode))
	}, nil
}
