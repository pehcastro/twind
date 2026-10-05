package dev

import (
	"context"
	"errors"
	"os"
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/pehcastro/twind/internal/dev/konst"
)

func ownGroup() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}
}

func askToExit(p *os.Process) error {
	return windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(p.Pid))
}

func Alive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer func() { _ = windows.CloseHandle(h) }()
	var code uint32
	return windows.GetExitCodeProcess(h, &code) == nil && code == konst.StillActive
}

func notify(ctx context.Context, dir string, wake chan<- struct{}) {
	name, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return
	}
	h, err := windows.CreateFile(name, windows.FILE_LIST_DIRECTORY, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return
	}
	stop := context.AfterFunc(ctx, func() { _ = windows.CancelIoEx(h, nil) })
	defer func() {
		stop()
		_ = windows.CloseHandle(h)
	}()
	buf := make([]byte, konst.NotifyBuffer)
	for ctx.Err() == nil {
		var n uint32
		if windows.ReadDirectoryChanges(h, &buf[0], uint32(len(buf)), true, windows.FILE_NOTIFY_CHANGE_FILE_NAME|windows.FILE_NOTIFY_CHANGE_SIZE|windows.FILE_NOTIFY_CHANGE_LAST_WRITE, &n, nil, 0) != nil {
			return
		}
		select {
		case wake <- struct{}{}:
		default:
		}
	}
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
