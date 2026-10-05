package dev

import (
	"errors"
	"os"
	"os/exec"
	"time"

	"github.com/pehcastro/twind/internal/dev/konst"
)

type process struct {
	cmd  *exec.Cmd
	done chan error
}

func Spawn(exe string, args, env []string, exited func() error) (Child, error) {
	cmd := exec.Command(exe, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = env
	cmd.SysProcAttr = ownGroup()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p := &process{cmd, make(chan error, 1)}
	go func() {
		err := cmd.Wait()
		p.done <- errors.Join(err, exited())
	}()
	return p, nil
}

func (p *process) Done() <-chan error { return p.done }

func (p *process) Stop() error {
	if askToExit(p.cmd.Process) == nil {
		select {
		case <-p.done:
			return nil
		case <-time.After(konst.StopGrace):
		}
	}
	err := p.cmd.Process.Kill()
	<-p.done
	if errors.Is(err, os.ErrProcessDone) {
		return nil
	}
	return err
}
