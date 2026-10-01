//go:build unix

package terminal

import (
	"errors"
	"io"
	"os"
	"os/signal"
	"time"

	"golang.org/x/sys/unix"

	"github.com/twind-dev/twind/twi/color"
)

func size(fd uintptr) (width, height int, err error) {
	ws, err := unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, err
	}
	return int(ws.Col), int(ws.Row), nil
}

func promised(uintptr) color.Profile {
	return color.None
}

func EnableVirtualTerminal(*os.File) (restore func() error, err error) {
	return func() error { return nil }, nil
}

const (
	wakeResize byte = 'r'
	wakeCancel byte = 'c'
)

type unixTTY struct {
	in      int
	out     uintptr
	saved   unix.Termios
	wake    [2]int
	winch   chan os.Signal
	relayed chan struct{}
}

func openTTY(in, out *os.File, _ Options) (tty, error) {
	fd := int(in.Fd())
	saved, err := unix.IoctlGetTermios(fd, ioctlGetTermios)
	if err != nil {
		return nil, err
	}
	t := &unixTTY{in: fd, out: out.Fd(), saved: *saved, winch: make(chan os.Signal, 1), relayed: make(chan struct{})}
	if err := unix.Pipe(t.wake[:]); err != nil {
		return nil, err
	}
	raw := *saved
	raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	raw.Oflag &^= unix.OPOST
	raw.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	raw.Cflag &^= unix.CSIZE | unix.PARENB
	raw.Cflag |= unix.CS8
	raw.Cc[unix.VMIN], raw.Cc[unix.VTIME] = 1, 0
	if err := unix.IoctlSetTermios(fd, ioctlSetTermios, &raw); err != nil {
		return nil, errors.Join(err, unix.Close(t.wake[0]), unix.Close(t.wake[1]))
	}
	signal.Notify(t.winch, unix.SIGWINCH)
	go func() {
		for range t.winch {
			_, _ = unix.Write(t.wake[1], []byte{wakeResize})
		}
		close(t.relayed)
	}()
	return t, nil
}

func (t *unixTTY) read(p []byte, wait time.Duration) (int, bool, error) {
	for {
		var fds unix.FdSet
		fds.Set(t.in)
		fds.Set(t.wake[0])
		var timeout *unix.Timeval
		if wait > 0 {
			tv := unix.NsecToTimeval(wait.Nanoseconds())
			timeout = &tv
		}
		ready, err := unix.Select(max(t.in, t.wake[0])+1, &fds, nil, nil, timeout)
		switch {
		case err == unix.EINTR:
			continue
		case err != nil:
			return 0, false, err
		case ready == 0:
			return 0, false, errQuiet
		case fds.IsSet(t.wake[0]):
			var wake [1]byte
			if _, err := unix.Read(t.wake[0], wake[:]); err != nil || wake[0] == wakeCancel {
				return 0, false, io.EOF
			}
			return 0, true, nil
		}
		n, err := unix.Read(t.in, p)
		switch {
		case err == unix.EINTR:
			continue
		case err != nil:
			return 0, false, err
		case n == 0:
			return 0, false, io.EOF
		}
		return n, false, nil
	}
}

func (t *unixTTY) size() (width, height int, err error) {
	return size(t.out)
}

func (t *unixTTY) conhost() bool {
	return false
}

func (t *unixTTY) font() Font {
	return Font{}
}

func (t *unixTTY) lacks(string, string) bool {
	return false
}

func (t *unixTTY) drawable() (window, error) {
	return nil, errNoWindow
}

func (t *unixTTY) cancel() {
	signal.Stop(t.winch)
	close(t.winch)
	<-t.relayed
	_, _ = unix.Write(t.wake[1], []byte{wakeCancel})
}

func (t *unixTTY) restore() error {
	return errors.Join(unix.IoctlSetTermios(t.in, ioctlSetTermios, &t.saved), unix.Close(t.wake[0]), unix.Close(t.wake[1]))
}
