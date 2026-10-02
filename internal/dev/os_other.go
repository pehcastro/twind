//go:build !unix && !windows

package dev

import (
	"context"
	"errors"
	"os"
	"syscall"
)

func notify(context.Context, string, chan<- struct{}) {}

func ownGroup() *syscall.SysProcAttr {
	return nil
}

func askToExit(*os.Process) error {
	return errors.ErrUnsupported
}

func SaveConsole() (restore func() error, err error) {
	return nil, errors.ErrUnsupported
}
