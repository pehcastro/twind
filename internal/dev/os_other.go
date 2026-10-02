//go:build !unix && !windows

package dev

import (
	"errors"
	"os"
	"syscall"
)

func ownGroup() *syscall.SysProcAttr {
	return nil
}

func askToExit(*os.Process) error {
	return errors.ErrUnsupported
}

func SaveConsole() (restore func() error, err error) {
	return nil, errors.ErrUnsupported
}
