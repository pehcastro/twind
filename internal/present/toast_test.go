package present_test

import (
	"bytes"
	"image"
	"strings"
	"sync"
	"testing"
	"time"

	docsapp "github.com/twind-dev/twind/apps/documentation"
	"github.com/twind-dev/twind/internal/present"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/terminal"
)

var toastGrid = image.Pt(187, 40)

type shownBackend struct {
	events  chan input.Event
	caps    terminal.Capabilities
	mu      sync.Mutex
	model   *present.Xterm
	stream  []byte
	surface []byte
	frames  int
}

func (b *shownBackend) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.frames++
	if b.model != nil {
		b.stream = append(b.stream, p...)
	}
	return len(p), nil
}

func (b *shownBackend) Paint(p terminal.Pixels) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.frames++
	stride := p.Grid.X * p.Cell.X * 4
	if p.Clear || b.surface == nil {
		b.surface = make([]byte, stride*p.Grid.Y*p.Cell.Y)
	}
	for _, t := range p.Tiles {
		r := image.Rect(t.Cells.Min.X*p.Cell.X, t.Cells.Min.Y*p.Cell.Y, t.Cells.Max.X*p.Cell.X, t.Cells.Max.Y*p.Cell.Y)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			row := b.surface[y*stride+r.Min.X*4 : y*stride+r.Max.X*4]
			if t.Pix == nil {
				clear(row)
				continue
			}
			copy(row, t.Pix[(y-r.Min.Y)*r.Dx()*4:])
		}
	}
	return true
}

func (b *shownBackend) Events() <-chan input.Event           { return b.events }
func (b *shownBackend) Sync() bool                           { return true }
func (b *shownBackend) Exit() error                          { return nil }
func (b *shownBackend) Size() (width, height int, err error) { return toastGrid.X, toastGrid.Y, nil }
func (b *shownBackend) Capabilities() terminal.Capabilities  { return b.caps }

type wallClock struct{}

func (wallClock) Now() time.Time                         { return time.Now() }
func (wallClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

func (b *shownBackend) settle() {
	for last := -1; ; {
		time.Sleep(150 * time.Millisecond)
		b.mu.Lock()
		n := b.frames
		b.mu.Unlock()
		if n == last {
			return
		}
		last = n
	}
}

func (b *shownBackend) region(t *testing.T, r image.Rectangle) []byte {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	cell, pix := b.caps.CellPixels, b.surface
	if b.model != nil {
		b.model.Feed(t, b.stream)
		b.stream, pix = b.stream[:0], b.model.Pixels(t)
	}
	stride := toastGrid.X * cell.X * 4
	var out []byte
	for y := r.Min.Y * cell.Y; y < r.Max.Y*cell.Y; y++ {
		out = append(out, pix[y*stride+r.Min.X*cell.X*4:y*stride+r.Max.X*cell.X*4]...)
	}
	return out
}

func toastSpots(t *testing.T, sheet style.Sheet) (success, undo image.Point) {
	t.Helper()
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		view, err := docsapp.New(rt, docsapp.Start{Page: "toaster", Theme: "twind-dark"})
		if err != nil {
			t.Fatal(err)
		}
		return view
	}, drive.Size(toastGrid.X, toastGrid.Y), drive.Styles(sheet))
	defer func() { _ = d.Close() }()
	find := func(word string) image.Point {
		for y, line := range strings.Split(d.Frame().Text(), "\n") {
			if x := strings.Index(line, word); x >= 0 {
				return image.Pt(len([]rune(line[:x]))+1, y)
			}
		}
		t.Fatalf("no %q on the Toaster page:\n%s", word, d.Frame().Text())
		return image.Point{}
	}
	success = find("Success")
	d.Click(success.X, success.Y)
	return success, find("Undo")
}

func TestClosedToastLeavesNoPixels(t *testing.T) {
	sheet, err := docsapp.Styles()
	if err != nil {
		t.Fatal(err)
	}
	success, undo := toastSpots(t, sheet)
	gdi, vscode := image.Pt(8, 17), image.Pt(7, 17)
	for _, caps := range []terminal.Capabilities{
		{Identity: terminal.IdentityZed, Sync: true, Graphics: terminal.GraphicsGDI, CellPixels: gdi},
		{Identity: terminal.IdentityConhost, Sync: true, Graphics: terminal.GraphicsGDI, CellPixels: gdi},
		{Identity: terminal.IdentityVSCode, Sync: true, Graphics: terminal.GraphicsSixel, CellPixels: vscode},
	} {
		b := &shownBackend{events: make(chan input.Event), caps: caps}
		if caps.Graphics == terminal.GraphicsSixel {
			b.model = present.NewXterm(toastGrid, caps.CellPixels)
		}
		rt := twi.New(twi.Backend(b, wallClock{}), twi.Styles(sheet), twi.ColorProfile(color.TrueColor))
		view, err := docsapp.New(rt, docsapp.Start{Page: "toaster", Theme: "twind-dark"})
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- rt.Run(view) }()
		b.settle()
		corner := image.Rect(120, 24, 187, 40)
		before := b.region(t, corner)
		send := func(events ...input.Event) {
			for _, e := range events {
				b.events <- e
			}
			b.settle()
		}
		move := func(x, y int) input.Event {
			return input.MouseEvent{X: x, Y: y, Button: input.MouseNone, Action: input.MouseMove}
		}
		for range 3 {
			send(input.MouseEvent{X: success.X, Y: success.Y, Button: input.MouseLeft, Action: input.MousePress}, input.MouseEvent{X: success.X, Y: success.Y, Button: input.MouseLeft, Action: input.MouseRelease})
		}
		if bytes.Equal(b.region(t, corner), before) {
			t.Fatalf("identity %d: no toast drawn in cells %v", caps.Identity, corner)
		}
		send(move(undo.X-10, undo.Y))
		send(move(undo.X, undo.Y-2))
		send(move(undo.X-10, undo.Y-5))
		send(move(success.X-40, success.Y-5))
		time.Sleep(4500 * time.Millisecond)
		b.settle()
		after, width := b.region(t, corner), corner.Dx()*caps.CellPixels.X
		var stale image.Rectangle
		for i := 0; i < len(after); i += 4 {
			if !bytes.Equal(after[i:i+4], before[i:i+4]) {
				x, y := corner.Min.X*caps.CellPixels.X+i/4%width, corner.Min.Y*caps.CellPixels.Y+i/4/width
				stale = stale.Union(image.Rect(x/caps.CellPixels.X, y/caps.CellPixels.Y, x/caps.CellPixels.X+1, y/caps.CellPixels.Y+1))
			}
		}
		if !stale.Empty() {
			t.Errorf("identity %d: cells %v still show pixels after every toast closed", caps.Identity, stale)
		}
		rt.Quit()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}
