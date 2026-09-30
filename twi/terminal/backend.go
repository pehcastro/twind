package terminal

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/text"
)

type Options struct{ NoMouse bool }

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
	Graphemes     bool
	Margins       bool
	Focus         bool
	InBandResize  bool
	Widths        text.Widths
}

type Backend struct {
	Events       <-chan input.Event
	Capabilities Capabilities

	out     io.Writer
	tty     tty
	opt     Options
	decoder input.Decoder
	answers chan answer
	leave   string
	asking  atomic.Bool
	cell    atomic.Pointer[image.Point]
	polling atomic.Bool
	settled chan struct{}
	mu      sync.Mutex
	asked   bool
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
	read(p []byte, wait time.Duration) (n int, recheck bool, err error)
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

func Query(in, out *os.File) (Capabilities, image.Point, error) {
	o, err := offered(os.Getenv)
	if err != nil || o == (offer{GraphicsNone, true}) {
		return Capabilities{}, image.Point{}, err
	}
	t, err := openTTY(in, out, Options{NoMouse: true})
	if err != nil {
		return Capabilities{}, image.Point{}, nil
	}
	return query(out, t, o)
}

func Probe(in, out *os.File, queries string) ([]byte, error) {
	t, err := openTTY(in, out, Options{NoMouse: true})
	if err != nil {
		return nil, err
	}
	_, raw, _, err := probe(out, t, queries+konst.Fence)
	return raw, err
}

func query(out io.Writer, t tty, o offer) (Capabilities, image.Point, error) {
	b, raw, replies, err := probe(out, t, konst.Probes+konst.InlineQueries)
	if b == nil {
		return Capabilities{}, image.Point{}, err
	}
	for _, r := range replies {
		if r.Kind == input.ReplyCursorPosition && len(r.Params) == 2 {
			return b.detect(raw, replies, o), image.Pt(r.Params[1]-1, r.Params[0]-1), err
		}
	}
	return Capabilities{}, image.Point{}, err
}

func probe(out io.Writer, t tty, queries string) (*Backend, []byte, []input.ReplyEvent, error) {
	events := make(chan input.Event, konst.EventBuffer)
	b := &Backend{Events: events, out: out, tty: t, opt: Options{NoMouse: true}, answers: make(chan answer, konst.ReplyBuffer)}
	go b.read(events)
	raw, replies, err := b.ask(queries)
	if err != nil {
		return nil, nil, nil, err
	}
	return b, raw, replies, b.Exit()
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
		leave:   konst.LeaveScreen,
		settled: make(chan struct{}),
	}
	go b.read(events)
	seq := konst.EnterScreen
	if !opt.NoMouse {
		seq += konst.MouseOn
	}
	raw, replies, err := b.ask(seq + konst.CursorHome + konst.GraphemesOn + konst.Probes + konst.Queries)
	if err != nil {
		return nil, err
	}
	b.Capabilities = b.detect(raw, replies, o)
	if b.Capabilities.Graphics != GraphicsNone {
		cell := b.Capabilities.CellPixels
		b.cell.Store(&cell)
		b.polling.Store(!b.Capabilities.InBandResize)
	}
	seq = ""
	if b.Capabilities.KittyKeyboard {
		seq += konst.KittyPush
	}
	if b.Capabilities.InBandResize {
		seq += konst.InBandOn
	}
	if seq != "" {
		if _, err := b.Write([]byte(seq)); err != nil {
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
	if _, err := b.Write([]byte(seq)); err != nil {
		return nil, nil, err
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
	caps := Capabilities{CellPixels: b.cellPixels(replies)}
	graphics := o.graphics
	var cursors [][]int
	for _, r := range replies {
		switch r.Kind {
		case input.ReplyMode:
			if len(r.Params) != 2 {
				continue
			}
			mode, state := r.Params[0], r.Params[1]
			caps.Sync = caps.Sync || mode == konst.SyncMode && (state == konst.ModeSet || state == konst.ModeReset)
			caps.Graphemes = caps.Graphemes || mode == konst.GraphemeMode && state >= konst.ModeSet && state <= konst.ModeKeptSet
			caps.Margins = caps.Margins || mode == konst.MarginMode && state >= konst.ModeSet && state <= konst.ModeKeptSet
			caps.Focus = caps.Focus || mode == konst.FocusMode && state >= konst.ModeSet && state <= konst.ModeKeptSet
			caps.InBandResize = caps.InBandResize || mode == konst.InBandMode && state >= konst.ModeSet && state <= konst.ModeKeptSet
		case input.ReplyKeyboardFlags:
			caps.KittyKeyboard = true
		case input.ReplyPrimaryAttributes:
			if slices.Index(r.Params, konst.SixelAttribute) > 0 {
				graphics = max(graphics, GraphicsSixel)
			}
		case input.ReplyCursorPosition:
			if len(r.Params) == 2 {
				cursors = append(cursors, r.Params)
			}
		case input.ReplySecondaryAttributes, input.ReplyWindow:
		}
	}
	if columns, _, err := b.tty.size(); err == nil && len(cursors) > int(text.Classes) {
		origin := cursors[0]
		for c := range text.Classes {
			end := cursors[c+1]
			if advance := end[1] - origin[1]; end[0] == origin[0] && advance > 0 && end[1] < columns {
				caps.Widths[c] = advance
			}
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

func (b *Backend) cellPixels(replies []input.ReplyEvent) image.Point {
	var cell, window image.Point
	for _, r := range replies {
		if r.Kind != input.ReplyWindow || len(r.Params) != konst.WindowParams || r.Params[1] <= 0 || r.Params[2] <= 0 {
			continue
		}
		switch size := image.Pt(r.Params[2], r.Params[1]); r.Params[0] {
		case konst.CellReport:
			cell = size
		case konst.WindowReport:
			window = size
		}
	}
	if w, h, err := b.tty.size(); cell == (image.Point{}) && err == nil && w > 0 && h > 0 {
		cell = image.Pt(window.X/w, window.Y/h)
	}
	if cell.X == 0 || cell.Y == 0 {
		return image.Point{}
	}
	return cell
}

func (b *Backend) read(events chan<- input.Event) {
	defer close(events)
	buf := make([]byte, konst.ReadBuffer)
	quiet := false
	width, height, _ := b.tty.size()
	var polled []input.ReplyEvent
	lastAsk := time.Now()
	for {
		var wait time.Duration
		if quiet {
			wait = konst.EscapeTimeout
		}
		n, recheck, err := b.tty.read(buf, wait)
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
		if recheck {
			w, h, err := b.tty.size()
			if err == nil && (w != width || h != height) {
				width, height = w, h
				evs = append(evs, input.ResizeEvent{Width: w, Height: h})
			}
		}
		if recheck || slices.ContainsFunc(evs, func(ev input.Event) bool { return caused(ev, time.Since(lastAsk)) }) {
			lastAsk = time.Now()
			b.askCell()
		}
		var replies []input.ReplyEvent
		for _, ev := range evs {
			switch ev := ev.(type) {
			case input.ReplyEvent:
				replies = append(replies, ev)
				switch ev.Kind {
				case input.ReplyWindow:
					polled = append(polled, ev)
				case input.ReplyPrimaryAttributes:
					if cell := b.cellPixels(polled); b.settle() && b.learn(cell) {
						events <- input.ResizeEvent{Width: width, Height: height, Cell: cell}
					}
					polled = polled[:0]
				case input.ReplySecondaryAttributes, input.ReplyMode, input.ReplyCursorPosition, input.ReplyKeyboardFlags:
				}
				continue
			case input.ResizeEvent:
				width, height = ev.Width, ev.Height
				b.learn(ev.Cell)
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

func caused(ev input.Event, since time.Duration) bool {
	switch ev := ev.(type) {
	case input.FocusEvent:
		return ev.Focused
	case input.KeyEvent, input.MouseEvent, input.PasteEvent:
		return since >= konst.CellPoll
	case input.ResizeEvent, input.ReplyEvent:
		return false
	}
	panic(fmt.Sprintf("terminal: unknown event %T", ev))
}

func (b *Backend) askCell() {
	if !b.polling.Load() {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.asked || b.exited {
		return
	}
	_, err := io.WriteString(b.out, konst.CellQuery)
	b.asked = err == nil
}

func (b *Backend) settle() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	asked := b.asked
	b.asked = false
	if asked && b.exited {
		close(b.settled)
	}
	return asked
}

func (b *Backend) learn(cell image.Point) bool {
	last := b.cell.Load()
	if last == nil || cell == (image.Point{}) || cell == *last {
		return false
	}
	b.cell.Store(&cell)
	return true
}

func (b *Backend) Size() (width, height int, err error) {
	return b.tty.size()
}

func (b *Backend) Write(frame []byte) (int, error) {
	if cell := b.cell.Load(); cell != nil {
		b.Capabilities.CellPixels = *cell
	}
	b.mu.Lock()
	n, err := b.out.Write(frame)
	b.mu.Unlock()
	if err != nil {
		return n, errors.Join(err, b.Exit())
	}
	return n, nil
}

func (b *Backend) Exit() error {
	b.mu.Lock()
	exited, asked := b.exited, b.asked
	b.exited = true
	b.mu.Unlock()
	if exited {
		return nil
	}
	if asked {
		select {
		case <-b.settled:
		case <-time.After(konst.QueryTimeout):
		}
	}
	seq := b.leave
	if b.Capabilities.InBandResize {
		seq = konst.InBandOff + seq
	}
	if !b.opt.NoMouse {
		seq = konst.MouseOff + seq
	}
	if b.Capabilities.KittyKeyboard {
		seq = konst.KittyPop + seq
	}
	if b.Capabilities.Graphemes {
		seq = konst.GraphemesOff + seq
	}
	_, err := io.WriteString(b.out, seq)
	b.tty.cancel()
	for range b.Events {
	}
	return errors.Join(err, b.tty.restore())
}
