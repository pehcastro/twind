package terminal

import (
	"errors"
	"image"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/input"
)

type fakeWindow struct {
	mu          sync.Mutex
	size        image.Point
	screen      []byte
	blits       []image.Rectangle
	invalidated []image.Rectangle
	released    bool
	refused     bool
}

func (w *fakeWindow) client() (image.Point, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.size == (image.Point{}) {
		return image.Point{}, errors.New("no client area")
	}
	return w.size, nil
}

func (w *fakeWindow) sized(size image.Point) {
	if len(w.screen) != size.X*size.Y*4 {
		w.screen = make([]byte, size.X*size.Y*4)
	}
}

func (w *fakeWindow) blit(pix []byte, size image.Point, r image.Rectangle) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.sized(size)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if i := (y*size.X + x) * 4; pix[i+3] == 255 {
				copy(w.screen[i:i+4], pix[i:i+4])
			}
		}
	}
	w.blits = append(w.blits, r)
}

func (w *fakeWindow) read(size image.Point, _ image.Rectangle) []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.refused {
		return nil
	}
	w.sized(size)
	return slices.Clone(w.screen)
}

func (w *fakeWindow) invalidate(r image.Rectangle) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.invalidated = append(w.invalidated, r)
}

func (w *fakeWindow) release() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.released = true
}

func (w *fakeWindow) repaint(r image.Rectangle) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			i := (y*w.size.X + x) * 4
			copy(w.screen[i:i+4], []byte{1, 2, 3, 0})
		}
	}
}

func (w *fakeWindow) counts() (blits, invalidated int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.blits), len(w.invalidated)
}

func conhostFake(win *fakeWindow) *fakeTerminal {
	term := newFake(conPTY...)
	term.tty.window, term.tty.win = true, win
	return term
}

func opaque(cells image.Rectangle, cell image.Point) []byte {
	pix := make([]byte, cells.Dx()*cell.X*cells.Dy()*cell.Y*4)
	for i := 0; i < len(pix); i += 4 {
		copy(pix[i:], []byte{9, 8, 7, 255})
	}
	return pix
}

