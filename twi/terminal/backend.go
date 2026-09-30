package terminal

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"slices"
	"strconv"
	"sync/atomic"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/input"
)

type Options struct{ Mouse bool }

type Graphics uint8

const (
	GraphicsNone Graphics = iota
	GraphicsSixel
	GraphicsITerm2
	GraphicsKitty
)

type Capabilities struct {
	Sync          bool
	KittyKeyboard bool
	Graphics      Graphics
	CellPixels    image.Point
}

type Backend struct {
	Events       <-chan input.Event
	Capabilities Capabilities

	out     io.Writer
	tty     tty
	opt     Options
	decoder input.Decoder
	answers chan answer
	asking  atomic.Bool
	resized atomic.Bool
	exited  bool
}

type answer struct {
	raw     []byte
	replies []input.ReplyEvent
}

type offer struct {
	graphics Graphics
	forced   bool
}

type tty interface {
	read(p []byte, quiet bool) (n int, resized bool, err error)
	size() (width, height int, err error)
	cancel()
	restore() error
}

var errQuiet = errors.New("terminal: no input within the escape timeout")

func Enter(in, out *os.File, opt Options) (*Backend, error) {
	o, err := offered(os.Getenv)
	if err != nil {
		return nil, err
	}
	t, err := openTTY(in, out, opt)
	if err != nil {
		return nil, err
	}
	return enter(out, t, opt, o)
}

func offered(env func(string) string) (offer, error) {
	switch v := env("TWIND_GRAPHICS"); v {
	case "":
	case "none":
		return offer{GraphicsNone, true}, nil
	case "sixel":
		return offer{GraphicsSixel, true}, nil
	case "iterm2":
		return offer{GraphicsITerm2, true}, nil
	case "kitty":
		return offer{GraphicsKitty, true}, nil
	default:
		return offer{}, fmt.Errorf("terminal: TWIND_GRAPHICS=%q, want none, sixel, iterm2 or kitty", v)
	}
	if p := env("TERM_PROGRAM"); p == "iTerm.app" || p == "WezTerm" || env("LC_TERMINAL") == "iTerm2" {
		return offer{graphics: GraphicsITerm2}, nil
	}
	return offer{}, nil
}

func enter(out io.Writer, t tty, opt Options, o offer) (*Backend, error) {
	events := make(chan input.Event, konst.EventBuffer)
	b := &Backend{
		Events:  events,
		out:     out,
		tty:     t,
		opt:     opt,
		answers: make(chan answer, konst.ReplyBuffer),
	}
	go b.read(events)
	seq := konst.EnterScreen
	if opt.Mouse {
		seq += konst.MouseOn
	}
	raw, replies, err := b.ask(seq + konst.Queries)
	if err != nil {
		return nil, err
	}
	b.Capabilities = b.detect(raw, replies, o)
	if b.Capabilities.KittyKeyboard {
		if _, err := b.Write([]byte(konst.KittyPush)); err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (b *Backend) ask(seq string) ([]byte, []input.ReplyEvent, error) {
	for len(b.answers) > 0 {
		<-b.answers
	}
	b.asking.Store(true)
	defer b.asking.Store(false)
	if _, err := io.WriteString(b.out, seq); err != nil {
		return nil, nil, errors.Join(err, b.Exit())
	}
	var raw []byte
	var replies []input.ReplyEvent
	timeout := time.After(konst.QueryTimeout)
	for {
		select {
		case <-timeout:
			return raw, replies, nil
		case a := <-b.answers:
			raw = append(raw, a.raw...)
			replies = append(replies, a.replies...)
			if slices.ContainsFunc(a.replies, func(r input.ReplyEvent) bool { return r.Kind == input.ReplyPrimaryAttributes }) {
				return raw, replies, nil
			}
		}
	}
}

func (b *Backend) detect(raw []byte, replies []input.ReplyEvent, o offer) Capabilities {
	caps := Capabilities{CellPixels: b.cellPixels(raw)}
	graphics := o.graphics
	for _, r := range replies {
		switch r.Kind {
		case input.ReplyMode:
			supported := len(r.Params) == 2 && (r.Params[1] == konst.ModeSet || r.Params[1] == konst.ModeReset)
			caps.Sync = caps.Sync || supported && r.Params[0] == konst.SyncMode
		case input.ReplyKeyboardFlags:
			caps.KittyKeyboard = true
		case input.ReplyPrimaryAttributes:
			if slices.Index(r.Params, konst.SixelAttribute) > 0 {
				graphics = max(graphics, GraphicsSixel)
			}
		case input.ReplySecondaryAttributes, input.ReplyCursorPosition:
		}
	}
	if bytes.Contains(raw, []byte(konst.KittyOK)) {
		graphics = GraphicsKitty
	}
	if o.forced {
		graphics = o.graphics
	}
	if caps.CellPixels != (image.Point{}) {
		caps.Graphics = graphics
	}
	return caps
}

func (b *Backend) cellPixels(raw []byte) image.Point {
	cell := pixels(raw, konst.CellReport)
	if w, h, err := b.tty.size(); cell == (image.Point{}) && err == nil && w > 0 && h > 0 {
		window := pixels(raw, konst.WindowReport)
		cell = image.Pt(window.X/w, window.Y/h)
	}
	if cell.X == 0 || cell.Y == 0 {
		return image.Point{}
	}
	return cell
}

func pixels(raw []byte, report string) image.Point {
	prefix := []byte(konst.CSI + report + ";")
	for {
		i := bytes.Index(raw, prefix)
		if i < 0 {
			return image.Point{}
		}
		raw = raw[i+len(prefix):]
		body, _, _ := bytes.Cut(raw, []byte("t"))
		hs, ws, _ := bytes.Cut(body, []byte(";"))
		h, errH := strconv.Atoi(string(hs))
		w, errW := strconv.Atoi(string(ws))
		if errH == nil && errW == nil && w > 0 && h > 0 {
			return image.Pt(w, h)
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
				b.resized.Store(true)
				evs = append(evs, input.ResizeEvent{Width: w, Height: h})
			}
		}
		var replies []input.ReplyEvent
		for _, ev := range evs {
			if r, ok := ev.(input.ReplyEvent); ok {
				replies = append(replies, r)
				continue
			}
			events <- ev
		}
		if b.asking.Load() && (n > 0 || len(replies) > 0) {
			select {
			case b.answers <- answer{bytes.Clone(buf[:n]), replies}:
			default:
			}
		}
	}
}

func (b *Backend) Size() (width, height int, err error) {
	return b.tty.size()
}

func (b *Backend) Write(frame []byte) (int, error) {
	if b.resized.Swap(false) && b.Capabilities.Graphics != GraphicsNone {
		raw, _, err := b.ask(konst.CellQuery)
		if err != nil {
			return 0, err
		}
		if cell := b.cellPixels(raw); cell != (image.Point{}) {
			b.Capabilities.CellPixels = cell
		}
	}
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
