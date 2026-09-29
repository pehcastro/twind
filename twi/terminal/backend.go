package terminal

import (
	"errors"
	"io"
	"os"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/input"
)

type Options struct{ Mouse bool }

type Capabilities struct {
	Sync          bool
	KittyKeyboard bool
}

type Backend struct {
	Events       <-chan input.Event
	Capabilities Capabilities

	out     io.Writer
	tty     tty
	opt     Options
	decoder input.Decoder
	replies chan input.ReplyEvent
	exited  bool
}

type tty interface {
	read(p []byte, quiet bool) (n int, resized bool, err error)
	size() (width, height int, err error)
	cancel()
	restore() error
}

var errQuiet = errors.New("terminal: no input within the escape timeout")

func Enter(in, out *os.File, opt Options) (*Backend, error) {
	t, err := openTTY(in, out, opt)
	if err != nil {
		return nil, err
	}
	return enter(out, t, opt)
}

func enter(out io.Writer, t tty, opt Options) (*Backend, error) {
	events := make(chan input.Event, konst.EventBuffer)
	b := &Backend{
		Events:  events,
		out:     out,
		tty:     t,
		opt:     opt,
		replies: make(chan input.ReplyEvent, konst.ReplyBuffer),
	}
	go b.read(events)
	seq := konst.EnterScreen
	if opt.Mouse {
		seq += konst.MouseOn
	}
	if _, err := b.Write([]byte(seq + konst.Queries)); err != nil {
		return nil, err
	}
	b.Capabilities = b.query()
	if b.Capabilities.KittyKeyboard {
		if _, err := b.Write([]byte(konst.KittyPush)); err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (b *Backend) query() Capabilities {
	var caps Capabilities
	timeout := time.After(konst.QueryTimeout)
	for {
		select {
		case <-timeout:
			return caps
		case r := <-b.replies:
			switch r.Kind {
			case input.ReplyPrimaryAttributes:
				return caps
			case input.ReplyMode:
				supported := len(r.Params) == 2 && (r.Params[1] == konst.ModeSet || r.Params[1] == konst.ModeReset)
				caps.Sync = caps.Sync || supported && r.Params[0] == konst.SyncMode
			case input.ReplyKeyboardFlags:
				caps.KittyKeyboard = true
			case input.ReplySecondaryAttributes, input.ReplyCursorPosition:
			}
		}
	}
}

func (b *Backend) read(events chan<- input.Event) {
	defer close(events)
	buf := make([]byte, konst.ReadBuffer)
	quiet := false
	width, height, _ := b.tty.size()
	for {
		n, resized, err := b.tty.read(buf, quiet)
		var evs []input.Event
		switch {
		case errors.Is(err, errQuiet):
			evs = b.decoder.Quiet()
		case err != nil:
			return
		default:
			evs = b.decoder.Decode(buf[:n])
		}
		quiet = n > 0
		if resized {
			w, h, err := b.tty.size()
			if err == nil && (w != width || h != height) {
				width, height = w, h
				evs = append(evs, input.ResizeEvent{Width: w, Height: h})
			}
		}
		for _, ev := range evs {
			if r, ok := ev.(input.ReplyEvent); ok {
				select {
				case b.replies <- r:
				default:
				}
				continue
			}
			events <- ev
		}
	}
}

func (b *Backend) Size() (width, height int, err error) {
	return b.tty.size()
}

func (b *Backend) Write(frame []byte) (int, error) {
	n, err := b.out.Write(frame)
	if err != nil {
		return n, errors.Join(err, b.Exit())
	}
	return n, nil
}

func (b *Backend) Exit() error {
	if b.exited {
		return nil
	}
	b.exited = true
	seq := konst.LeaveScreen
	if b.opt.Mouse {
		seq = konst.MouseOff + seq
	}
	if b.Capabilities.KittyKeyboard {
		seq = konst.KittyPop + seq
	}
	_, err := io.WriteString(b.out, seq)
	b.tty.cancel()
	for range b.Events {
	}
	return errors.Join(err, b.tty.restore())
}