func TestGDIChosenForConhostOnly(t *testing.T) {
	cases := []struct {
		name     string
		answers  []string
		conhost  bool
		win      *fakeWindow
		offer    offer
		graphics Graphics
		cell     image.Point
	}{
		{"conhost with a window", conPTY, true, &fakeWindow{size: image.Pt(800, 480)}, offer{}, GraphicsGDI, image.Pt(10, 20)},
		{"conhost with slack under the grid", conPTY, true, &fakeWindow{size: image.Pt(807, 495)}, offer{}, GraphicsGDI, image.Pt(10, 20)},
		{"conhost without a window", conPTY, true, nil, offer{}, GraphicsNone, image.Point{}},
		{"conhost minimised", conPTY, true, &fakeWindow{}, offer{}, GraphicsNone, image.Point{}},
		{"conhost refusing GDI objects", conPTY, true, &fakeWindow{size: image.Pt(800, 480), refused: true}, offer{}, GraphicsNone, image.Point{}},
		{"conhost window smaller than the grid", conPTY, true, &fakeWindow{size: image.Pt(60, 20)}, offer{}, GraphicsNone, image.Point{}},
		{"conhost forced none", conPTY, true, &fakeWindow{size: image.Pt(800, 480)}, offer{forced: true}, GraphicsNone, image.Point{}},
		{"conhost forced sixel", conPTY, true, &fakeWindow{size: image.Pt(800, 480)}, offer{graphics: GraphicsSixel, forced: true}, GraphicsNone, image.Point{}},
		{"windows terminal offering a window", windowsTerminal, false, &fakeWindow{size: image.Pt(800, 480)}, offer{}, GraphicsSixel, image.Pt(10, 20)},
		{"inbox conpty offering a window", conPTY, false, &fakeWindow{size: image.Pt(800, 480)}, offer{}, GraphicsNone, image.Point{}},
		{"zed offering a window", zed, false, &fakeWindow{size: image.Pt(800, 480)}, offer{zed: true}, GraphicsNone, image.Pt(7, 16)},
	}
	for _, tc := range cases {
		term := newFake(tc.answers...)
		term.tty.window, term.tty.win = tc.conhost, tc.win
		b, err := enter(term, term.tty, Options{}, tc.offer)
		if err != nil {
			t.Fatal(err)
		}
		if b.Capabilities.Graphics != tc.graphics || b.Capabilities.CellPixels != tc.cell {
			t.Errorf("%s: graphics %d cell %v, want %d %v", tc.name, b.Capabilities.Graphics, b.Capabilities.CellPixels, tc.graphics, tc.cell)
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
		if tc.win != nil && tc.graphics != GraphicsGDI {
			if blits, invalidated := tc.win.counts(); blits+invalidated > 0 || tc.win.released != (tc.conhost && !tc.offer.forced) {
				t.Errorf("%s: an unused window was drawn on %d times or released %v", tc.name, blits+invalidated, tc.win.released)
			}
		}
		term = newFake(append([]string{"\x1b[3;1R"}, tc.answers...)...)
		term.tty.window, term.tty.win = tc.conhost, tc.win
		caps, _, err := query(term, term.tty, tc.offer)
		if err != nil {
			t.Fatal(err)
		}
		if caps.Graphics == GraphicsGDI {
			t.Errorf("%s: query reported GDI", tc.name)
		}
	}
}

func TestGDIRedrawsWhatTheConsoleRepainted(t *testing.T) {
	win := &fakeWindow{size: image.Pt(800, 480)}
	term := conhostFake(win)
	b, err := enter(term, term.tty, Options{}, offer{})
	if err != nil {
		t.Fatal(err)
	}
	cell := b.Capabilities.CellPixels
	ring := image.Rect(1, 1, 9, 2)
	grid := image.Pt(80, 24)
	b.Paint(Pixels{Cell: cell, Grid: grid, Clear: true, Tiles: []Tile{{Cells: ring, Pix: opaque(ring, cell)}}})
	blits, _ := win.counts()
	if blits != 1 {
		t.Fatalf("painting one tile blitted %d times, want 1", blits)
	}
	c := b.canvas.Load()
	if _, redraw := c.refit(80, 24, time.Now().Add(time.Hour)); redraw {
		t.Fatalf("an unchanged window asked for a redraw")
	}
	c.check()
	if now, _ := win.counts(); now != blits {
		t.Errorf("nothing repainted, yet the check drew %d times", now-blits)
	}
	win.repaint(image.Rect(30, 20, 40, 40))
	c.check()
	if now, _ := win.counts(); now != blits+1 {
		t.Errorf("after the console repainted part of the ring the check drew %d times, want 1", now-blits)
	}
	if i := (30*800 + 35) * 4; !slices.Equal(win.screen[i:i+4], []byte{9, 8, 7, 255}) {
		t.Errorf("repainted pixel is %v after the check, want the ring", win.screen[i:i+4])
	}
	b.Paint(Pixels{Cell: cell, Grid: grid, Tiles: []Tile{{Cells: ring}}})
	if _, invalidated := win.counts(); win.invalidated[invalidated-1] != image.Rect(10, 20, 90, 40) {
		t.Errorf("a cleared tile invalidated %v, want its pixels %v", win.invalidated[invalidated-1], image.Rect(10, 20, 90, 40))
	}
	blits, _ = win.counts()
	b.Paint(Pixels{Cell: image.Pt(9, 18), Grid: grid, Tiles: []Tile{{Cells: ring, Pix: opaque(ring, image.Pt(9, 18))}}})
	if now, _ := win.counts(); now != blits {
		t.Errorf("pixels for an old cell size were drawn")
	}
	if err := b.Exit(); err != nil {
		t.Fatal(err)
	}
	b.Paint(Pixels{Cell: cell, Tiles: []Tile{{Cells: ring, Pix: opaque(ring, cell)}}})
	if now, _ := win.counts(); now != blits || !win.released {
		t.Errorf("after Exit: %d more blits, released %v, want none and released", now-blits, win.released)
	}
}

func (w *fakeWindow) resize(size image.Point) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.size = size
}

func (w *fakeWindow) last() (blit, invalidated image.Rectangle) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.blits[len(w.blits)-1], w.invalidated[len(w.invalidated)-1]
}

func TestGDIGrowAfterShrinkDrawsOnlyForTheNewSize(t *testing.T) {
	win := &fakeWindow{size: image.Pt(800, 480)}
	term := conhostFake(win)
	term.later = conPTY
	b, err := enter(term, term.tty, Options{}, offer{})
	if err != nil {
		t.Fatal(err)
	}
	cell, ring := image.Pt(10, 20), image.Rect(1, 1, 9, 2)
	tiles := []Tile{{Cells: ring, Pix: opaque(ring, cell)}}
	if !b.Paint(Pixels{Cell: cell, Grid: image.Pt(80, 24), Clear: true, Tiles: tiles}) {
		t.Fatalf("the first frame was refused")
	}
	for _, step := range []struct {
		name       string
		client     image.Point
		grid, from image.Point
	}{
		{"shrink", image.Pt(600, 360), image.Pt(60, 18), image.Pt(80, 24)},
		{"grow back", image.Pt(800, 480), image.Pt(80, 24), image.Pt(60, 18)},
	} {
		win.resize(step.client)
		blits, _ := win.counts()
		if b.Paint(Pixels{Cell: cell, Grid: step.from, Tiles: tiles}) {
			t.Errorf("%s: a frame for the old size was drawn after the client changed", step.name)
		}
		b.canvas.Load().check()
		if now, _ := win.counts(); now != blits {
			t.Errorf("%s: %d blits while the window no longer matches the canvas, want 0", step.name, now-blits)
		}
		if _, invalidated := win.last(); invalidated != (image.Rectangle{Max: step.client}) {
			t.Errorf("%s: invalidated %v, want the whole new client %v", step.name, invalidated, image.Rectangle{Max: step.client})
		}
		term.tty.resize <- step.grid
		events(t, b, step.name, input.ResizeEvent{Width: step.grid.X, Height: step.grid.Y}, input.ResizeEvent{Width: step.grid.X, Height: step.grid.Y})
		if b.Paint(Pixels{Cell: cell, Grid: step.from, Clear: true, Tiles: tiles}) {
			t.Errorf("%s: a full frame laid out for the old grid was drawn", step.name)
		}
		if now, _ := win.counts(); now != blits {
			t.Errorf("%s: %d blits before a frame for the new size, want 0", step.name, now-blits)
		}
		if !b.Paint(Pixels{Cell: cell, Grid: step.grid, Clear: true, Tiles: tiles}) {
			t.Fatalf("%s: the first frame for the new size was refused", step.name)
		}
		if blit, _ := win.last(); blit != image.Rect(10, 20, 90, 40) || len(win.screen) != step.client.X*step.client.Y*4 {
			t.Errorf("%s: drew %v on a %d byte window, want %v on the new client", step.name, blit, len(win.screen), image.Rect(10, 20, 90, 40))
		}
	}
	if err := b.Exit(); err != nil {
		t.Fatal(err)
	}
}

func TestGDIFontChangeSendsTheNewCell(t *testing.T) {
	win := &fakeWindow{size: image.Pt(800, 480)}
	term := conhostFake(win)
	b, err := enter(term, term.tty, Options{}, offer{})
	if err != nil {
		t.Fatal(err)
	}
	ring := image.Rect(0, 0, 8, 1)
	b.Paint(Pixels{Cell: image.Pt(10, 20), Grid: image.Pt(80, 24), Clear: true, Tiles: []Tile{{Cells: ring, Pix: opaque(ring, image.Pt(10, 20))}}})
	win.mu.Lock()
	win.size = image.Pt(960, 576)
	win.mu.Unlock()
	events(t, b, "font change", input.ResizeEvent{Width: 80, Height: 24, Cell: image.Pt(12, 24)})
	if _, invalidated := win.counts(); invalidated == 0 || win.invalidated[invalidated-1] != image.Rect(0, 0, 960, 576) {
		t.Errorf("a font change invalidated %v, want the whole window", win.invalidated)
	}
	if _, err := b.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if b.Capabilities.CellPixels != image.Pt(12, 24) {
		t.Errorf("capabilities cell %v after the font change, want 12x24", b.Capabilities.CellPixels)
	}
	if err := b.Exit(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(term.out(), konst.LeaveScreen) {
		t.Errorf("exit did not leave the screen")
	}
}

func entered(t *testing.T) []string {
	var lines []string
	for _, tc := range []struct {
		name    string
		answers []string
		offer   offer
	}{
		{"windows-terminal", windowsTerminal, offer{}},
		{"kitty", kitty, offer{}},
		{"inbox-conpty", conPTY, offer{}},
		{"zed", zed, offer{zed: true}},
		{"silent", nil, offer{}},
		{"forced-sixel", windowsTerminal, offer{graphics: GraphicsSixel, forced: true}},
	} {
		term := newFake(tc.answers...)
		b, err := enter(term, term.tty, Options{}, tc.offer)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := b.Write([]byte("frame")); err != nil {
			t.Fatal(err)
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, tc.name+" "+strconv.Quote(term.out()))
	}
	return lines
}

func TestNonGDIEnterBytesUnchanged(t *testing.T) {
	fixture, err := os.ReadFile("testdata/entered.txt")
	if err != nil {
		t.Fatal(err)
	}
	want, got := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(fixture), "\r", "")), "\n"), entered(t)
	if len(got) != len(want) {
		t.Fatalf("%d terminals, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("bytes differ from HEAD:\n got %s\nwant %s", got[i], want[i])
		}
	}
}
